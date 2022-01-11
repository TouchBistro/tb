package ios

import (
	"github.com/TouchBistro/tb/cli"
	"github.com/spf13/cobra"
)

func NewiOSCommand(c *cli.Container) *cobra.Command {
	iosCmd := &cobra.Command{
		Use:   "ios",
		Short: "Run and manage iOS apps",
		Long:  `tb app ios allows running and managing iOS apps.`,
	}
	iosCmd.AddCommand(newLogsCommand(c), newRunCommand(c))
	return iosCmd
}
