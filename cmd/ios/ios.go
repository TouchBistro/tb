package ios

import (
	"runtime"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	"github.com/spf13/cobra"
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

		err := config.InitIOS()
		if err != nil {
			fatal.ExitErr(err, "Failed to initialize iOS config")
		}
	},
}

func IOS() *cobra.Command {
	return iosCmd
}
