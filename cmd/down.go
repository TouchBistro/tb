package cmd

import (
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stops any running services and removes all containers",
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := deps.Resolve(deps.Docker); err != nil {
			log.Fatal(err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := docker.StopContainersAndServices()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("removing any running containers...")
		err = docker.RmContainers()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("...done")
	},
}

func init() {
	RootCmd.AddCommand(downCmd)
}
