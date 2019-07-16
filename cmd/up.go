package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/git"
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
	selectedServices map[string]config.Service
	opts             options
)

func cloneMissingRepos() {
	// We need to clone every repo to resolve of all the references in the compose files to files in the repos.
	services := config.Services()
	log.Info("Checking repos...")
	for name, s := range services {
		if !s.IsGithubRepo {
			continue
		}

		path := fmt.Sprintf("%s/%s", config.TBRootPath(), name)
		if util.FileOrDirExists(path) {
			continue
		}

		log.Infof("%s is missing. cloning...\n", name)
		err := git.Clone(name, config.TBRootPath())
		if err != nil {
			message := fmt.Sprintf("failed cloning repo %s", name)
			util.FatalErr(message, err)
		}
	}
	log.Info("...done")
}

func initECRLogin() {
	log.Info("Logging into ECR...")
	err := docker.ECRLogin()
	if err != nil {
		util.FatalErr("Failled logging into ECR", err)
	}
	log.Info("...done")
}

func initDockerStop() {
	var err error

	log.Info("stopping any running containers or services...")
	err = docker.StopContainersAndServices()
	if err != nil {
		util.FatalErr("failed stopping containers and services", err)
	}

	log.Info("removing stopped containers...")
	err = docker.RmContainers()
	if err != nil {
		util.FatalErr("failed removing containers", err)
	}
	log.Info("...done")
}

func pullTBBaseImages() {
	log.Info("Pulling the latest touchbistro base images...")
	for _, b := range config.BaseImages() {
		err := docker.Pull(b)
		if err != nil {
			message := fmt.Sprintf("Failed pulling docker image: %s", b)
			util.FatalErr(message, err)
		}
	}
	log.Info("...done")
}

func execDBPrepare(name string, isECR bool) {
	var composeName string
	var err error

	if isECR {
		composeName = name + "-ecr"
	} else {
		composeName = name
	}

	log.Debugf("Resetting test database...")
	composeArgs := fmt.Sprintf("%s run --rm %s yarn db:prepare:test", composeFile, composeName)
	err = util.Exec("docker-compose", strings.Fields(composeArgs)...)
	if err != nil {
		util.FatalErr("Failed running yarn db:prepare:test", err)
	}

	log.Debugf("Resetting development database...")
	composeArgs = fmt.Sprintf("%s run --rm %s yarn db:prepare", composeFile, composeName)
	err = util.Exec("docker-compose", strings.Fields(composeArgs)...)
	if err != nil {
		util.FatalErr("Failed running yarn db:prepare", err)
	}
}

func dockerComposeBuild(serviceNames []string) {
	var str strings.Builder
	for _, s := range serviceNames {
		if strings.HasSuffix(s, "-ecr") {
			continue
		}
		str.WriteString(s)
		str.WriteString(" ")
	}

	buildArgs := fmt.Sprintf("%s build --parallel %s", composeFile, str.String())
	err := util.Exec("docker-compose", strings.Fields(buildArgs)...)
	if err != nil {
		util.FatalErr("Could not build docker-compose services", err)
	}
}

func dockerComposeUp(serviceNames []string) {
	var err error
	log.Info("Starting docker-compose up in detached mode...")

	upArgs := fmt.Sprintf("%s up -d %s", composeFile, strings.Join(serviceNames, " "))
	err = util.Exec("docker-compose", strings.Fields(upArgs)...)
	if err != nil {
		util.FatalErr("Could not docker-compose up", err)
	}
}

func validatePlaylistName(playlistName string) {
	if len(playlistName) == 0 {
		util.Fatal("Playlist name cannot be blank")
	}
	names := config.GetPlaylist(playlistName)
	if len(names) == 0 {
		message := fmt.Sprintf("You must specify at least one service in playlist %s\n", playlistName)
		util.Fatal(message)
	}
}

func toComposeNames(configs map[string]config.Service) []string {
	names := make([]string, len(configs))
	for name, s := range configs {
		var composeName string
		if s.ECR {
			composeName = name + "-ecr"
		} else {
			composeName = name
		}
		names = append(names, composeName)
	}
	return names
}

func filterByNames(configs map[string]config.Service, names []string) map[string]config.Service {
	selected := make(map[string]config.Service)
	for _, name := range names {
		if _, ok := configs[name]; ok {
			selected[name] = configs[name]
		}
	}

	return selected
}

func initSelectedServices() {
	if len(opts.cliServiceNames) > 0 && opts.playlistName != "" {
		util.Fatal("can only specify one of --playlist or --services")
	}

	var names []string
	if opts.playlistName != "" {
		validatePlaylistName(opts.playlistName)
		names = config.GetPlaylist(opts.playlistName)
	} else if len(opts.cliServiceNames) > 0 {
		// TODO: be more strict about failing if any cliServicesName is invalid.
		names = opts.cliServiceNames
	} else {
		util.Fatal("must specify either --playlist or --services")
	}

	selectedServices = filterByNames(config.Services(), names)
	if len(selectedServices) == 0 {
		util.Fatal("You must specify at least one service from TouchBistro/tb/config.json")
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
	tb up -s postgres,localstack`,
	Args: cobra.NoArgs,
	PreRun: func(cmd *cobra.Command, args []string) {
		if opts.shouldSkipServerStart {
			os.Setenv("START_SERVER", "false")
		} else {
			os.Setenv("START_SERVER", "true")
		}

		initSelectedServices()

		err := deps.Resolve(
			deps.Brew,
			deps.Aws,
			deps.Lazydocker,
			deps.Node,
			deps.Yarn,
			deps.Docker,
		)
		if err != nil {
			util.FatalErr("Could not resolve dependencies", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		composeFile = docker.ComposeFile()

		cloneMissingRepos()
		initECRLogin()
		initDockerStop()

		if !opts.shouldSkipDockerPull {
			pullTBBaseImages()
		}

		if !opts.shouldSkipDockerPull {
			log.Info("Pulling the latest ecr images for selected services...")
			for _, s := range selectedServices {
				if opts.shouldSkipDockerPull {
					if s.ECR {
						err := docker.Pull(s.ImageURI)
						if err != nil {
							util.FatalErr(fmt.Sprintf("Failed pulling docker image %s", s.ImageURI), err)
						}
					}
				}

			}
			log.Info("...done")
		}

		if !opts.shouldSkipGitPull {
			// Pull latest github repos
			log.Info("Pulling the latest git branch for selected services...")
			// TODO: Parallelize this shit
			for name, s := range selectedServices {
				if s.IsGithubRepo && !s.ECR {
					err := git.Pull(name)
					if err != nil {
						util.FatalErr(fmt.Sprintf("Failed pulling git repo %s", s.ImageURI), err)
					}
				}
			}
			log.Info("...done")
		}

		composeServiceNames := toComposeNames(selectedServices)
		dockerComposeBuild(composeServiceNames)

		if !opts.shouldSkipDBPrepare {
			log.Info("Performing database migrations and seeds...")
			// TODO: Parallelize this shit
			for name, s := range selectedServices {
				if !s.Migrations {
					continue
				}
				// TODO: merge compose files into one again
				execDBPrepare(name, s.ECR)
			}
			log.Info("...done")
		}

		// TODO: merge compose files into one again
		dockerComposeUp(composeServiceNames)

		// Maybe we start this earlier and run compose build and migrations etc. in a separate goroutine so that people have a nicer output?
		err = util.Exec("lazydocker")
		if err != nil {
			util.FatalErr("Failed running lazydocker", err)
		}
	},
}

func init() {
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipServerStart, "no-start-servers", false, "dont start servers with yarn start or yarn serve on container boot")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDBPrepare, "no-db-reset", false, "dont reset databases with yarn db:prepare")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDockerPull, "no-ecr-pull", false, "dont get new ecr images")
	upCmd.PersistentFlags().StringVar(&opts.playlistName, "playlist", "", "the name of a service playlist")
	upCmd.PersistentFlags().StringSliceVarP(&opts.cliServiceNames, "services", "s", []string{}, "comma separated list of services to start. eg --services postgres,localstack.")

	rootCmd.AddCommand(upCmd)
}
