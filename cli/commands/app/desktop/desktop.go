package desktop

import (
	"github.com/TouchBistro/tb/cli"
	"github.com/spf13/cobra"
)

func NewDesktopCommand(c *cli.Container) *cobra.Command {
	desktopCmd := &cobra.Command{
		Use:   "desktop",
		Short: "Running and manage desktop apps",
		Long:  `tb app desktop allows running and managing desktop apps.`,
	}
	desktopCmd.AddCommand(newRunCommand(c))
	return desktopCmd
}
