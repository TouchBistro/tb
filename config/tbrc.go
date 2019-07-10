package config

import (
	"fmt"
	"os"

	"github.com/TouchBistro/tb/util"
)

var tbrc UserConfig

type Playlist struct {
	Extends  string   `yaml:"extends"`
	Services []string `yaml:"services"`
}

type UserConfig struct {
	LogLevel  string              `yaml:"log-level"`
	Playlists map[string]Playlist `yaml:"playlists"`
}

func InitRC() error {
	rcPath := fmt.Sprintf("%s/.tbrc.yml", os.Getenv("HOME"))

	// Create default tbrc if it doesn't exist
	if !util.FileOrDirExists(rcPath) {
		util.CreateFile(rcPath, rcTemplate)
	}

	err := util.ReadYaml(rcPath, &tbrc)
	return err
}

func TBRC() *UserConfig {
	return &tbrc
}

const rcTemplate = `# Only print logs equal or high to this level
log-level: "info"
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
`
