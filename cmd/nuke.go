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

var (
	shouldNukeContainers bool
	shouldNukeImages     bool
	shouldNukeVolumes    bool
	shouldNukeNetworks   bool
	shouldNukeRepos      bool
	shouldNukeAll        bool
)

var nukeCmd = &cobra.Command{
	Use:   "nuke",
	Short: "Removes all docker images, containers, volumes and networks",
	PreRun: func(cmd *cobra.Command, args []string) {
		if !shouldNukeContainers &&
			!shouldNukeImages &&
			!shouldNukeVolumes &&
			!shouldNukeNetworks &&
			!shouldNukeRepos &&
			!shouldNukeAll {
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

		if shouldNukeContainers || shouldNukeAll {
			log.Println("Removing containers...")
			err = docker.RmContainers()

			if err != nil {
				log.Fatal(err)
			}
			log.Println("...done")
		}

		if shouldNukeImages || shouldNukeAll {
			log.Println("Removing images...")
			err = docker.RmImages()

			if err != nil {
				log.Fatal(err)
			}
			log.Println("...done")
		}

		if shouldNukeNetworks || shouldNukeAll {
			log.Println("Removing networks...")
			err = docker.RmNetworks()

			if err != nil {
				log.Fatal(err)
			}
			log.Println("...done")
		}

		// TODO figure out how to do this
		// if shouldNukeVolumes || shouldNukeAll {
		// 	log.Println("Removing volumes...")
		// 	err = docker.RmVolumes()

		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}
		// 	log.Println("...done")
		// }

		if shouldNukeRepos || shouldNukeAll {
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
	nukeCmd.Flags().BoolVar(&shouldNukeContainers, "containers", false, "nuke all containers")
	nukeCmd.Flags().BoolVar(&shouldNukeImages, "images", false, "nuke all images")
	nukeCmd.Flags().BoolVar(&shouldNukeVolumes, "volumes", false, "nuke all volumes")
	nukeCmd.Flags().BoolVar(&shouldNukeNetworks, "networks", false, "nuke all networks")
	nukeCmd.Flags().BoolVar(&shouldNukeRepos, "repos", false, "nuke all repos")
	nukeCmd.Flags().BoolVar(&shouldNukeAll, "all", false, "nuke evenrything")
}
