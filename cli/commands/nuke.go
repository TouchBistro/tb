package commands

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/TouchBistro/goutils/fatal"
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

By default, nuke will prompt the user to select which resources to remove.
Flags may be provided to bypass the prompt and specify which resources to remove.
The special --all flag causes all resources to be removed, and also removes the
directory where tb stores data.

If any docker resources are specified to be removed, any running service containers will
first be stopped and all service containers will be removed.

tb nuke will not remove any docker resources that are not managed by tb.

Examples:

Prompt to select resources to remove:

	tb nuke

Remove all docker images and volumes (will also remove docker containers):

	tb nuke --images --volumes

Remove all downloaded iOS and desktop apps:

	tb nuke --ios --desktop

Remove everything (completely wipe all tb data):

	tb nuke --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no flags were provided do an interactive prompt and ask the user
			// what they would like to remove.
			if !opts.nukeContainers && !opts.nukeImages && !opts.nukeVolumes &&
				!opts.nukeNetworks && !opts.nukeRepos && !opts.nukeDesktopApps &&
				!opts.nukeIOSBuilds && !opts.nukeRegistries && !opts.nukeAll {
				choices := []struct {
					name        string
					optionField *bool
				}{
					{"Containers", &opts.nukeContainers},
					{"Images", &opts.nukeImages},
					{"Volumes", &opts.nukeVolumes},
					{"Networks", &opts.nukeNetworks},
					{"Repos", &opts.nukeRepos},
					{"Desktop Apps", &opts.nukeDesktopApps},
					{"iOS Apps", &opts.nukeIOSBuilds},
					{"Registries", &opts.nukeRegistries},
				}
				var promptOptions []string
				for _, c := range choices {
					promptOptions = append(promptOptions, c.name)
				}

				prompt := &survey.MultiSelect{
					Message:  "Choose what to remove:",
					Options:  promptOptions,
					PageSize: len(promptOptions), // Make sure all choices are rendered without pagination
				}
				var selected []int
				// Use required validator to enforce that at least one option is selected.
				err := survey.AskOne(prompt, &selected, survey.WithValidator(survey.Required))
				if err != nil {
					return &fatal.Error{
						Msg: "Failed to prompt for what to remove",
						Err: err,
					}
				}
				for _, si := range selected {
					*choices[si].optionField = true
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
				return &fatal.Error{
					Msg: "Failed to clean up tb data",
					Err: err,
				}
			}

			// If --all was used removed the entire .tb dir as a way to completely clean up all trace of tb.
			if opts.nukeAll {
				if err := os.RemoveAll(c.Engine.Workdir()); err != nil {
					return &fatal.Error{
						Msg: "Failed to remove .tb root directory",
						Err: err,
					}
				}
			}
			c.Tracker.Info("✔ Cleaned up tb data")
			return nil
		},
	}

	flags := nukeCmd.Flags()
	flags.BoolVar(&opts.nukeContainers, "containers", false, "Remove all service containers")
	flags.BoolVar(&opts.nukeImages, "images", false, "Remove all images (implies --containers)")
	flags.BoolVar(&opts.nukeVolumes, "volumes", false, "Remove all volumes (implies --containers)")
	flags.BoolVar(&opts.nukeNetworks, "networks", false, "Remove all networks (implies --containers)")
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
