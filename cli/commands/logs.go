package commands

import (
	"os"

	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type logsOptions struct {
	skipGitPull bool
}

func newLogsCommand(c *cli.Container) *cobra.Command {
	var opts logsOptions
	logsCmd := &cobra.Command{
		Use:   "logs [services...]",
		Args:  cobra.ArbitraryArgs,
		Short: "View logs from containers",
		Long: `View logs from service containers. By default logs from all running service containers are shown.
Service names can be provided as args to filter logs to only containers for those services.

Examples:

Show logs from all service containers:

	tb logs

Show logs only from the postgres and redis containers:

	tb logs postgres redis`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Engine.Logs(c.Ctx, os.Stdout, engine.LogsOptions{
				ServiceNames: args,
				// TODO(@cszatmary): Make these configurable through flags.
				// This would be a breaking change though.
				Follow:      true,
				Tail:        -1,
				SkipGitPull: opts.skipGitPull,
			})
		},
	}

	flags := logsCmd.Flags()
	flags.BoolVar(&opts.skipGitPull, "no-git-pull", false, "Don't update git repositories")
	return logsCmd
}
