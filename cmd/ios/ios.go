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
}

func init() {
	if runtime.GOOS != "darwin" {
		return
	}

	err := config.InitIOS()
	if err != nil {
		fatal.ExitErr(err, "Failed to initialize iOS config")
	}
}

func IOS() *cobra.Command {
	return iosCmd
}
