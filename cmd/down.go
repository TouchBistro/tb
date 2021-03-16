package cmd

import (
	"os"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
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
		var names []string
		for _, serviceName := range args {
			s, err := config.LoadedServices().Get(serviceName)
			if err != nil {
				fatal.ExitErrf(err, "%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
			}
			names = append(names, s.DockerName())
		}

		s := spinner.New(
			spinner.WithStartMessage("Stopping docker services"),
			spinner.WithStopMessage("Finished stopping docker services"),
			spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
		)
		log.SetOutput(s)
		defer log.SetOutput(os.Stderr)
		s.Start()

		err := docker.ComposeStop(names)
		if err != nil {
			s.Stop()
			fatal.ExitErr(err, "failed stopping services")
		}
		s.UpdateMessage("Removing stopped service containers")
		err = docker.ComposeRm(names)
		s.Stop()
		if err != nil {
			fatal.ExitErr(err, "failed removing stopped containers")
		}
	},
}

func init() {
	downCmd.Flags().BoolVar(&downOpts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
	rootCmd.AddCommand(downCmd)
}
