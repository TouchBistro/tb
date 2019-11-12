package cmd

import (
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/fatal"
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
		services := ""

		if len(args) > 0 {
			var b strings.Builder
			for _, serviceName := range args {
				// Make sure it's a valid service
				s, ok := config.Services()[serviceName]
				if !ok {
					fatal.Exitf("%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
				}

				b.WriteString(config.ComposeName(serviceName, s))
				b.WriteString(" ")
			}

			services = b.String()
		}

		err := docker.StopContainersAndServices(services)
		if err != nil {
			fatal.ExitErr(err, "could not stop containers and services")
		}

		log.Println("removing stopped containers...")
		err = docker.ComposeRm(services)
		if err != nil {
			fatal.ExitErr(err, "could not remove stopped containers")
		}
		log.Println("done")
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
