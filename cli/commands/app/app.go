package app

import (
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/cli/commands/app/desktop"
	"github.com/TouchBistro/tb/cli/commands/app/ios"
	"github.com/spf13/cobra"
)

func NewAppCommand(c *cli.Container) *cobra.Command {
	appCmd := &cobra.Command{
		Use:   "app",
		Short: "tb app allows running and managing different kinds of applications",
	}
	appCmd.AddCommand(desktop.NewDesktopCommand(c), ios.NewiOSCommand(c), newListCommand(c))
	return appCmd
}
