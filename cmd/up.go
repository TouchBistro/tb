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
		// TODO: Only resolve deps needed for this command.
		err := deps.Resolve()
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
		// ECR Login
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

		// Stop running docker containers
		err = docker.StopAllContainers()
		if err != nil {
			log.Fatal(err)
		}

		// Stop docker-compose services
		composeFiles, err := docker.ComposeFiles()
		if err != nil {
			log.Fatal(err)
		}
		stopArgs := fmt.Sprintf("%s stop", composeFiles)
		_, err = util.Exec("docker-compose", strings.Fields(stopArgs)...)
		if err != nil {
			log.Fatal(err)
		}

		// Remove running docker containers
		err = docker.RmContainers()
		if err != nil {
			log.Fatal(err)
		}

		// Pull latest tb images
		log.Println("Pulling the latest touchbistro base images...")
		for _, b := range config.BaseImages() {
			err := docker.Pull(b)
			if err != nil {
				log.Fatal(err)
			}
		}
		log.Println("done...")

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

		// Pull Latest ECR images
		log.Println("Pulling the latest ecr images...")
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
		log.Println("Pulling the latest git branch...")
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

		// TODO: Launch lazydocker instead.
		// Running up without detaching to see all logs in a single stream.
		log.Println("Running docker-compose up...")
		upArgs := fmt.Sprintf("%s up --abort-on-container-exit %s", composeFiles, strings.Join(composeServiceNames, " "))
		_, err = util.Exec("docker-compose", strings.Fields(upArgs)...)
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
