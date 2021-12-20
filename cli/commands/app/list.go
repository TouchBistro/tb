package app

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newListCommand(c *cli.Container) *cobra.Command {
	var listOpts struct {
		listIOSApps     bool
		listDesktopApps bool
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "Lists all available apps",
		Long: `Lists all available apps. Flags can be used to list only specific types of apps.

	Examples:
	- List only iOS apps
	  tb apps list --ios`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no flags provided show everything
			if !listOpts.listIOSApps &&
				!listOpts.listDesktopApps {
				listOpts.listIOSApps = true
				listOpts.listDesktopApps = true
			}
			res := c.Engine.AppList(engine.AppListOptions{
				ListiOSApps:     listOpts.listIOSApps,
				ListDesktopApps: listOpts.listDesktopApps,
			})
			if listOpts.listIOSApps {
				fmt.Println("iOS Apps:")
				sort.Strings(res.IOSApps)
				for _, n := range res.IOSApps {
					fmt.Printf("  - %s\n", n)
				}
			}
			if listOpts.listDesktopApps {
				fmt.Println("Desktop Apps:")
				sort.Strings(res.DesktopApps)
				for _, n := range res.DesktopApps {
					fmt.Printf("  - %s\n", n)
				}
			}
		},
	}
	listCmd.Flags().BoolVar(&listOpts.listIOSApps, "ios", false, "list iOS apps")
	listCmd.Flags().BoolVar(&listOpts.listDesktopApps, "desktop", false, "list desktop apps")
	return listCmd
}
