package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/registry"
	"github.com/TouchBistro/tb/service"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const tbrcName = ".tbrc.yml"

type userConfig struct {
	DebugEnabled        bool                               `yaml:"debug"`
	ExperimentalEnabled bool                               `yaml:"experimental"`
	Playlists           map[string]playlist.Playlist       `yaml:"playlists"`
	Overrides           map[string]service.ServiceOverride `yaml:"overrides"`
	Registries          []registry.Registry                `yaml:"registries"`
}

/* Getters for private & computed vars */

func IsExperimentalEnabled() bool {
	return tbrc.ExperimentalEnabled
}

func Registries() []registry.Registry {
	return tbrc.Registries
}

func LoadTBRC() error {
	tbrcPath := filepath.Join(os.Getenv("HOME"), tbrcName)

	// Create default tbrc if it doesn't exist
	if !file.FileOrDirExists(tbrcPath) {
		err := file.CreateFile(tbrcPath, rcTemplate)
		if err != nil {
			return errors.Wrapf(err, "couldn't create default tbrc at %s", tbrcPath)
		}
	}

	f, err := os.Open(tbrcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", tbrcPath)
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&tbrc)
	if err != nil {
		return errors.Wrapf(err, "couldn't read yaml file at %s", tbrcPath)
	}

	var logLevel log.Level
	if tbrc.DebugEnabled {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
		fatal.ShowStackTraces = false
	}

	log.SetLevel(logLevel)
	log.SetFormatter(&log.TextFormatter{
		// TODO: Remove the log level - its quite ugly
		DisableTimestamp: true,
	})

	if IsExperimentalEnabled() {
		log.Infoln(color.Yellow("🚧 Experimental mode enabled 🚧"))
		log.Infoln(color.Yellow("If you find any bugs please report them in an issue: https://github.com/TouchBistro/tb/issues"))
	}

	// Resolve registry paths
	for i, r := range tbrc.Registries {
		isLocal := r.LocalPath != ""

		// Set true path for usage later
		if isLocal {
			// Remind people they are using a local version in case they forgot
			log.Infof("❗ Using a local version of the %s registry ❗", color.Cyan(r.Name))

			// Local paths can be prefixed with ~ for convenience
			if strings.HasPrefix(r.LocalPath, "~") {
				r.Path = filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(r.LocalPath, "~"))
			} else {
				path, err := filepath.Abs(r.LocalPath)
				if err != nil {
					return errors.Wrapf(err, "failed to resolve absolute path to local registry %s", r.Name)
				}

				r.Path = path
			}
		} else {
			r.Path = filepath.Join(RegistriesPath(), r.Name)
		}

		tbrc.Registries[i] = r
	}

	return nil
}

const rcTemplate = `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  # - name: ExampleOrg/tb-registry
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  # db:
    # services:
      # - postgres
  # dev-tools:
    # extends: db
    # services:
      # - localstack
# Override service configuration
overrides:
  # venue-admin-frontend
    # remote:
    # enabled: true
    # tag: feat/new-menu
`
