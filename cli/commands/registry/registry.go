package registry

import (
	"github.com/TouchBistro/tb/cli"
	"github.com/spf13/cobra"
)

func NewRegistryCommand(c *cli.Container) *cobra.Command {
	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage registries from the command line",
		Long: `tb registry manages registries from the command line.

A registry contains configuration to define services, playlists, and apps that tb can run.
See https://github.com/TouchBistro/tb/blob/master/docs/registries.md for more details.`,
	}
	registryCmd.AddCommand(newAddCommand(c), newValidateCommand(c))
	return registryCmd
}
