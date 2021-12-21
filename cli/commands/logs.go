package commands

import (
	"os"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/TouchBistro/tb/resource"
	"github.com/spf13/cobra"
)

func newLogsCommand(c *cli.Container) *cobra.Command {
	var logsOpts struct {
		skipGitPull bool
	}
	logsCmd := &cobra.Command{
		Use:   "logs [services...]",
		Short: "View logs from containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			services, err := c.Engine.ResolveServices(args)
			if errors.Is(err, resource.ErrNotFound) {
				return &cli.ExitError{
					Message: "Try running `tb list` to see available services",
					Err:     err,
				}
			} else if err != nil {
				return err
			}
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			return c.Engine.Logs(ctx, services, os.Stderr, engine.LogsOptions{
				// TODO(@cszatmary): Make these configurable through flags.
				// This would be a breaking change though.
				Follow: true,
				Tail:   -1,
			})
		},
	}
	logsCmd.Flags().BoolVar(&logsOpts.skipGitPull, "no-git-pull", false, "dont update git repositories")
	return logsCmd
}
