package commands

import (
	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/resource"
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
			// DISCUSS(@cszatmary): Does it makes sense to allow stopping services that don't exist in
			// registries anymore? Ex: a service is removed, but you still have it running locally?
			// It seems to make more sense to not allow services that have been removed. However, this
			// means you could get into weird states where a service was removed from the registry and
			// now you have no way to stop tb without manually docker commands.
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
			if err := c.Engine.Down(ctx, services); err != nil {
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
