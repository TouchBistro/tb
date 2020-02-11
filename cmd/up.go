package cmd

import (
	"fmt"
	"os"
	"strings"

	"time"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/fatal"
	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/login"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type options struct {
	shouldSkipServicePreRun bool
	shouldSkipServerStart   bool
	shouldSkipGitPull       bool
	shouldSkipDockerPull    bool
	cliServiceNames         []string
	playlistName            string
}

var opts options

func performLoginStrategies(loginStrategies []login.LoginStrategy) {
	log.Infof("☐ Logging into services.")

	successCh := make(chan string)
	failedCh := make(chan error)

	for _, s := range loginStrategies {
		log.Infof("\t☐ Logging into %s", s.Name())
		go func(successCh chan string, failedCh chan error, s login.LoginStrategy) {
			err := s.Login()
			if err != nil {
				failedCh <- err
				return
			}
			successCh <- s.Name() + " login"
		}(successCh, failedCh, s)
	}

	util.SpinnerWait(successCh, failedCh, "\t☑ Finished %s\n", "Error while logging into services", len(loginStrategies))
	log.Info("☑ Finished logging into services")
}

func cleanupPrevDocker(services config.ServiceMap) {
	log.Debug("stopping compose services...")
	serviceNames := make([]string, 0)
	for name := range services {
		serviceNames = append(serviceNames, name)
	}
	err := docker.ComposeStop(serviceNames)
	if err != nil {
		fatal.ExitErr(err, "failed stopping containers and services")
	}
	err = docker.ComposeRm(serviceNames)
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

func dockerComposeBuild(services config.ServiceMap, composeFile string) {
	log.Info("☐ building images for non-remote services")

	var builder strings.Builder
	for name, s := range services {
		if !s.UseRemote() {
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

func dockerComposeUp(services config.ServiceMap, composeFile string) {
	serviceNames := config.ComposeNames(services)

	log.Info("☐ starting docker-compose up in detached mode")

	upArgs := fmt.Sprintf("%s up -d %s", composeFile, strings.Join(serviceNames, " "))
	err := util.Exec("compose-up", "docker-compose", strings.Fields(upArgs)...)
	if err != nil {
		fatal.ExitErr(err, "could not run docker-compose up")
	}

	log.Info("☑ finished starting docker-compose up in detached mode")
}

func selectServices() config.ServiceMap {
	if len(opts.cliServiceNames) > 0 && opts.playlistName != "" {
		fatal.Exit("you can only specify one of --playlist or --services.\nTry tb up --help for some examples.")
	}

	var names []string

	// parsing --playlist
	if opts.playlistName != "" {
		name := opts.playlistName
		if len(name) == 0 {
			fatal.Exit("playlist name cannot be blank. try running tb up --help")
		}
		var err error
		names, err = config.GetPlaylist(name, make(map[string]bool))
		if err != nil {
			fatal.ExitErr(err, "☒ failed resolving service playlist")
		}
		if len(names) == 0 {
			fatal.Exitf("playlist \"%s\" is empty or nonexistent.\ntry running tb list --playlists to see all the available playlists.\n", name)
		}
		// parsing --services
	} else if len(opts.cliServiceNames) > 0 {
		names = opts.cliServiceNames
	} else {
		fatal.Exit("you must specify either --playlist or --services.\ntry tb up --help for some examples.")
	}

	services := config.Services()
	selectedServices := make(config.ServiceMap, len(names))
	for _, name := range names {
		if _, ok := services[name]; !ok {
			fatal.Exitf("%s is not a tb service name.\n Try tb list to see all available servies.\n", name)
		}
		selectedServices[name] = services[name]
	}

	if len(selectedServices) == 0 {
		fatal.Exit("you must specify at least one service from TouchBistro/tb/config.json.\nTry tb list --services to see all the available playlists.")
	}

	return selectedServices
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
		if opts.shouldSkipServerStart {
			os.Setenv("START_SERVER", "false")
		} else {
			os.Setenv("START_SERVER", "true")
		}

		err := deps.Resolve(
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
		selectedServices := selectServices()
		composeNames := config.ComposeNames(selectedServices)
		log.Infof("running the following services: %s", strings.Join(composeNames, ", "))

		// We have to clone every possible repo instead of just selected services
		// Because otherwise docker-compose will complaing about missing build paths
		err := config.CloneMissingRepos(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos")
		}

		fmt.Println()

		loginStrategies, err := config.LoginStategies()
		if err != nil {
			fatal.ExitErr(err, "Failed to get login strategies")
		}

		if loginStrategies != nil {
			performLoginStrategies(loginStrategies)
			fmt.Println()
		}

		log.Infof("☐ Cleaning up previous Docker state.")
		successCh := make(chan string)
		failedCh := make(chan error)

		go func() {
			cleanupPrevDocker(selectedServices)
			successCh <- "Docker Cleanup"
		}()

		util.SpinnerWait(successCh, failedCh, "☑ Finished %s\n", "Error cleaning up previous Docker state", 1)

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
			for _, s := range selectedServices {
				if s.UseRemote() {
					uri := s.ImageURI()
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

			for _, repo := range config.Repos(selectedServices) {
				log.Infof("\t☐ pulling %s\n", repo)
				go func(successCh chan string, failedCh chan error, name, root string) {
					err := git.Pull(name, root)
					if err != nil {
						failedCh <- err
						return
					}
					successCh <- name
				}(successCh, failedCh, repo, config.ReposPath())
				count++
			}

			util.SpinnerWait(successCh, failedCh, "\t☑ finished pulling %s\n", "failed pulling git repo", count)

			log.Info("☑ finished pulling latest default git branch for selected services")
			fmt.Println()
		}

		composeFile := docker.ComposeFile()
		dockerComposeBuild(selectedServices, composeFile)
		fmt.Println()

		if !opts.shouldSkipServicePreRun {
			log.Info("☐ performing preRun step for services")
			successCh = make(chan string)
			failedCh = make(chan error)
			count := 0
			for name, s := range selectedServices {
				if s.PreRun == "" {
					continue
				}

				log.Infof("\t☐ running preRun command %s for %s. this may take a long time.\n", s.PreRun, name)
				composeArgs := fmt.Sprintf("%s run --rm %s %s", composeFile, config.ComposeName(name, s), s.PreRun)
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
			util.SpinnerWait(successCh, failedCh, "\t☑ finished running preRun command for %s.\n", "failed running preRun command", count)

			log.Info("☑ finished performing all preRun steps")
			fmt.Println()
		}

		dockerComposeUp(selectedServices, composeFile)
		fmt.Println()

		// Maybe we start this earlier and run compose build and preRun etc. in a separate goroutine so that people have a nicer output?
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
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipServerStart, "no-start-servers", false, "dont start servers with yarn start or yarn serve on container boot")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipServicePreRun, "no-service-prerun", false, "dont run preRun command for services")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDockerPull, "no-remote-pull", false, "dont get new remote images")
	upCmd.PersistentFlags().StringVarP(&opts.playlistName, "playlist", "p", "", "the name of a service playlist")
	upCmd.PersistentFlags().StringSliceVarP(&opts.cliServiceNames, "services", "s", []string{}, "comma separated list of services to start. eg --services postgres,localstack.")

	rootCmd.AddCommand(upCmd)
}
