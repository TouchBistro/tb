package commands

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newListCommand(c *cli.Container) *cobra.Command {
	var listOpts struct {
		listServices        bool
		listPlaylists       bool
		listCustomPlaylists bool
		treeMode            bool
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "Lists all available services",
		Run: func(cmd *cobra.Command, args []string) {
			// If no flags provided show everything
			if !listOpts.listServices && !listOpts.listPlaylists && !listOpts.listCustomPlaylists {
				listOpts.listServices = true
				listOpts.listPlaylists = true
				listOpts.listCustomPlaylists = true
			}
			listResult := c.Engine.List(engine.ListOptions{
				ListServices:        listOpts.listServices,
				ListPlaylists:       listOpts.listPlaylists,
				ListCustomPlaylists: listOpts.listCustomPlaylists,
				TreeMode:            listOpts.treeMode,
			})

			if listOpts.listServices {
				fmt.Println("Services:")
				sort.Strings(listResult.Services)
				for _, n := range listResult.Services {
					fmt.Printf("  - %s\n", n)
				}
			}
			if listOpts.listPlaylists {
				fmt.Println("Playlists:")
				printPlaylists(listResult.Playlists, listOpts.treeMode)
			}
			if listOpts.listCustomPlaylists {
				fmt.Println("Custom Playlists:")
				printPlaylists(listResult.CustomPlaylists, listOpts.treeMode)
			}
		},
	}
	listCmd.Flags().BoolVarP(&listOpts.listServices, "services", "s", false, "list services")
	listCmd.Flags().BoolVarP(&listOpts.listPlaylists, "playlists", "p", false, "list playlists")
	listCmd.Flags().BoolVarP(&listOpts.listCustomPlaylists, "custom-playlists", "c", false, "list custom playlists")
	listCmd.Flags().BoolVarP(&listOpts.treeMode, "tree", "t", false, "tree mode, show playlist services")
	return listCmd
}

func printPlaylists(playlists []engine.PlaylistSummary, tree bool) {
	sort.Slice(playlists, func(i, j int) bool {
		return playlists[i].Name < playlists[j].Name
	})
	for _, ps := range playlists {
		fmt.Printf("  - %s\n", ps.Name)
		if !tree {
			continue
		}
		sort.Strings(ps.Services)
		for _, s := range ps.Services {
			fmt.Printf("    - %s\n", s)
		}
	}
}
