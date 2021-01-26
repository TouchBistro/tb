package cmd

import (
	"os"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/spf13/cobra"
)

type logsOptions struct {
	shouldSkipGitPull bool
}

var logsOpts logsOptions

var logsCmd = &cobra.Command{
	Use:   "logs [services...]",
	Short: "View logs from containers",
	PreRun: func(cmd *cobra.Command, args []string) {
		err := config.CloneOrPullRepos(!logsOpts.shouldSkipGitPull)
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var services []string
		if len(args) > 0 {
			for _, serviceName := range args {
				// Make sure it's a valid service
				s, err := config.LoadedServices().Get(serviceName)
				if err != nil {
					fatal.ExitErrf(err, "%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
				}
				services = append(services, s.DockerName())
			}
		}

		c := command.New(command.WithStdout(os.Stdout), command.WithStderr(os.Stderr))
		err := docker.ComposeLogs(services, c)
		if err != nil {
			fatal.ExitErr(err, "Could not view logs.")
		}
	},
}

func init() {
	logsCmd.Flags().BoolVar(&logsOpts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
	rootCmd.AddCommand(logsCmd)
}
