package cmd

import (
	"fmt"
	"strings"

	"time"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/fatal"
	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/npm"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type options struct {
	shouldSkipDBPrepare   bool
	shouldSkipServerStart []string
	shouldSkipGitPull     bool
	shouldSkipDockerPull  bool
	cliServiceNames       []string
	playlistName          string
}

var (
	composeFile      string
	selectedServices config.ServiceMap
	opts             options
)

func attemptNPMLogin() {
	err := npm.Login()
	if err != nil {
		fatal.ExitErr(err, "☒ failed logging into NPM")
	}
}

func attemptECRLogin() {
	err := docker.ECRLogin()
	if err != nil {
		fatal.ExitErr(err, "failed logging into ECR")
	}
}

func cleanupPrevDocker() {
	err := docker.StopContainersAndServices()
	if err != nil {
		fatal.ExitErr(err, "failed stopping containers and services")
	}
	err = docker.RmContainers()
	if err != nil {
		fatal.ExitErr(err, "failed removing containers")
	}
}

func pullTBBaseImages() {
	log.Info("☐ pulling latest touchbistro base images")

	successCh := make(chan string)
	failedCh := make(chan error)

	for _, b := range config.BaseImages() {
		log.Infof("\t☐ pulling %s\n", b)
		go func(successCh chan string, failedCh chan error, b string) {
			err := docker.Pull(b)
			if err != nil {
				failedCh <- err
				return
			}
			successCh <- b
		}(successCh, failedCh, b)
	}

	util.SpinnerWait(successCh, failedCh, "\t☑ finished pulling %s\n", "failed pulling docker image", len(config.BaseImages()))

	log.Info("☑ finished pulling latest touchbistro base images")
}

func dockerComposeBuild() {
	log.Info("☐ building images for non-ecr / remote services")

	var builder strings.Builder
	for name, s := range selectedServices {
		if !s.ECR && s.DockerhubImage == "" {
			builder.WriteString(name)
			builder.WriteString(" ")
		}
	}

	str := builder.String()

	if str == "" {
		log.Info("☑ no services to build")
		return
	}

	buildArgs := fmt.Sprintf("%s build --parallel %s", composeFile, str)
	err := util.Exec("compose-build", "docker-compose", strings.Fields(buildArgs)...)
	if err != nil {
		fatal.ExitErr(err, "could not build docker-compose services")
	}

	log.Info("☑ finished docker compose build.")
	fmt.Println()
}

func dockerComposeUp() {
	serviceNames := config.ComposeNames(selectedServices)

	log.Info("☐ starting docker-compose up in detached mode")

	upArgs := fmt.Sprintf("%s up -d %s", composeFile, strings.Join(serviceNames, " "))
	err := util.Exec("compose-up", "docker-compose", strings.Fields(upArgs)...)
	if err != nil {
		fatal.ExitErr(err, "could not run docker-compose up")
	}

	stopArgs := fmt.Sprintf("%s stop %s", composeFile, strings.Join(opts.shouldSkipServerStart, " "))
	err = util.Exec("compose-up", "docker-compose", strings.Fields(stopArgs)...)
	if err != nil {
		fatal.ExitErr(err, "could not stop skipped services")
	}

	log.Info("☑ finished starting docker-compose up in detached mode")
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Starts services from a playlist name or as a comma separated list of services",
	Long: `Starts services from a playlist name or as a comma separated list of services.

Examples:
- run the services defined under the "core" key in playlists.yml
	tb up --playlist core

- run only postgres and localstack
	tb up --services postgres,localstack`,
	Args: cobra.NoArgs,
	PreRun: func(cmd *cobra.Command, args []string) {
		var err error

		if len(opts.cliServiceNames) > 0 && opts.playlistName != "" {
			fatal.Exit("you can only specify one of --playlist or --services.\nTry tb up --help for some examples.")
		}

		selectedServices, err = config.SelectServices(opts.cliServiceNames, opts.playlistName)
		if err != nil {
			fatal.ExitErr(err, "failed resolving selected services")
		}

		composeNames := config.ComposeNames(selectedServices)
		log.Infof("running the following services: %s", strings.Join(composeNames, ", "))

		//check -n flag for valid service names

		err = deps.Resolve(
			deps.Brew,
			deps.Aws,
			deps.Lazydocker,
			deps.Node,
			deps.Yarn,
		)
		if err != nil {
			fatal.ExitErr(err, "could not resolve dependencies")
		}
		fmt.Println()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		composeFile = docker.ComposeFile()

		// We have to clone every possible repo instead of just selected services
		// Because otherwise docker-compose will complaing about missing build paths
		err = config.CloneMissingRepos(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos")
		}

		fmt.Println()

		successCh := make(chan string)
		failedCh := make(chan error)

		log.Infof("Logging into NPM, ECR, and cleaning up up previous Docker state.")

		go func() {
			attemptNPMLogin()
			successCh <- "NPM Login"
		}()
		go func() {
			attemptECRLogin()
			successCh <- "ECR Login"
		}()
		go func() {
			cleanupPrevDocker()
			successCh <- "Docker Cleanup"
		}()

		util.SpinnerWait(successCh, failedCh, "\t☑ Finished %s\n", "Error while setting up", 3)
		log.Info("☑ Finished setup tasks")
		fmt.Println()

		// check for docker disk usage after cleanup
		full, usage, err := docker.CheckDockerDiskUsage()
		if err != nil {
			fatal.ExitErr(err, "☒ failed checking docker status")
		}
		log.Infof("Current docker disk usage: %.2fGB", float64(usage)/1024/1024/1024)

		if full {
			if util.Prompt("Your free disk space is running out, would you like to cleanup? y/n >") {
				// Images are all we really care about as far as space cleaning
				log.Infoln("Removing images...")
				go func(successCh chan string, failedCh chan error) {
					pruned, err := docker.PruneImages()
					if err != nil {
						failedCh <- err
						return
					}
					cleaned := fmt.Sprintf("%.2f", ((float64(pruned)/1024)/1024)/1024)
					successCh <- cleaned
				}(successCh, failedCh)
				util.SpinnerWait(successCh, failedCh, "\t☑ finished pruning docker images, reclaimed %sGB\n", "failed pruning docker images", 1)
			} else {
				log.Infoln("Continuing, but unexpected behavior is possible if docker usage isn't cleaned.")
			}
		}

		if !opts.shouldSkipDockerPull {
			pullTBBaseImages()
			fmt.Println()
		}

		if !opts.shouldSkipDockerPull {
			log.Info("☐ pulling the latest docker images for selected services")
			successCh = make(chan string)
			failedCh = make(chan error)
			count := 0
			for name, s := range selectedServices {
				if s.ECR || s.DockerhubImage != "" {
					var uri string
					if s.ECR {
						uri = config.ResolveEcrURI(name, s.ECRTag)
					} else {
						uri = s.DockerhubImage
					}

					log.Infof("\t☐ pulling image %s\n", uri)
					go func() {
						err := docker.Pull(uri)
						if err != nil {
							failedCh <- err
							return
						}
						successCh <- uri
					}()
					count++
				}
			}

			util.SpinnerWait(successCh, failedCh, "\t☑ finished pulling %s\n", "failed pulling docker image", count)
			log.Info("☑ finished pulling docker images for selected services")
			fmt.Println()
		}

		if !opts.shouldSkipGitPull {
			// Pull latest github repos
			log.Info("☐ pulling the latest default git branch for selected services")
			successCh = make(chan string)
			failedCh = make(chan error)
			count := 0

			for _, repoName := range config.RepoNames(selectedServices) {
				log.Infof("\t☐ pulling %s\n", repoName)
				go func(successCh chan string, failedCh chan error, name, root string) {
					err := git.Pull(name, root)
					if err != nil {
						failedCh <- err
						return
					}
					successCh <- name
				}(successCh, failedCh, repoName, config.TBRootPath())
				count++
			}

			util.SpinnerWait(successCh, failedCh, "\t☑ finished pulling %s\n", "failed pulling git repo", count)

			log.Info("☑ finished pulling latest default git branch for selected services")
			fmt.Println()
		}

		dockerComposeBuild()
		fmt.Println()

		if !opts.shouldSkipDBPrepare {
			log.Info("☐ performing database migrations and seeds")
			successCh = make(chan string)
			failedCh = make(chan error)
			count := 0
			for name, s := range selectedServices {
				if !s.Migrations {
					continue
				}

				log.Infof("\t☐ resetting development database for %s. this may take a long time.\n", name)
				composeArgs := fmt.Sprintf("%s run --rm %s yarn db:prepare", composeFile, config.ComposeName(name, s))
				go func(successCh chan string, failedCh chan error, name string, args ...string) {
					err := util.Exec(name, "docker-compose", args...)
					if err != nil {
						failedCh <- err
						return
					}
					successCh <- name
				}(successCh, failedCh, name, strings.Fields(composeArgs)...)
				count++
				// We need to wait a bit in between launching goroutines or else they all create seperated docker-compose environments
				// Any ideas better than a sleep hack are appreciated
				time.Sleep(time.Second)
			}
			util.SpinnerWait(successCh, failedCh, "\t☑ finished resetting development database for %s.\n", "failed running yarn db:prepare", count)

			log.Info("☑ finished performing all migrations and seeds")
			fmt.Println()
		}

		dockerComposeUp()
		fmt.Println()

		// Maybe we start this earlier and run compose build and migrations etc. in a separate goroutine so that people have a nicer output?
		log.Info("☐ Starting lazydocker")
		err = util.Exec("lazydocker", "lazydocker")
		if err != nil {
			fatal.ExitErr(err, "failed running lazydocker")
		}

		log.Info("☑ finished with lazydocker.")

		fmt.Println()
		log.Info("🔈 the containers are still running in the background. If you want to terminate them, run tb down")
	},
}

func init() {
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDBPrepare, "no-db-reset", false, "dont reset databases with yarn db:prepare")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDockerPull, "no-ecr-pull", false, "dont get new ecr images")
	upCmd.PersistentFlags().StringVarP(&opts.playlistName, "playlist", "p", "", "the name of a service playlist")
	upCmd.PersistentFlags().StringSliceVarP(&opts.cliServiceNames, "services", "s", []string{}, "comma separated list of services to start. eg --services postgres,localstack.")
	upCmd.PersistentFlags().StringSliceVarP(&opts.shouldSkipServerStart, "no-start-servers", "n", []string{}, "comma seperated list of services to avoid starting after creation")

	rootCmd.AddCommand(upCmd)
}
