package config

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	tbrcFileName = ".tbrc.yml"
)

type ExperimentalOptions struct {
	UseRecipes bool `yaml:"recipes"`
}

type Playlist struct {
	Extends  string   `yaml:"extends"`
	Services []string `yaml:"services"`
}

type ServiceOverride struct {
	Build struct {
		Command string `yaml:"command"`
		Target  string `yaml:"target"`
	} `yaml:"build"`
	EnvVars map[string]string `yaml:"envVars"`
	PreRun  string            `yaml:"preRun"`
	Remote  struct {
		Command string `yaml:"command"`
		Enabled bool   `yaml:"enabled"`
		Tag     string `yaml:"tag,omitempty"`
	} `yaml:"remote,omitempty"`
}

type UserConfig struct {
	DebugEnabled bool                       `yaml:"debug"`
	Experimental ExperimentalOptions        `yaml:"experimental"`
	Recipes      []Recipe                   `yaml:"recipes"`
	Playlists    map[string]Playlist        `yaml:"playlists"`
	Overrides    map[string]ServiceOverride `yaml:"overrides"`
}

func TBRC() *UserConfig {
	return &tbrc
}

func saveTBRC(rc UserConfig) error {
	rcPath := filepath.Join(os.Getenv("HOME"), tbrcFileName)
	rcFile, err := os.OpenFile(rcPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", rcPath)
	}
	defer rcFile.Close()

	err = yaml.NewEncoder(rcFile).Encode(&rc)
	return errors.Wrapf(err, "failed to write %s", tbrcFileName)
}

const rcTemplate = `# Toggle debug mode for more verbose logging
debug: false
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  db:
    services:
      - postgres
  dev-tools:
    extends: db
    services:
      - localstack
# Override service configuration
overrides:
  #mokta:
    #remote:
      #enabled: false
  #venue-admin-frontend:
    #remote:
      #enabled: true
      #tag: tag-name
`
