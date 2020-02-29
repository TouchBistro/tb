package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/login"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var tbrc userConfig
var serviceConfig ServiceConfig
var playlists map[string]Playlist
var tbRoot string

const (
	servicesPath             = "services.yml"
	playlistPath             = "playlists.yml"
	dockerComposePath        = "docker-compose.yml"
	localstackEntrypointPath = "localstack-entrypoint.sh"
	lazydockerConfigPath     = "lazydocker.yml"
)

/* Getters for private & computed vars */

func TBRootPath() string {
	return tbRoot
}

func ReposPath() string {
	return filepath.Join(tbRoot, "repos")
}

func Services() ServiceMap {
	return serviceConfig.Services
}

func Playlists() map[string]Playlist {
	return playlists
}

func LoginStategies() ([]login.LoginStrategy, error) {
	s, err := login.ParseStrategies(serviceConfig.Global.LoginStategies)
	return s, errors.Wrap(err, "Failed to parse login strategies")
}

func BaseImages() []string {
	return serviceConfig.Global.BaseImages
}

/* Private functions */

func setupEnv() error {
	// Set $TB_ROOT so it works in the docker-compose file
	tbRoot = filepath.Join(os.Getenv("HOME"), ".tb")
	os.Setenv("TB_ROOT", tbRoot)

	// Create $TB_ROOT directory if it doesn't exist
	if !file.FileOrDirExists(tbRoot) {
		err := os.Mkdir(tbRoot, 0755)
		if err != nil {
			return errors.Wrapf(err, "failed to create $TB_ROOT directory at %s", tbRoot)
		}
	}
	return nil
}

func Init() error {
	err := setupEnv()
	if err != nil {
		return errors.Wrap(err, "failed to setup $TB_ROOT env")
	}

	return legacyInit()
}

func GetPlaylist(name string, deps map[string]bool) ([]string, error) {
	// TODO: Make this less yolo if Init() wasn't called
	if playlists == nil {
		log.Panic("this is a bug. playlists is not initialised")
	}
	customList := tbrc.Playlists

	// Check custom playlists first
	if playlist, ok := customList[name]; ok {
		// Resolve parent playlist defined in extends
		if playlist.Extends != "" {
			deps[name] = true
			if deps[playlist.Extends] {
				msg := fmt.Sprintf("Circular dependency of services, %s and %s", playlist.Extends, name)
				return []string{}, errors.New(msg)
			}
			parentPlaylist, err := GetPlaylist(playlist.Extends, deps)
			return append(parentPlaylist, playlist.Services...), err
		}

		return playlist.Services, nil
	} else if playlist, ok := playlists[name]; ok {
		if playlist.Extends != "" {
			deps[name] = true
			if deps[playlist.Extends] {
				msg := fmt.Sprintf("Circular dependency of services, %s and %s", playlist.Extends, name)
				return []string{}, errors.New(msg)
			}
			parentPlaylist, err := GetPlaylist(playlist.Extends, deps)
			return append(parentPlaylist, playlist.Services...), err
		}

		return playlist.Services, nil
	}

	return []string{}, nil
}
