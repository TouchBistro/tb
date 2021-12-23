package commands

import (
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newDownCommand(c *cli.Container) *cobra.Command {
	downCmd := &cobra.Command{
		Use:   "down [services...]",
		Short: "Stop and remove containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			err := c.Engine.Down(ctx, engine.DownOptions{ServiceNames: args})
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

	flags := downCmd.Flags()
	flags.Bool("no-git-pull", false, "dont update git repositories")
	err := flags.MarkDeprecated("no-git-pull", "it is a no-op and will be removed")
	if err != nil {
		// MarkDeprecated only errors if the flag name is wrong or the message isn't set
		// which is a programming error, so we wanna blow up
		panic(err)
	}
	return downCmd
}
