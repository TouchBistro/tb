package ios

import (
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	"github.com/spf13/cobra"
)

var iosCmd = &cobra.Command{
	Use:   "ios",
	Short: "tb ios allows running and managing iOS apps",
}

func init() {
	err := config.InitIOS()
	if err != nil {
		fatal.ExitErr(err, "Failed to initialize iOS config")
	}
}

func IOS() *cobra.Command {
	return iosCmd
}
