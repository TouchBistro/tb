package cmd

import (
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Args:  cobra.NoArgs,
	Short: "Stops any running services and removes all containers",
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := deps.Resolve(deps.Docker); err != nil {
			util.FatalErr(err, "Could not resolve dependencies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := docker.StopContainersAndServices()
		if err != nil {
			util.FatalErr(err, "Could not stop containers and services")
		}

		log.Println("removing stopped containers...")
		err = docker.RmContainers()
		if err != nil {
			util.FatalErr(err, "Could not remove stopped containers")
		}
		log.Println("...done")
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
