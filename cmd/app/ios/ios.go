package ios

import (
	"github.com/spf13/cobra"
)

var iosCmd = &cobra.Command{
	Use:   "ios",
	Short: "tb app ios allows running and managing iOS apps",
}

func IOSCmd() *cobra.Command {
	return iosCmd
}
