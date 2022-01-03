package registry

import (
	"errors"
	"fmt"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/config"
	"github.com/spf13/cobra"
)

func newAddCommand(c *cli.Container) *cobra.Command {
	return &cobra.Command{
		Use:   "add <registry-name>",
		Args:  cli.ExpectSingleArg("registry name"),
		Short: "Add a registry",
		Long: `Adds a registry to tb. If the registry has already been added, the command will no-op.

Examples:

Add the registry named TouchBistro/tb-registry-example:

	tb registry add TouchBistro/tb-registry-example`,
		RunE: func(cmd *cobra.Command, args []string) error {
			registryName := args[0]
			err := config.AddRegistry(registryName, "")
			if errors.Is(err, config.ErrRegistryExists) {
				c.Tracker.Infof(color.Green("☑ registry %s has already been added"), registryName)
				return nil
			} else if err != nil {
				return &cli.ExitError{
					Message: fmt.Sprintf("failed to add registry %s", registryName),
					Err:     err,
				}
			}
			c.Tracker.Infof(color.Green("Successfully added registry %s"), registryName)
			return nil
		},
	}
}
