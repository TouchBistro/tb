package cmd

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/util"
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
	shouldNukeRegistries bool
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
			!nukeOpts.shouldNukeConfig &&
			!nukeOpts.shouldNukeIOSBuilds &&
			!nukeOpts.shouldNukeRegistries &&
			!nukeOpts.shouldNukeAll {
			fatal.Exit("Error: Must specify what to nuke. try tb nuke --help to see all the options.")
		}

		err := config.CloneMissingRepos()
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		// Passing nothing to compose will shut everything down
		err := docker.ComposeStop(make([]string, 0))
		if err != nil {
			fatal.ExitErr(err, "Failed stopping docker compose services.")
		}

		if nukeOpts.shouldNukeContainers || nukeOpts.shouldNukeAll {
			log.Infoln("Stopping running containers...")
			err = docker.StopAllContainers()
			if err != nil {
				fatal.ExitErr(err, "Failed stopping docker containers")
			}

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

			repos := make([]string, 0)
			it := config.LoadedServices().Iter()
			for it.HasNext() {
				s := it.Next()
				if s.HasGitRepo() {
					repos = append(repos, s.GitRepo)
				}
			}
			repos = util.UniqueStrings(repos)

			for _, repo := range repos {
				log.Debugf("Removing repo %s...", repo)
				repoPath := filepath.Join(config.ReposPath(), repo)
				err := os.RemoveAll(repoPath)
				if err != nil {
					fatal.ExitErrf(err, "Failed removing repo %s.", repo)
				}
			}

			log.Infoln("Removing any remaining repo directorys...")
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

		if nukeOpts.shouldNukeRegistries || nukeOpts.shouldNukeAll {
			log.Infoln("Removing registries...")
			for _, r := range config.Registries() {
				// Don't remove local registries
				if r.LocalPath != "" {
					continue
				}

				log.Debugf("Removing registry %s...", r.Name)
				err := os.RemoveAll(r.Path)
				if err != nil {
					fatal.ExitErrf(err, "Failed removing registry %s", r.Name)
				}
			}

			log.Infoln("Removing any remaining registry directories...")
			err := os.RemoveAll(config.RegistriesPath())
			if err != nil {
				fatal.ExitErr(err, "Failed removing registries.")
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
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeRegistries, "registries", false, "nuke all registries")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeAll, "all", false, "nuke everything")
}
