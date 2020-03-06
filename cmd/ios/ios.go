package ios

import (
	"runtime"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/simulator"
	"github.com/spf13/cobra"
)

// Flags available for multiple commands
var (
	iosVersion string
	deviceName string
	iosOpts    struct {
		noUpdateRegistries bool
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

		err := config.Init(config.InitOptions{
			UpdateRegistries: !iosOpts.noUpdateRegistries,
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
	iosCmd.PersistentFlags().BoolVar(&iosOpts.noUpdateRegistries, "no-update-registries", false, "Don't update registries when tb is run")
}

func IOS() *cobra.Command {
	return iosCmd
}
