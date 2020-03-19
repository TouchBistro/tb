package app

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/tb/config"
	"github.com/spf13/cobra"
)

type listOptions struct {
	shouldListIOSApps     bool
	shouldListDesktopApps bool
}

var listOpts listOptions

var listCmd = &cobra.Command{
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
		if !listOpts.shouldListIOSApps &&
			!listOpts.shouldListDesktopApps {
			listOpts.shouldListIOSApps = true
			listOpts.shouldListDesktopApps = true
		}

		if listOpts.shouldListIOSApps {
			fmt.Println("iOS Apps:")
			names := config.LoadedIOSApps().Names()
			sort.Strings(names)

			for _, n := range names {
				fmt.Printf("  - %s\n", n)
			}
		}

		if listOpts.shouldListDesktopApps {
			fmt.Println("Desktop Apps:")
			names := config.LoadedDesktopApps().Names()
			sort.Strings(names)

			for _, n := range names {
				fmt.Printf("  - %s\n", n)
			}
		}
	},
}

func init() {
	appCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listOpts.shouldListIOSApps, "ios", false, "list iOS apps")
	listCmd.Flags().BoolVar(&listOpts.shouldListDesktopApps, "desktop", false, "list desktop apps")
}
