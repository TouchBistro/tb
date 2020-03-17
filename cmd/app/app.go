package app

import (
	"runtime"

	"github.com/TouchBistro/goutils/fatal"
	iosCmd "github.com/TouchBistro/tb/cmd/app/ios"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/simulator"
	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "tb app allows running and managing different kinds of applications",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Put app specific configuration & setup logic here

		// Check if current command is an ios subcommand
		isIOSCommand := cmd.Parent().Name() == "ios"

		if isIOSCommand && runtime.GOOS != "darwin" {
			fatal.Exit("Error: tb app ios is only supported on macOS")
		}

		// Get global flag value
		noRegistryPull, err := cmd.Flags().GetBool("no-registry-pull")
		if err != nil {
			// This is a coding error
			fatal.ExitErr(err, "failed to get flag")
		}

		// Need to do this explicitly here since we are defining PersistentPreRun
		// PersistentPreRun overrides the parent command's one if defined, so the one in root won't be run.
		err = config.Init(config.InitOptions{
			UpdateRegistries: !noRegistryPull,
			LoadServices:     false,
			LoadApps:         true,
		})
		if err != nil {
			fatal.ExitErr(err, "Failed to initialize config files")
		}

		if isIOSCommand {
			err = simulator.LoadSimulators()
			if err != nil {
				fatal.ExitErr(err, "Failed to find available iOS simulators")
			}
		}
	},
}

func init() {
	appCmd.AddCommand(iosCmd.IOSCmd())
}

func AppCmd() *cobra.Command {
	return appCmd
}
