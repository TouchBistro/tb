package config

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const tbrcName = ".tbrc.yml"

type Playlist struct {
	Extends  string   `yaml:"extends"`
	Services []string `yaml:"services"`
}

type BuildOverride struct {
	Command string `yaml:"command"`
	Target  string `yaml:"target"`
}

type RemoteOverride struct {
	Command string `yaml:"command"`
	Enabled bool   `yaml:"enabled"`
	Tag     string `yaml:"tag"`
}

type ServiceOverride struct {
	Build   BuildOverride     `yaml:"build"`
	EnvVars map[string]string `yaml:"envVars"`
	PreRun  string            `yaml:"preRun"`
	Remote  RemoteOverride    `yaml:"remote"`
}

type userConfig struct {
	DebugEnabled        bool                       `yaml:"debug"`
	ExperimentalEnabled bool                       `yaml:"experimental"`
	Playlists           map[string]Playlist        `yaml:"playlists"`
	Overrides           map[string]ServiceOverride `yaml:"overrides"`
}

/* Getters for private & computed vars */

func IsExperimentalEnabled() bool {
	return tbrc.ExperimentalEnabled
}

// TODO remove this once recipe stuff is implemented
// This is a temp hack so tb list still works
func CustomPlaylists() map[string]Playlist {
	return tbrc.Playlists
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

	return nil
}

const rcTemplate = `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
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
