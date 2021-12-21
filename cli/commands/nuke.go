package commands

import (
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newNukeCommand(c *cli.Container) *cobra.Command {
	var nukeOpts struct {
		nukeContainers  bool
		nukeImages      bool
		nukeVolumes     bool
		nukeNetworks    bool
		nukeRepos       bool
		nukeDesktopApps bool
		nukeIOSBuilds   bool
		nukeRegistries  bool
		nukeAll         bool
		skipGitPull     bool
	}
	nukeCmd := &cobra.Command{
		Use:   "nuke",
		Short: "Removes all docker images, containers, volumes and networks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !nukeOpts.nukeContainers && !nukeOpts.nukeImages && !nukeOpts.nukeVolumes &&
				!nukeOpts.nukeNetworks && !nukeOpts.nukeRepos && !nukeOpts.nukeDesktopApps &&
				!nukeOpts.nukeIOSBuilds && !nukeOpts.nukeRegistries && !nukeOpts.nukeAll {
				return &cli.ExitError{
					Message: "Error: Must specify what to nuke. Try tb nuke --help to see all the options.",
				}
			}
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			err := c.Engine.Nuke(ctx, engine.NukeOptions{
				RemoveContainers:  nukeOpts.nukeContainers || nukeOpts.nukeAll,
				RemoveImages:      nukeOpts.nukeImages || nukeOpts.nukeAll,
				RemoveNetworks:    nukeOpts.nukeNetworks || nukeOpts.nukeAll,
				RemoveVolumes:     nukeOpts.nukeVolumes || nukeOpts.nukeAll,
				RemoveRepos:       nukeOpts.nukeRepos || nukeOpts.nukeAll,
				RemoveDesktopApps: nukeOpts.nukeDesktopApps || nukeOpts.nukeAll,
				RemoveiOSApps:     nukeOpts.nukeIOSBuilds || nukeOpts.nukeAll,
				RemoveRegistries:  nukeOpts.nukeRegistries || nukeOpts.nukeAll,
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
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeContainers, "containers", false, "nuke all containers")
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeImages, "images", false, "nuke all images")
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeVolumes, "volumes", false, "nuke all volumes")
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeNetworks, "networks", false, "nuke all networks")
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeRepos, "repos", false, "nuke all repos")
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeDesktopApps, "desktop", false, "nuke all downloaded desktop app builds")
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeIOSBuilds, "ios", false, "nuke all downloaded iOS builds")
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeRegistries, "registries", false, "nuke all registries")
	nukeCmd.Flags().BoolVar(&nukeOpts.nukeAll, "all", false, "nuke everything")
	nukeCmd.Flags().BoolVar(&nukeOpts.skipGitPull, "no-git-pull", false, "dont update git repositories")
	err := nukeCmd.Flags().MarkDeprecated("no-git-pull", "it is a no-op")
	if err != nil {
		// MarkDeprecated only errors if the flag name is wrong or the message isn't set
		// which is a programming error, so we wanna blow up
		panic(err)
	}
	return nukeCmd
}
