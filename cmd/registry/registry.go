package registry

import (
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "tb registry manages registries from the command line",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// This is here so the one in rootCmd doesn't run
	},
}

func RegistryCmd() *cobra.Command {
	return registryCmd
}
