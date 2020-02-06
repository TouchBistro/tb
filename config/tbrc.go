package config

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

var tbrc UserConfig

type Playlist struct {
	Extends  string   `yaml:"extends"`
	Services []string `yaml:"services"`
}

type ServiceOverride struct {
	Remote struct {
		Enabled bool   `yaml:"enabled"`
		Tag     string `yaml:"tag"`
	} `yaml:"remote"`
}

type UserConfig struct {
	DebugEnabled bool                       `yaml:"debug"`
	Playlists    map[string]Playlist        `yaml:"playlists"`
	Overrides    map[string]ServiceOverride `yaml:"overrides"`
}

func InitRC() error {
	rcPath := filepath.Join(os.Getenv("HOME"), ".tbrc.yml")

	// Create default tbrc if it doesn't exist
	if !util.FileOrDirExists(rcPath) {
		err := util.CreateFile(rcPath, rcTemplate)
		if err != nil {
			return errors.Wrapf(err, "couldn't create default tbrc at %s", rcPath)
		}
	}

	err := util.ReadYaml(rcPath, &tbrc)
	return errors.Wrapf(err, "couldn't read yaml file at %s", rcPath)
}

func TBRC() *UserConfig {
	return &tbrc
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
