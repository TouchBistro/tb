package cmd

import (
	"os"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/fatal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type nukeOptions struct {
	shouldNukeContainers bool
	shouldNukeImages     bool
	shouldNukeVolumes    bool
	shouldNukeNetworks   bool
	shouldNukeRepos      bool
	shouldNukeConfig     bool
	shouldNukeIOSBuilds  bool
	shouldNukeAll        bool
}

var nukeOpts nukeOptions

var nukeCmd = &cobra.Command{
	Use:   "nuke",
	Short: "Removes all docker images, containers, volumes and networks",
	PreRun: func(cmd *cobra.Command, args []string) {
		if !nukeOpts.shouldNukeContainers &&
			!nukeOpts.shouldNukeIOSBuilds &&
			!nukeOpts.shouldNukeImages &&
			!nukeOpts.shouldNukeVolumes &&
			!nukeOpts.shouldNukeNetworks &&
			!nukeOpts.shouldNukeRepos &&
			!nukeOpts.shouldNukeConfig &&
			!nukeOpts.shouldNukeAll {
			fatal.Exit("Error: Must specify what to nuke. try tb nuke --help to see all the options.")
		}

		err := config.CloneMissingRepos(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		// Passing nothing to compose will shut everything down
		err := docker.ComposeStop(make([]string, 0))
		if err != nil {
			fatal.ExitErr(err, "Failed stopping docker containers and services.")
		}

		if nukeOpts.shouldNukeContainers || nukeOpts.shouldNukeAll {
			log.Infoln("Removing containers...")
			err = docker.RmContainers()
			if err != nil {
				fatal.ExitErr(err, "Failed removing docker containers")
			}
			log.Infoln("...done")
		}

		if nukeOpts.shouldNukeImages || nukeOpts.shouldNukeAll {
			log.Infoln("Removing images...")
			err = docker.RmImages()
			if err != nil {
				fatal.ExitErr(err, "Failed removing docker images.")
			}
			log.Infoln("...done")
		}

		if nukeOpts.shouldNukeNetworks || nukeOpts.shouldNukeAll {
			log.Infoln("Removing networks...")
			err = docker.RmNetworks()
			if err != nil {
				fatal.ExitErr(err, "Failed removing docker networks.")
			}
			log.Infoln("...done")
		}

		if nukeOpts.shouldNukeVolumes || nukeOpts.shouldNukeAll {
			log.Infoln("Removing volumes...")
			err = docker.RmVolumes()
			if err != nil {
				fatal.ExitErr(err, "Failed removing docker volumes.")
			}
			log.Infoln("...done")
		}

		if nukeOpts.shouldNukeRepos || nukeOpts.shouldNukeAll {
			log.Infoln("Removing repos...")
			err := os.RemoveAll(config.ReposPath())
			if err != nil {
				fatal.ExitErr(err, "Failed removing repos.")
			}
			log.Infoln("...done")
		}

		if nukeOpts.shouldNukeConfig || nukeOpts.shouldNukeAll {
			log.Infoln("Removing config files...")
			err := config.RmFiles()
			if err != nil {
				fatal.ExitErr(err, "Failed removing config files.")
			}
			log.Infoln("...done")
		}

		if nukeOpts.shouldNukeIOSBuilds || nukeOpts.shouldNukeAll {
			log.Infoln("Removing ios builds...")
			err := os.RemoveAll(config.IOSBuildPath())
			if err != nil {
				fatal.ExitErr(err, "Failed removing ios builds.")
			}
			log.Infoln("...done")
		}
	},
}

func init() {
	rootCmd.AddCommand(nukeCmd)
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeContainers, "containers", false, "nuke all containers")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeImages, "images", false, "nuke all images")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeVolumes, "volumes", false, "nuke all volumes")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeNetworks, "networks", false, "nuke all networks")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeRepos, "repos", false, "nuke all repos")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeConfig, "config", false, "nuke all config files")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeIOSBuilds, "ios", false, "nuke all downloaded iOS builds")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeAll, "all", false, "nuke everything")
}
