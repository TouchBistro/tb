package cmd

import (
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/goutils/fatal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down [services...]",
	Short: "Stop and remove containers",
	PreRun: func(cmd *cobra.Command, args []string) {
		err := config.CloneMissingRepos(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		log.Debug("stopping compose services...")
		err := docker.ComposeStop(args)
		if err != nil {
			fatal.ExitErr(err, "failed stopping compose services")
		}
		log.Debug("...done")
		if err != nil {
			fatal.ExitErr(err, "could not stop containers and services")
		}

		log.Println("removing stopped containers...")
		err = docker.ComposeRm(args)
		if err != nil {
			fatal.ExitErr(err, "could not remove stopped containers")
		}
		log.Println("done")
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
