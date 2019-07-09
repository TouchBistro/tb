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
	playlistName          string
}

var (
	composeFiles string
	opts         options
)

func initComposeFiles() {
	var err error
	composeFiles, err = docker.ComposeFiles()
	if err != nil {
		log.Fatal(err)
	}
}

func cloneMissingRepos() {
	// We need to clone every repo to resolve of all the references in the compose files to files in the repos.
	services := *config.All()
	log.Println("Checking repos...")
	for _, s := range services {
		path := fmt.Sprintf("./%s", s.Name)
		if !s.IsGithubRepo {
			continue
		}

		if util.FileOrDirExists(path) {
			continue
		}

		log.Printf("%s is missing. cloning...\n", s.Name)
		err := git.Clone(s.Name)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("...done")
}

func initECRLogin() {
	log.Println("Logging into ECR...")
	err := docker.ECRLogin()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("...done")
}

func initDockerStop() {
	var err error

	log.Println("stopping running containers...")
	err = docker.StopAllContainers()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("...done")

	log.Println("stopping compose services...")
	stopArgs := fmt.Sprintf("%s stop", composeFiles)
	_, err = util.Exec("docker-compose", strings.Fields(stopArgs)...)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("...done")

	log.Println("removing any running containers...")
	err = docker.RmContainers()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("...done")
}

func pullTBBaseImages() {
	log.Println("Pulling the latest touchbistro base images...")
	for _, b := range config.BaseImages() {
		err := docker.Pull(b)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("...done")
}

func execDBPrepare(s config.Service) {
	var composeName string
	var err error

	if s.ECR {
		composeName = s.Name + "-ecr"
	} else {
		composeName = s.Name
	}

	log.Println("Resetting test database...")
	composeArgs := fmt.Sprintf("%s run --rm %s yarn db:prepare:test", composeFiles, composeName)
	_, err = util.Exec("docker-compose", strings.Fields(composeArgs)...)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Resetting development database...")
	composeArgs = fmt.Sprintf("%s run --rm %s yarn db:prepare", composeFiles, composeName)
	_, err = util.Exec("docker-compose", strings.Fields(composeArgs)...)
	if err != nil {
		log.Fatal(err)
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

	buildArgs := fmt.Sprintf("%s build --parallel %s", composeFiles, str.String())
	_, err := util.Exec("docker-compose", strings.Fields(buildArgs)...)
	if err != nil {
		log.Fatal(err)
	}
}

func dockerComposeUp(serviceNames []string) {
	var err error

	log.Println("Running docker-compose up...")
	if opts.shouldSkipServerStart {
		os.Setenv("START_SERVER", "false")
	} else {
		os.Setenv("START_SERVER", "true")
	}

	upArgs := fmt.Sprintf("%s up -d %s", composeFiles, strings.Join(serviceNames, " "))
	_, err = util.Exec("docker-compose", strings.Fields(upArgs)...)
	if err != nil {
		log.Fatal(err)
	}
}

func validatePlaylistName(playlistName string) {
	names := config.GetPlaylist(playlistName)
	if len(names) == 0 {
		log.Println("You must specify at least one service")
		os.Exit(1)
	}
}

func toComposeNames(configs []config.Service) []string {
	names := make([]string, len(configs))
	for _, s := range configs {
		var name string
		if s.ECR {
			name = s.Name + "-ecr"
		} else {
			name = s.Name
		}
		names = append(names, name)
	}
	return names
}

func filterByPlaylist(configs []config.Service, playlistName string) []config.Service {
	names := config.GetPlaylist(playlistName)

	playlistServices := make(map[string]bool)
	for _, name := range names {
		playlistServices[name] = true
	}

	selected := make([]config.Service, 0)
	for _, s := range configs {
		if _, ok := playlistServices[s.Name]; !ok {
			continue
		}

		selected = append(selected, s)
	}
	return selected
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Starts services defined in docker-compose.*.yml files",
	PreRun: func(cmd *cobra.Command, args []string) {
		log.Println("skip server option:", opts.shouldSkipServerStart)
		log.Println("skip db option:", opts.shouldSkipDBPrepare)
		log.Println("playlist option:", opts.playlistName)
		validatePlaylistName(opts.playlistName)

		err := deps.Resolve(
			deps.Brew,
			deps.Jq,
			deps.Aws,
			deps.Lazydocker,
			deps.Node,
			deps.Yarn,
			deps.Docker,
		)
		if err != nil {
			log.Fatal(err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		cloneMissingRepos()
		initECRLogin()
		initComposeFiles()
		initDockerStop()

		if !opts.shouldSkipDockerPull {
			pullTBBaseImages()
		}

		selectedServices := filterByPlaylist(*config.All(), opts.playlistName)

		if !opts.shouldSkipDockerPull {
			log.Println("Pulling the latest ecr images for selected services...")
			for _, s := range selectedServices {
				if opts.shouldSkipDockerPull {
					if s.ECR {
						err := docker.Pull(s.ImageURI)
						if err != nil {
							log.Fatal(err)
						}
					}
				}

			}
			log.Println("...done")
		}

		if !opts.shouldSkipGitPull {
			// Pull latest github repos
			log.Println("Pulling the latest git branch for selected services...")
			// TODO: Parallelize this shit
			for _, s := range selectedServices {
				if s.IsGithubRepo && !s.ECR {
					err := git.Pull(s.Name)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
			log.Println("...done")
		}

		composeServiceNames := toComposeNames(selectedServices)
		dockerComposeBuild(composeServiceNames)

		if !opts.shouldSkipDBPrepare {
			log.Println("Performing database migrations and seeds...")
			// TODO: Parallelize this shit
			for _, s := range selectedServices {
				if !s.Migrations {
					continue
				}
				// TODO: merge compose files into one again
				execDBPrepare(s)
			}
			log.Println("...done")
		}

		// TODO: merge compose files into one again
		dockerComposeUp(composeServiceNames)

		// Maybe we start this earlier and run compose build and migrations etc. in a separate goroutine so that people have a nicer output?
		_, err = util.Exec("lazydocker")
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipServerStart, "no-start-servers", false, "dont start servers with yarn start or yarn serve on container boot")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDBPrepare, "no-db-reset", false, "dont reset databases with yarn db:prepare")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
	upCmd.PersistentFlags().BoolVar(&opts.shouldSkipDockerPull, "no-ecr-pull", false, "dont get new ecr images")
	upCmd.PersistentFlags().StringVar(&opts.playlistName, "playlist", "custom", "the name of a service playlist")

	RootCmd.AddCommand(upCmd)
}
