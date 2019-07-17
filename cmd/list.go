package cmd

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/tb/config"
	"github.com/spf13/cobra"
)

var (
	shouldListServices        bool
	shouldListPlaylists       bool
	shouldListCustomPlaylists bool
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
			listNames(getServiceNames(config.Services()))
		}

		if shouldListPlaylists {
			fmt.Println("Playlists:")
			listNames(getPlaylistNames(config.Playlists()))
		}

		if shouldListCustomPlaylists {
			fmt.Println("Custom Playlists:")
			listNames(getPlaylistNames(config.TBRC().Playlists))
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&shouldListServices, "services", "s", false, "list services")
	listCmd.Flags().BoolVarP(&shouldListPlaylists, "playlists", "p", false, "list playlists")
	listCmd.Flags().BoolVarP(&shouldListCustomPlaylists, "custom-playlists", "c", false, "list custom playlists")
}

func getServiceNames(services map[string]config.Service) []string {
	names := make([]string, len(services))
	i := 0
	for name := range services {
		names[i] = name
		i++
	}

	return names
}

func getPlaylistNames(playlists map[string]config.Playlist) []string {
	names := make([]string, len(playlists))
	i := 0
	for name := range playlists {
		names[i] = name
		i++
	}

	return names
}

func listNames(names []string) {
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
	}
}
