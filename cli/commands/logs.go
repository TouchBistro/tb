package commands

import (
	"os"

	"github.com/TouchBistro/goutils/progress"
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
		Short: "View logs from containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			return c.Engine.Logs(ctx, os.Stdout, engine.LogsOptions{
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
	flags.BoolVar(&opts.skipGitPull, "no-git-pull", false, "dont update git repositories")
	return logsCmd
}
