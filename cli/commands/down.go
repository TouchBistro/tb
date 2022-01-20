package commands

import (
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newDownCommand(c *cli.Container) *cobra.Command {
	downCmd := &cobra.Command{
		Use:   "down [services...]",
		Args:  cobra.ArbitraryArgs,
		Short: "Stop and remove containers",
		Long: `Stops and removes running service containers.
By default all running service containers are stopped and removed.
Args can be provided to only stop and remove specific containers.

Examples:

Stop and remove all service containers:

	tb down

Stop and remove on the postgres and redis containers:

	tb down postgres redis`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := c.Engine.Down(c.Ctx, engine.DownOptions{ServiceNames: args})
			if err != nil {
				return &fatal.Error{
					Msg: "Failed to stop services",
					Err: err,
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
