package cmd

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type nukeOptions struct {
	shouldNukeContainers  bool
	shouldNukeImages      bool
	shouldNukeVolumes     bool
	shouldNukeNetworks    bool
	shouldNukeRepos       bool
	shouldNukeDesktopApps bool
	shouldNukeIOSBuilds   bool
	shouldNukeRegistries  bool
	shouldNukeAll         bool
	shouldSkipGitPull     bool
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
			!nukeOpts.shouldNukeDesktopApps &&
			!nukeOpts.shouldNukeIOSBuilds &&
			!nukeOpts.shouldNukeRegistries &&
			!nukeOpts.shouldNukeAll {
			fatal.Exit("Error: Must specify what to nuke. try tb nuke --help to see all the options.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		s := spinner.New(
			spinner.WithStartMessage("Cleaning up tb resources"),
			spinner.WithStopMessage("Finished cleaning up tb resources"),
			spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
		)
		log.SetOutput(s)
		defer log.SetOutput(os.Stderr)
		s.Start()

		// Make sure containers are stopped before removing docker resources
		// to ensure no weirdness
		if nukeOpts.shouldNukeContainers || nukeOpts.shouldNukeImages ||
			nukeOpts.shouldNukeVolumes || nukeOpts.shouldNukeNetworks ||
			nukeOpts.shouldNukeAll {
			s.UpdateMessage("Stopping running containers")
			err := docker.StopAllContainers()
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed stopping docker containers")
			}
		}

		if nukeOpts.shouldNukeContainers || nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing containers")
			err := docker.RmContainers()
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed removing docker containers")
			}
		}

		if nukeOpts.shouldNukeImages || nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing images")
			_, err := docker.PruneImages()
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed removing docker images.")
			}
		}

		if nukeOpts.shouldNukeNetworks || nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing networks")
			err := docker.RmNetworks()
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed removing docker networks.")
			}
		}

		if nukeOpts.shouldNukeVolumes || nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing volumes")
			err := docker.RmVolumes()
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed removing docker volumes.")
			}
		}

		if nukeOpts.shouldNukeRepos || nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing git repositories")
			var repos []string
			it := config.LoadedServices().Iter()
			for it.HasNext() {
				s := it.Next()
				if s.HasGitRepo() {
					repos = append(repos, s.GitRepo.Name)
				}
			}
			repos = util.UniqueStrings(repos)

			for _, repo := range repos {
				log.Debugf("Removing repo %s", repo)
				repoPath := filepath.Join(config.ReposPath(), repo)
				err := os.RemoveAll(repoPath)
				if err != nil {
					s.Stop()
					fatal.ExitErrf(err, "Failed removing repo %s.", repo)
				}
			}

			log.Debugf("Removing any remaining repo directorys")
			err := os.RemoveAll(config.ReposPath())
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed removing repos.")
			}
		}

		if nukeOpts.shouldNukeDesktopApps || nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing desktop app builds")
			err := os.RemoveAll(config.DesktopAppsPath())
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed removing desktop app builds.")
			}
		}

		if nukeOpts.shouldNukeIOSBuilds || nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing iOS app builds")
			err := os.RemoveAll(config.IOSBuildPath())
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed removing ios builds.")
			}
		}

		if nukeOpts.shouldNukeRegistries || nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing registries")
			for _, r := range config.Registries() {
				// Don't remove local registries
				if r.LocalPath != "" {
					continue
				}

				log.Debugf("Removing registry %s", r.Name)
				err := os.RemoveAll(r.Path)
				if err != nil {
					s.Stop()
					fatal.ExitErrf(err, "Failed removing registry %s", r.Name)
				}
			}

			log.Debug("Removing any remaining registry directories")
			err := os.RemoveAll(config.RegistriesPath())
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "Failed removing registries.")
			}
		}

		if nukeOpts.shouldNukeAll {
			s.UpdateMessage("Removing any remaining files managed by tb")
			rootPath := config.TBRootPath()
			err := os.RemoveAll(rootPath)
			if err != nil {
				s.Stop()
				fatal.ExitErrf(err, "Failed removing files in %s", rootPath)
			}
		}
		s.Stop()
	},
}

func init() {
	rootCmd.AddCommand(nukeCmd)
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeContainers, "containers", false, "nuke all containers")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeImages, "images", false, "nuke all images")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeVolumes, "volumes", false, "nuke all volumes")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeNetworks, "networks", false, "nuke all networks")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeRepos, "repos", false, "nuke all repos")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeDesktopApps, "desktop", false, "nuke all downloaded desktop app builds")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeIOSBuilds, "ios", false, "nuke all downloaded iOS builds")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeRegistries, "registries", false, "nuke all registries")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldNukeAll, "all", false, "nuke everything")
	nukeCmd.Flags().BoolVar(&nukeOpts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")

	err := nukeCmd.Flags().MarkDeprecated("no-git-pull", "it is a no-op")
	if err != nil {
		// MarkDeprecated only errors if the flag name is wrong or the message isn't set
		// which is a programming error, so we wanna blow up
		fatal.ExitErr(err, "Failed to deprecate flag no-git-pull")
	}
}
