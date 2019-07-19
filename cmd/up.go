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

func cloneMissingRepos() {
	log.Info("☐ checking ~/.tb directory for missing git repos")
	// We need to clone every repo to resolve of all the references in the compose files to files in the repos.
	services := config.Services()

	for name, s := range services {
		if !s.IsGithubRepo {
			continue
		}

		path := fmt.Sprintf("%s/%s", config.TBRootPath(), name)
		if util.FileOrDirExists(path) {
			continue
		}

		log.Infof("\t☐ %s is missing. cloning git repo\n", name)
		err := git.Clone(name, config.TBRootPath())
		if err != nil {
			fatal.ExitErrf(err, "failed cloning git repo %s", name)
		}
		log.Infof("\t☑ finished cloning %s\n", name)
	}

	log.Info("☑ finished checking git repos")
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

func execDBPrepare(name string, isECR bool) {
	var composeName string
	var err error

	if isECR {
		composeName = name + "-ecr"
	} else {
		composeName = name
	}

	// TODO: Make a flag to turn this back on for people who need it - I don't think most people use this.
	// log.Infof("\t☐ resetting test database for %s.\n", name)
	// composeArgs := fmt.Sprintf("%s run --rm %s yarn db:prepare:test", composeFile, composeName)
	// err = util.Exec("docker-compose", strings.Fields(composeArgs)...)
	// if err != nil {
	// 	fatal.ExitErr(err, "failed running yarn db:prepare:test")
	// }
	// log.Infof("\t☑ finished resetting test database for %s.\n", name)

	log.Infof("\t☐ resetting development database for %s. this may take a long time.\n", name)
	composeArgs := fmt.Sprintf("%s run --rm %s yarn db:prepare", composeFile, composeName)
	err = util.Exec("docker-compose", strings.Fields(composeArgs)...)
	if err != nil {
		fatal.ExitErr(err, "failed running yarn db:prepare")
	}
	log.Infof("\t☑ finished resetting development database for %s.\n", name)
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
	fmt.Println()
}

func validatePlaylistName(playlistName string) {
	if len(playlistName) == 0 {
		// TODO: Color the commands bro
		fatal.Exit("playlist name cannot be blank. try running tb up --help")
	}
	names := config.GetPlaylist(playlistName)
	if len(names) == 0 {
		fatal.Exitf("playlist \"%s\" is empty or nonexistent.\ntry running tb up --tree to see all the available playlists.\n", playlistName)
	}
}

func toComposeNames(configs config.ServiceMap) []string {
	names := make([]string, 0)
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

func filterByNames(configs config.ServiceMap, names []string) config.ServiceMap {
	selected := make(config.ServiceMap)
	for _, name := range names {
		if _, ok := configs[name]; ok {
			selected[name] = configs[name]
		}
	}

	return selected
}

func selectServices() {
	if len(opts.cliServiceNames) > 0 && opts.playlistName != "" {
		fatal.Exit("you can only specify one of --playlist or --services.\nTry tb up --help for some examples.")
	}

	var names []string
	if opts.playlistName != "" {
		validatePlaylistName(opts.playlistName)
		names = config.GetPlaylist(opts.playlistName)
	} else if len(opts.cliServiceNames) > 0 {
		// TODO: be more strict about failing if any cliServicesName is invalid.
		names = opts.cliServiceNames
	} else {
		fatal.Exit("you must specify either --playlist or --services.\nTry tb up --help for some examples.")
	}

	selectedServices = filterByNames(config.Services(), names)
	if len(selectedServices) == 0 {
		fatal.Exit("you must specify at least one service from TouchBistro/tb/config.json.\nTry tb list --services to see all the available playlists.")
	}

	// TODO: Tell the user what services they are about to run.
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

		cloneMissingRepos()
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
			log.Info("☐ pulling the latest ecr images for selected services")
			for name, s := range selectedServices {
				if s.ECR {
					uri := config.ResolveEcrURI(name, s.ECRTag)

					log.Infof("\t☐ pulling image %s\n", uri)
					err := docker.Pull(uri)
					if err != nil {
						fatal.ExitErrf(err, "failed pulling docker image %s", uri)
					}
					log.Infof("\t☐ finished pulling image %s\n", uri)
				}

			}
			log.Info("☑ finished pulling ecr images for selected services")
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

		composeServiceNames := toComposeNames(selectedServices)

		dockerComposeBuild(composeServiceNames)

		if !opts.shouldSkipDBPrepare {
			log.Info("☐ performing database migrations and seeds")
			// TODO: Parallelize this shit
			for name, s := range selectedServices {
				if !s.Migrations {
					continue
				}
				execDBPrepare(name, s.ECR)
			}
			log.Info("☑ finished performing all migrations and seeds")
			fmt.Println()
		}

		dockerComposeUp(composeServiceNames)

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
