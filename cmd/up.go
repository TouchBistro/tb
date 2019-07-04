package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/git"
	"github.com/TouchBistro/tb/util"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Starts services defined in docker-compose.*.yml files",
	PreRun: func(cmd *cobra.Command, args []string) {
		// TODO: Only get what you need for this
		err := deps.Resolve()
		if err != nil {
			log.Fatal(err)
		}

		// Clone all Repos. We need this because of all the references in the compose files.
		services := *config.All()
		log.Println("Checking repos...")
		for _, s := range services {
			path := fmt.Sprintf("./%s", s.Name)
			if !s.IsGithubRepo {
				continue
			}

			if !util.FileOrDirExists(path) {
				fmt.Printf("%s is missing. cloning...\n", s.Name)
				err = git.Clone(s.Name)
			}
			if err != nil {
				log.Fatal(err)
			}
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: WACK ZONE: Think about ruby, and node 12
		fmt.Println("Pulling the latest touchbistro base images...")

		for _, b := range config.BaseImages() {
			err := docker.Pull(b)
			if err != nil {
				log.Fatal(err)
			}
		}
		fmt.Println("done")

		// TODO: Use a flag or config file or services.txt whatever, only grab what we need
		services := *config.All()

		// Clone services
		for _, s := range services {
			path := fmt.Sprintf("./%s", s.Name)
			if s.IsGithubRepo { //
				var err error
				if !util.FileOrDirExists(path) {
					fmt.Printf("%s is missing. cloning...\n", s.Name)
					err = git.Clone(s.Name)
				} else {
					// We probably want an option for this.
					fmt.Printf("%s is present. pulling latest...\n", s.Name)
					err = git.Pull(s.Name)
				}
				if err != nil {
					log.Fatal(err)
				}
			}

			if s.HasECRImage { // TODO: only pull if user is running a -ecr version of the repo.
				docker.Pull(s.ImageURI)
			}
		}

		// Stop running docker containers
		var err error
		err = docker.StopAllContainers()
		if err != nil {
			log.Fatal(err)
		}

		// Stop docker-compose services
		files, err := util.ComposeFiles() // TODO: scope err gotcha
		if err != nil {
			log.Fatal(err)
		}
		cmdStr := fmt.Sprintf("%s stop", files)
		err = util.Exec("docker-compose", strings.Fields(cmdStr)...)
		if err != nil {
			log.Fatal(err)
		}

		// Remove running docker containers
		err = docker.RmContainers()
		if err != nil {
			log.Fatal(err)
		}

		// Start building shit
		// cmdStr = fmt.Sprintf("%s build --parallel %s", files, enabledServices)
		// err = util.Exec("docker-compose", strings.Fields(cmdStr)...)

	},
}

func init() {
	RootCmd.AddCommand(upCmd)
}
