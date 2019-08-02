package config

import (
	"fmt"
	"os"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

var tbrc UserConfig

type Playlist struct {
	Extends  string   `yaml:"extends"`
	Services []string `yaml:"services"`
}

type UserConfig struct {
	DebugEnabled bool                       `yaml:"debug"`
	Playlists    map[string]Playlist        `yaml:"playlists"`
	Overrides    map[string]ServiceOverride `yaml:"overrides"`
}

func InitRC() error {
	rcPath := fmt.Sprintf("%s/.tbrc.yml", os.Getenv("HOME"))

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
debug: true
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  dev-tools:
    services:
      - localstack
  partner-custom:
    extends: dev-tools
    services:
      - partners-config-service
      - partners-etl-service
# Override service configuration
overrides:
  #mokta:
    #ecr: false
  #venue-admin-frontend:
    #ecr: true
    #ecrTag: master-65392c89be11a78b6caa3924c7af73ca76bcaac7
`
