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

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Starts services defined in docker-compose.*.yml files",
	PreRun: func(cmd *cobra.Command, args []string) {
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

		// Clone all Repos.
		// We need this because of all the references in the compose files to files in the repos.
		services := *config.All()
		log.Println("Checking repos...")
		for _, s := range services {
			path := fmt.Sprintf("./%s", s.Name)
			if !s.IsGithubRepo {
				continue
			}

			if !util.FileOrDirExists(path) {
				log.Printf("%s is missing. cloning...\n", s.Name)
				err = git.Clone(s.Name)
			}
			if err != nil {
				log.Fatal(err)
			}
		}
		log.Println("...done")

		// ECR Login
		log.Println("Logging into ECR...")
		err = docker.ECRLogin()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("...done")
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		noStartServers, _ := cmd.PersistentFlags().GetBool("no-start-servers")
		log.Println("starting server option enabled", noStartServers)
		if noStartServers {
			os.Setenv("START_SERVER", "false")
		} else {
			os.Setenv("START_SERVER", "true")
		}

		log.Println("stopping running containers...")
		err = docker.StopAllContainers()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("...done")

		log.Println("stopping compose services...")
		err = docker.ComposeStop()

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

		// Pull latest tb images
		log.Println("Pulling the latest touchbistro base images...")
		for _, b := range config.BaseImages() {
			err := docker.Pull(b)
			if err != nil {
				log.Fatal(err)
			}
		}
		log.Println("...done")

		selectedServices := make([]config.Service, 0)
		composeServiceNames := make([]string, 0)

		for _, s := range *config.All() {
			if s.Name != "core-backend" {
				continue
			}

			if s.ECR {
				composeServiceNames = append(composeServiceNames, s.Name+"-ecr")
			} else {
				composeServiceNames = append(composeServiceNames, s.Name)
			}

			selectedServices = append(selectedServices, s)
		}

		for _, s := range selectedServices {
			log.Println("Selected service: ", s)
		}

		// Pull Latest ECR images
		log.Println("Pulling the latest ecr images for selected services...")
		for _, s := range selectedServices {
			if s.ECR {
				err := docker.Pull(s.ImageURI)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
		log.Println("...done")

		// Pull latest github repos
		log.Println("Pulling the latest git branch for selected services...")
		for _, s := range selectedServices {
			if s.IsGithubRepo && !s.ECR {
				err := git.Pull(s.Name)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
		log.Println("...done")

		// Start building enabled services
		composeFiles, err := docker.ComposeFiles()
		if err != nil {
			log.Fatal(err)
		}

		buildArgs := fmt.Sprintf("%s build --parallel %s", composeFiles, strings.Join(composeServiceNames, " "))
		_, err = util.Exec("docker-compose", strings.Fields(buildArgs)...)
		if err != nil {
			log.Fatal(err)
		}

		skipDBReset, _ := cmd.PersistentFlags().GetBool("no-db-reset")
		if !skipDBReset {
			log.Println("Performing database migrations and seeds...")
			for _, s := range selectedServices {
				if !s.Migrations {
					continue
				}

				var composeName string
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
			log.Println("...done")
		}

		log.Println("Running docker-compose up...")
		upArgs := fmt.Sprintf("%s up -d %s", composeFiles, strings.Join(composeServiceNames, " "))
		_, err = util.Exec("docker-compose", strings.Fields(upArgs)...)
		if err != nil {
			log.Fatal(err)
		}

		_, err = util.Exec("lazydocker")
		if err != nil {
			log.Fatal(err)
		}

	},
}

func init() {
	upCmd.PersistentFlags().Bool("no-start-servers", false, "dont start servers with yarn start or yarn serve on container boot")
	upCmd.PersistentFlags().Bool("no-db-reset", false, "dont reset databases with yarn db:prepare")

	RootCmd.AddCommand(upCmd)
}
