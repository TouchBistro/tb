package registry

import (
	"github.com/TouchBistro/tb/cli"
	"github.com/spf13/cobra"
)

func NewRegistryCommand(c *cli.Container) *cobra.Command {
	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "tb registry manages registries from the command line",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// This is here so the one in rootCmd doesn't run
		},
	}
	registryCmd.AddCommand(newAddCommand(c), newValidateCommand(c))
	return registryCmd
}
