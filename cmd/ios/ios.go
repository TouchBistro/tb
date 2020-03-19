package ios

import (
	"runtime"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/simulator"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Flags available for multiple commands
var (
	iosVersion string
	deviceName string
	iosOpts    struct {
		noRegistryPull bool
	}
)

var iosCmd = &cobra.Command{
	Use:   "ios",
	Short: "tb ios allows running and managing iOS apps",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Put ios specific configuration & setup logic here that should run before every subcommand
		// Don't put in root so we don't blow up CI which runs in a linux container
		// Also no need to do ios specific stuff if not using ios commands

		if runtime.GOOS != "darwin" {
			fatal.Exit("Error: tb ios is only supported on macOS")
		}

		log.Infoln(color.Yellow("tb ios is deprecated and will be removed in the next major release"))
		log.Infoln(color.Yellow("Please use tb app ios instead"))

		// Need to do this explicitly here since we are defining PersistentPreRun
		// PersistentPreRun overrides the parent command's one if defined, so the one in root won't be run.
		err := config.Init(config.InitOptions{
			UpdateRegistries: !iosOpts.noRegistryPull,
			LoadServices:     false,
			LoadApps:         true,
		})
		if err != nil {
			fatal.ExitErr(err, "Failed to initialize config files")
		}

		err = simulator.LoadSimulators()
		if err != nil {
			fatal.ExitErr(err, "Failed to find available iOS simulators")
		}
	},
}

func init() {
	iosCmd.PersistentFlags().BoolVar(&iosOpts.noRegistryPull, "no-registry-pull", false, "Don't pull latest version of registries when tb is run")
}

func IOS() *cobra.Command {
	return iosCmd
}
