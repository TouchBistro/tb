package ios

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/tb/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Args:    cobra.NoArgs,
	Short:   "Lists all available iOS and macOS apps",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("iOS Apps:")
		names := config.LoadedIOSApps().Names()
		sort.Strings(names)
		for _, name := range names {
			fmt.Printf("  - %s\n", name)
		}
	},
}

func init() {
	iosCmd.AddCommand(listCmd)
}
