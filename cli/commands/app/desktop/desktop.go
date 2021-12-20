package desktop

import (
	"github.com/TouchBistro/tb/cli"
	"github.com/spf13/cobra"
)

func NewDesktopCommand(c *cli.Container) *cobra.Command {
	desktopCmd := &cobra.Command{
		Use:   "desktop",
		Short: "tb app desktop allows running and managing desktop applications",
	}
	desktopCmd.AddCommand(newRunCommand(c))
	return desktopCmd
}
