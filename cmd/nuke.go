package cmd

import (
	"log"
	"os"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/git"
	"github.com/spf13/cobra"
)

type nukeOptions struct {
	shouldNukeContainers bool
	shouldNukeImages     bool
	shouldNukeVolumes    bool
	shouldNukeNetworks   bool
	shouldNukeRepos      bool
	shouldNukeAll        bool
}

var nukeOpts nukeOptions

var nukeCmd = &cobra.Command{
	Use:   "nuke",
	Short: "Removes all docker images, containers, volumes and networks",
	PreRun: func(cmd *cobra.Command, args []string) {
		if !nukeOpts.shouldNukeContainers &&
			!nukeOpts.shouldNukeImages &&
			!nukeOpts.shouldNukeVolumes &&
			!nukeOpts.shouldNukeNetworks &&
			!nukeOpts.shouldNukeRepos &&
			!nukeOpts.shouldNukeAll {
			log.Fatalln("Error: Must specify what to nuke")
		}

		err := deps.Resolve(deps.Docker)
		if err != nil {
			log.Fatal(err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := docker.StopAllContainers()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("stopping compose services...")
		err = docker.ComposeStop()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("...done")

		if nukeOpts.shouldNukeContainers || nukeOpts.shouldNukeAll {
			log.Println("Removing containers...")
			err = docker.RmContainers()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("...done")
		}

		if nukeOpts.shouldNukeImages || nukeOpts.shouldNukeAll {
			log.Println("Removing images...")
			err = docker.RmImages()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("...done")
		}

		if nukeOpts.shouldNukeNetworks || nukeOpts.shouldNukeAll {
			log.Println("Removing networks...")
			err = docker.RmNetworks()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("...done")
		}

		if nukeOpts.shouldNukeVolumes || nukeOpts.shouldNukeAll {
			log.Println("Removing volumes...")
			err = docker.RmVolumes()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("...done")
		}

		if nukeOpts.shouldNukeRepos || nukeOpts.shouldNukeAll {
			log.Println("Removing repos...")
			for _, repo := range git.RepoNames(config.All()) {
				err = os.RemoveAll(repo)
				if err != nil {
					log.Fatal(err)
				}
			}
			log.Println("...done")
		}
	},
}

func init() {
	RootCmd.AddCommand(nukeCmd)
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeContainers, "containers", false, "nuke all containers")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeImages, "images", false, "nuke all images")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeVolumes, "volumes", false, "nuke all volumes")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeNetworks, "networks", false, "nuke all networks")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeRepos, "repos", false, "nuke all repos")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeAll, "all", false, "nuke evenrything")
}
