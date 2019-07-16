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
			util.FatalErr("Could not resolve dependencies", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := docker.StopContainersAndServices()
		if err != nil {
			util.FatalErr("Could not stop containers and services", err)
		}

		log.Println("removing stopped containers...")
		err = docker.RmContainers()
		if err != nil {
			util.FatalErr("Could not remove stopped containers", err)
		}
		log.Println("...done")
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
