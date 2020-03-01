package cmd

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/service"
	"github.com/spf13/cobra"
)

var (
	shouldListServices        bool
	shouldListPlaylists       bool
	shouldListCustomPlaylists bool
	isTreeMode                bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Args:    cobra.NoArgs,
	Short:   "Lists all available services",
	Run: func(cmd *cobra.Command, args []string) {
		// If no flags provided show everything
		if !shouldListServices &&
			!shouldListPlaylists &&
			!shouldListCustomPlaylists {
			shouldListServices = true
			shouldListPlaylists = true
			shouldListCustomPlaylists = true
		}

		if shouldListServices {
			fmt.Println("Services:")
			listServices(config.Services())
		}

		if shouldListPlaylists {
			fmt.Println("Playlists:")
			listPlaylists(config.Playlists().Names(), isTreeMode)
		}

		if shouldListCustomPlaylists {
			fmt.Println("Custom Playlists:")
			listPlaylists(config.Playlists().CustomNames(), isTreeMode)
		}
	},
}

func listServices(services *service.ServiceCollection) {
	names := make([]string, 0)
	services.ForEach(func(s service.Service) {
		names = append(names, s.FullName())
	})

	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
	}
}

func listPlaylists(names []string, tree bool) {
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
		if !tree {
			continue
		}

		list, err := config.Playlists().GetServices(name)
		if err != nil {
			fatal.ExitErr(err, "☒ failed resolving service playlist")
		}

		for _, s := range list {
			fmt.Printf("    - %s\n", s)
		}
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&shouldListServices, "services", "s", false, "list services")
	listCmd.Flags().BoolVarP(&shouldListPlaylists, "playlists", "p", false, "list playlists")
	listCmd.Flags().BoolVarP(&shouldListCustomPlaylists, "custom-playlists", "c", false, "list custom playlists")
	listCmd.Flags().BoolVarP(&isTreeMode, "tree", "t", false, "tree mode, show playlist services")
}
