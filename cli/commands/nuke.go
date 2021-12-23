package commands

import (
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type nukeOptions struct {
	nukeContainers  bool
	nukeImages      bool
	nukeVolumes     bool
	nukeNetworks    bool
	nukeRepos       bool
	nukeDesktopApps bool
	nukeIOSBuilds   bool
	nukeRegistries  bool
	nukeAll         bool
}

func newNukeCommand(c *cli.Container) *cobra.Command {
	var opts nukeOptions
	nukeCmd := &cobra.Command{
		Use:   "nuke",
		Short: "Removes all docker images, containers, volumes and networks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opts.nukeContainers && !opts.nukeImages && !opts.nukeVolumes &&
				!opts.nukeNetworks && !opts.nukeRepos && !opts.nukeDesktopApps &&
				!opts.nukeIOSBuilds && !opts.nukeRegistries && !opts.nukeAll {
				return &cli.ExitError{
					Message: "Error: Must specify what to nuke. Try tb nuke --help to see all the options.",
				}
			}
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			err := c.Engine.Nuke(ctx, engine.NukeOptions{
				RemoveContainers:  opts.nukeContainers || opts.nukeAll,
				RemoveImages:      opts.nukeImages || opts.nukeAll,
				RemoveNetworks:    opts.nukeNetworks || opts.nukeAll,
				RemoveVolumes:     opts.nukeVolumes || opts.nukeAll,
				RemoveRepos:       opts.nukeRepos || opts.nukeAll,
				RemoveDesktopApps: opts.nukeDesktopApps || opts.nukeAll,
				RemoveiOSApps:     opts.nukeIOSBuilds || opts.nukeAll,
				RemoveRegistries:  opts.nukeRegistries || opts.nukeAll,
			})
			if err != nil {
				return &cli.ExitError{
					Message: "Failed to clean up tb data",
					Err:     err,
				}
			}
			c.Tracker.Info("✔ Cleaned up tb data")
			return nil
		},
	}

	flags := nukeCmd.Flags()
	flags.BoolVar(&opts.nukeContainers, "containers", false, "nuke all containers")
	flags.BoolVar(&opts.nukeImages, "images", false, "nuke all images")
	flags.BoolVar(&opts.nukeVolumes, "volumes", false, "nuke all volumes")
	flags.BoolVar(&opts.nukeNetworks, "networks", false, "nuke all networks")
	flags.BoolVar(&opts.nukeRepos, "repos", false, "nuke all repos")
	flags.BoolVar(&opts.nukeDesktopApps, "desktop", false, "nuke all downloaded desktop app builds")
	flags.BoolVar(&opts.nukeIOSBuilds, "ios", false, "nuke all downloaded iOS builds")
	flags.BoolVar(&opts.nukeRegistries, "registries", false, "nuke all registries")
	flags.BoolVar(&opts.nukeAll, "all", false, "nuke everything")
	flags.Bool("no-git-pull", false, "dont update git repositories")
	err := flags.MarkDeprecated("no-git-pull", "it is a no-op and will be removed")
	if err != nil {
		// MarkDeprecated only errors if the flag name is wrong or the message isn't set
		// which is a programming error, so we wanna blow up
		panic(err)
	}
	return nukeCmd
}
