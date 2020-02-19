package cmd

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
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
			playlists, _ := config.Playlists()
			listPlaylists(playlists, isTreeMode)
		}

		if shouldListCustomPlaylists {
			fmt.Println("Custom Playlists:")
			_, customPlaylists := config.Playlists()
			listPlaylists(customPlaylists, isTreeMode)
		}
	},
}

func listServices(services config.ServiceMap) {
	names := make([]string, len(services))
	i := 0
	for name := range services {
		names[i] = name
		i++
	}

	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
	}
}

func listPlaylists(playlists map[string]config.Playlist, tree bool) {
	names := make([]string, len(playlists))
	i := 0
	for name := range playlists {
		names[i] = name
		i++
	}

	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
		if !tree {
			continue
		}
		list, err := config.GetPlaylist(name, make(map[string]bool))
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
