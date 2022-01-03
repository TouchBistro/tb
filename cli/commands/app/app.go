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
		Short: "Run and manage apps",
		Long: `tb app provides functionality for working with different kinds of applications.

For working with iOS apps see 'tb app ios'. For working with desktop apps see 'tb app destkop'.`,
	}
	appCmd.AddCommand(desktop.NewDesktopCommand(c), ios.NewiOSCommand(c), newListCommand(c))
	return appCmd
}
