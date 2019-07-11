package cmd

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
	Short:   "Lists all available services",
	Run: func(cmd *cobra.Command, args []string) {
		services := config.Services()
		names := make([]string, 0, len(services))
		for name := range services {
			names = append(names, name)
		}

		sort.Strings(names)
		for _, name := range names {
			fmt.Printf("%s\n", name)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
