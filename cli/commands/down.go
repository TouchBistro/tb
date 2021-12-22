package commands

import (
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newDownCommand(c *cli.Container) *cobra.Command {
	var downOpts struct {
		skipGitPull bool
	}
	downCmd := &cobra.Command{
		Use:   "down [services...]",
		Short: "Stop and remove containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			err := c.Engine.Down(ctx, args, engine.DownOptions{SkipGitPull: downOpts.skipGitPull})
			if err != nil {
				return &cli.ExitError{
					Message: "Failed to stop services",
					Err:     err,
				}
			}
			c.Tracker.Info("✔ Stopped services")
			return nil
		},
	}
	downCmd.Flags().BoolVar(&downOpts.skipGitPull, "no-git-pull", false, "dont update git repositories")
	return downCmd
}
