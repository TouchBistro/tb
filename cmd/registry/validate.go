package registry

import (
	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/registry"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <path>",
	Args:  cobra.ExactArgs(1),
	Short: "Validates the config files of a registry at the given path.",
	Long: `Validates the config files of a registry at the given path.

Example:
- validates the config files in the current directory
  tb registry validate .`,
	Run: func(cmd *cobra.Command, args []string) {
		registryPath := args[0]
		log.Infof(color.Cyan("Validating registry files at path %s..."), registryPath)

		err := registry.Validate(registryPath)
		if err != nil {
			fatal.ExitErrf(err, "failed to validate registry at path %s", registryPath)
		}

		log.Info(color.Green("✅ registry is valid"))
	},
}

func init() {
	registryCmd.AddCommand(validateCmd)
}
