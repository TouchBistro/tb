package registry

import (
	"os"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <registry-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Adds a registry to tb",
	Long: `Adds a registry to tb.

Example:
- adds the registry named TouchBistro/tb-registry-example
  tb registry add TouchBistro/tb-registry-example`,
	Run: func(cmd *cobra.Command, args []string) {
		registryName := args[0]
		log.Infof(color.Cyan("☐ Adding registry %s..."), registryName)

		err := config.AddRegistry(registryName)
		if err == config.ErrRegistryExists {
			log.Infof(color.Green("☑ registry %s has already been added"), registryName)
			os.Exit(0)
		} else if err != nil {
			fatal.ExitErrf(err, "failed to add registry %s", registryName)
		}

		log.Infof(color.Green("☑ Successfully added registry %s"), registryName)
	},
}

func init() {
	registryCmd.AddCommand(addCmd)
}
