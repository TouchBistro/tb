package cmd

import (
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type downOptions struct {
	shouldSkipGitPull bool
}

var downOpts downOptions

var downCmd = &cobra.Command{
	Use:   "down [services...]",
	Short: "Stop and remove containers",
	PreRun: func(cmd *cobra.Command, args []string) {
		err := config.CloneOrPullRepos(!downOpts.shouldSkipGitPull)
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		log.Debug("stopping compose services...")

		names := make([]string, len(args))
		for _, serviceName := range args {
			s, err := config.LoadedServices().Get(serviceName)
			if err != nil {
				fatal.ExitErrf(err, "%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
			}

			names = append(names, util.DockerName(s.FullName()))
		}

		err := docker.ComposeStop(names)
		if err != nil {
			fatal.ExitErr(err, "failed stopping compose services")
		}
		log.Debug("...done")
		if err != nil {
			fatal.ExitErr(err, "could not stop containers and services")
		}

		log.Println("removing stopped containers...")
		err = docker.ComposeRm(names)
		if err != nil {
			fatal.ExitErr(err, "could not remove stopped containers")
		}
		log.Println("done")
	},
}

func init() {
	downCmd.Flags().BoolVar(&downOpts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")

	rootCmd.AddCommand(downCmd)
}
