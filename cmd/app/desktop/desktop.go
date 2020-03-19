package desktop

import (
	"github.com/spf13/cobra"
)

var desktopCmd = &cobra.Command{
	Use:   "desktop",
	Short: "tb app desktop allows running and managing desktop applications",
}

func DesktopCmd() *cobra.Command {
	return desktopCmd
}
