package cmd

import (
	"fmt"
	"os"
	"strings"

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
	shouldSkipServerStart bool
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
	log.Info("☐ logging into NPM")

	err := npm.Login()
	if err != nil {
		fatal.ExitErr(err, "☒ failed logging into NPM")
	}

	log.Info("☑ finished logging into NPM")
}

func attemptECRLogin() {
	log.Info("☐ logging into ECR")

	err := docker.ECRLogin()
	if err != nil {
		fatal.ExitErr(err, "failed logging into ECR")
	}

	log.Info("☑ finished logging into ECR")
}

func cleanupPrevDocker() {
	log.Info("☐ removing previous docker resources. this may take a long time")

	log.Info("\t☐ stopping any running containers or services")
	err := docker.StopContainersAndServices()
	if err != nil {
		fatal.ExitErr(err, "failed stopping containers and services")
	}
	log.Info("\t☑ finished stopping running containers and services")

	log.Info("\t☐ removing stopped containers")
	err = docker.RmContainers()
	if err != nil {
		fatal.ExitErr(err, "failed removing containers")
	}
	log.Info("\t☑ finished removing stopped containers")

	log.Info("☑ finished cleaning docker resources")
}

func pullTBBaseImages() {
	log.Info("☐ pulling latest touchbistro base images")

	for _, b := range config.BaseImages() {
		log.Infof("\t☐ pulling %s\n", b)
		err := docker.Pull(b)
		if err != nil {
			fatal.ExitErrf(err, "failed pulling docker image: %s", b)
		}
		log.Infof("\t☑ finished pulling %s\n", b)
	}

	log.Info("☑ finished pulling latest touchbistro base images")
}

func dockerComposeBuild(serviceNames []string) {
	log.Info("☐ building images for non-ecr / remote services")

	var builder strings.Builder
	for _, s := range serviceNames {
		if strings.HasSuffix(s, "-ecr") {
			continue
		}
		builder.WriteString(s)
		builder.WriteString(" ")
	}

	str := builder.String()
	if str == "" {
		log.Info("☑ no services to build")
		return
	}

	buildArgs := fmt.Sprintf("%s build --parallel %s", composeFile, str)
	err := util.Exec("docker-compose", strings.Fields(buildArgs)...)
	if err != nil {
		fatal.ExitErr(err, "could not build docker-compose services")
	}

	log.Info("☑ finished docker compose build.")
	fmt.Println()
}

func dockerComposeUp(serviceNames []string) {
	log.Info("☐ starting docker-compose up in detached mode")

	upArgs := fmt.Sprintf("%s up -d %s", composeFile, strings.Join(serviceNames, " "))
	err := util.Exec("docker-compose", strings.Fields(upArgs)...)
	if err != nil {
		fatal.ExitErr(err, "could not run docker-compose up")
	}

	log.Info("☑ finished starting docker-compose up in detached mode")
}

func selectServices() {
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

		names = config.GetPlaylist(name)
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
	selectedServices = make(config.ServiceMap, len(names))
	for _, name := range names {
		if _, ok := services[name]; !ok {
			fatal.Exitf("%s is not a tb service name.\n Try tb list to see all available servies.\n", name)
		}
		selectedServices[name] = services[name]
	}

	if len(selectedServices) == 0 {
		fatal.Exit("you must specify at least one service from TouchBistro/tb/config.json.\nTry tb list --services to see all the available playlists.")
	}
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

		selectServices()

		composeNames := config.ComposeNames(selectedServices)
		log.Infof("running the following services: %s", strings.Join(composeNames, ", "))

		err := deps.Resolve(
			deps.Brew,
			deps.Aws,
			deps.Lazydocker,
			deps.Node,
			deps.Yarn,
			deps.Docker,
		)
		if err != nil {
			fatal.ExitErr(err, "could not resolve dependencies")
		}
		fmt.Println()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		composeFile = docker.ComposeFile()

		err = config.Clone(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos")
		}

		fmt.Println()

		attemptNPMLogin()
		fmt.Println()

		attemptECRLogin()
		fmt.Println()

		cleanupPrevDocker()
		fmt.Println()

		if !opts.shouldSkipDockerPull {
			pullTBBaseImages()
			fmt.Println()
		}

		if !opts.shouldSkipDockerPull {
			log.Info("☐ pulling the latest docker images for selected services")
			for name, s := range selectedServices {
				if s.IsGithubRepo && !s.ECR {
					continue
				}

				var uri string
				if s.ECR {
					uri = config.ResolveEcrURI(name, s.ECRTag)
				} else {
					uri = s.ImageURI
				}

				log.Infof("\t☐ pulling image %s\n", uri)
				err := docker.Pull(uri)
				if err != nil {
					fatal.ExitErrf(err, "failed pulling docker image %s", uri)
				}
				log.Infof("\t☐ finished pulling image %s\n", uri)
			}
			log.Info("☑ finished pulling docker images for selected services")
			fmt.Println()
		}

		if !opts.shouldSkipGitPull {
			// Pull latest github repos
			log.Info("☐ pulling the latest default git branch for selected services")
			// TODO: Parallelize this shit
			for name, s := range selectedServices {
				if s.IsGithubRepo {
					log.Infof("\t☐ pulling %s\n", name)
					err := git.Pull(name, config.TBRootPath())
					if err != nil {
						fatal.ExitErrf(err, "failed pulling git repo %s", name)
					}
					log.Infof("\t☐ finished pulling %s\n", name)
				}
			}
			log.Info("☑ finished pulling latest default git branch for selected services")
			fmt.Println()
		}

		composeServiceNames := config.ComposeNames(selectedServices)

		dockerComposeBuild(composeServiceNames)
		fmt.Println()

		if !opts.shouldSkipDBPrepare {
			log.Info("☐ performing database migrations and seeds")
			// TODO: Parallelize this shit
			for name, s := range selectedServices {
				if !s.Migrations {
					continue
				}

				var composeName string
				if s.ECR {
					composeName = name + "-ecr"
				} else {
					composeName = name
				}

				log.Infof("\t☐ resetting development database for %s. this may take a long time.\n", name)
				composeArgs := fmt.Sprintf("%s run --rm %s yarn db:prepare", composeFile, composeName)
				err := util.Exec("docker-compose", strings.Fields(composeArgs)...)
				if err != nil {
					fatal.ExitErrf(err, "failed running yarn db:prepare for service %s", name)
				}

				log.Infof("\t☑ finished resetting development database for %s.\n", name)
			}
			log.Info("☑ finished performing all migrations and seeds")
			fmt.Println()
		}

		dockerComposeUp(composeServiceNames)
		fmt.Println()

		// Maybe we start this earlier and run compose build and migrations etc. in a separate goroutine so that people have a nicer output?
		log.Info("☐ Starting lazydocker")
		err = util.Exec("lazydocker")
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
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDBPrepare, "no-db-reset", false, "dont reset databases with yarn db:prepare")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDockerPull, "no-ecr-pull", false, "dont get new ecr images")
	upCmd.PersistentFlags().StringVarP(&opts.playlistName, "playlist", "p", "", "the name of a service playlist")
	upCmd.PersistentFlags().StringSliceVarP(&opts.cliServiceNames, "services", "s", []string{}, "comma separated list of services to start. eg --services postgres,localstack.")

	rootCmd.AddCommand(upCmd)
}
