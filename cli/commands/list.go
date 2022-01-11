package commands

import (
	"fmt"
	"sort"

	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type listOptions struct {
	listServices        bool
	listPlaylists       bool
	listCustomPlaylists bool
	treeMode            bool
}

func newListCommand(c *cli.Container) *cobra.Command {
	var opts listOptions
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List available services and playlists",
		Long: `Lists available services, playlists, and custom playlists.
Custom playlists are playlists that are defined in a user's .tbrc.yml.

Examples:

List all services, playlists, and custom playlists:

	tb list

List only services:

	tb list --services

List only custom playlists along with the services in each playlist (tree mode):

	tb list --custom-playlists --tree`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no flags provided show everything
			if !opts.listServices && !opts.listPlaylists && !opts.listCustomPlaylists {
				opts.listServices = true
				opts.listPlaylists = true
				opts.listCustomPlaylists = true
			}
			listResult := c.Engine.List(engine.ListOptions{
				ListServices:        opts.listServices,
				ListPlaylists:       opts.listPlaylists,
				ListCustomPlaylists: opts.listCustomPlaylists,
				TreeMode:            opts.treeMode,
			})

			if opts.listServices {
				fmt.Println("Services:")
				sort.Strings(listResult.Services)
				for _, n := range listResult.Services {
					fmt.Printf("  - %s\n", n)
				}
			}
			if opts.listPlaylists {
				fmt.Println("Playlists:")
				printPlaylists(listResult.Playlists, opts.treeMode)
			}
			if opts.listCustomPlaylists {
				fmt.Println("Custom Playlists:")
				printPlaylists(listResult.CustomPlaylists, opts.treeMode)
			}
		},
	}

	flags := listCmd.Flags()
	flags.BoolVarP(&opts.listServices, "services", "s", false, "List services")
	flags.BoolVarP(&opts.listPlaylists, "playlists", "p", false, "List playlists")
	flags.BoolVarP(&opts.listCustomPlaylists, "custom-playlists", "c", false, "List custom playlists")
	flags.BoolVarP(&opts.treeMode, "tree", "t", false, "Tree mode, show each playlist's services")
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
