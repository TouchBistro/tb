package app

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type listOptions struct {
	listIOSApps     bool
	listDesktopApps bool
}

func newListCommand(c *cli.Container) *cobra.Command {
	var opts listOptions
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
			if !opts.listIOSApps && !opts.listDesktopApps {
				opts.listIOSApps = true
				opts.listDesktopApps = true
			}
			res := c.Engine.AppList(engine.AppListOptions{
				ListiOSApps:     opts.listIOSApps,
				ListDesktopApps: opts.listDesktopApps,
			})
			if opts.listIOSApps {
				fmt.Println("iOS Apps:")
				sort.Strings(res.IOSApps)
				for _, n := range res.IOSApps {
					fmt.Printf("  - %s\n", n)
				}
			}
			if opts.listDesktopApps {
				fmt.Println("Desktop Apps:")
				sort.Strings(res.DesktopApps)
				for _, n := range res.DesktopApps {
					fmt.Printf("  - %s\n", n)
				}
			}
		},
	}

	flags := listCmd.Flags()
	flags.BoolVar(&opts.listIOSApps, "ios", false, "list iOS apps")
	flags.BoolVar(&opts.listDesktopApps, "desktop", false, "list desktop apps")
	return listCmd
}
