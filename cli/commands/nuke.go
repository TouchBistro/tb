package commands

import (
	"os"

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
		Args:  cobra.NoArgs,
		Short: "Remove tb resources and data",
		Long: `Removes resources and data. tb nuke can remove the following:
docker containers, docker images, docker networks, docker volumes, cloned service git repos,
downloaded iOS apps, downloaded desktop apps, cloned registries.

Flags must be provided to specify which resources to remove. The special --all flag causes all
resources to be removed, and also removes the directory where tb stores data.

If any docker resources are specified to be removed, any running service containers will first be stopped.

tb nuke will not remove any docker resources that are not managed by tb.

Examples:

Remove all docker containers and images:

	tb nuke --containers --images

Remove all downloaded iOS and desktop apps:

	tb nuke --ios --desktop

Remove everything (completely wipe all tb data):

	tb nuke --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opts.nukeContainers && !opts.nukeImages && !opts.nukeVolumes &&
				!opts.nukeNetworks && !opts.nukeRepos && !opts.nukeDesktopApps &&
				!opts.nukeIOSBuilds && !opts.nukeRegistries && !opts.nukeAll {
				return &cli.ExitError{
					Message: "Error: Must specify what to nuke. Try tb nuke --help to see all the options.",
				}
			}
			err := c.Engine.Nuke(c.Ctx, engine.NukeOptions{
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

			// If --all was used removed the entire .tb dir as a way to completely clean up all trace of tb.
			if opts.nukeAll {
				if err := os.RemoveAll(c.Engine.Workdir()); err != nil {
					return &cli.ExitError{
						Message: "Failed to remove .tb root directory",
						Err:     err,
					}
				}
			}
			c.Tracker.Info("✔ Cleaned up tb data")
			return nil
		},
	}

	flags := nukeCmd.Flags()
	flags.BoolVar(&opts.nukeContainers, "containers", false, "Remove all service containers")
	flags.BoolVar(&opts.nukeImages, "images", false, "Remove all images")
	flags.BoolVar(&opts.nukeVolumes, "volumes", false, "Remove all volumes")
	flags.BoolVar(&opts.nukeNetworks, "networks", false, "Remove all networks")
	flags.BoolVar(&opts.nukeRepos, "repos", false, "Remove all service git repos")
	flags.BoolVar(&opts.nukeDesktopApps, "desktop", false, "Remove all downloaded desktop app builds")
	flags.BoolVar(&opts.nukeIOSBuilds, "ios", false, "Remove all downloaded iOS app builds")
	flags.BoolVar(&opts.nukeRegistries, "registries", false, "Remove all cloned registries")
	flags.BoolVar(&opts.nukeAll, "all", false, "Remove everything")
	flags.Bool("no-git-pull", false, "dont update git repositories")
	err := flags.MarkDeprecated("no-git-pull", "it is a no-op and will be removed")
	if err != nil {
		// MarkDeprecated only errors if the flag name is wrong or the message isn't set
		// which is a programming error, so we wanna blow up
		panic(err)
	}
	return nukeCmd
}
