package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/login"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type PlaylistMap map[string]Playlist

// Package state for storing config info
var tbrc UserConfig
var serviceConfig ServiceConfig
var playlists PlaylistMap
var appConfig AppConfig
var tbRoot string

// TODO legacy - remove these
const (
	servicesPath             = "services.yml"
	playlistPath             = "playlists.yml"
	dockerComposePath        = "docker-compose.yml"
	localstackEntrypointPath = "localstack-entrypoint.sh"
	lazydockerConfigPath     = "lazydocker.yml"
)

type InitOptions struct {
	LoadServices  bool
	LoadApps      bool
	UpdateRecipes bool
}

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

func setupEnv(rcPath string) error {
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

	// Create default tbrc if it doesn't exist
	if !file.FileOrDirExists(rcPath) {
		err := file.CreateFile(rcPath, rcTemplate)
		if err != nil {
			return errors.Wrapf(err, "couldn't create default tbrc at %s", rcPath)
		}
	}

	return nil
}

func Init(opts InitOptions) error {
	// Setup env and load .tbrc.yml
	rcPath := filepath.Join(os.Getenv("HOME"), tbrcFileName)
	setupEnv(rcPath)

	log.Debugf("Loading %s...", tbrcFileName)
	rcFile, err := os.Open(rcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", rcPath)
	}
	defer rcFile.Close()

	err = yaml.NewDecoder(rcFile).Decode(&tbrc)
	if err != nil {
		return errors.Wrapf(err, "failed to decode %s", tbrcFileName)
	}

	if !tbrc.Experimental.UseRecipes {
		log.Debugln("Using legacy config init")
		return legacyInit()
	}

	log.Debugf("Resolving recipes...")
	// Make sure recipes exist and resolve path
	for i, r := range tbrc.Recipes {
		resolvedRecipe, err := resolveRecipe(r, opts.UpdateRecipes)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve recipe %s", r.Name)
		}
		tbrc.Recipes[i] = resolvedRecipe
	}

	if opts.LoadServices {
		serviceConfigMap := make(map[string]ServiceConfig)
		playlistsMap := make(map[string]PlaylistMap)
		for _, r := range tbrc.Recipes {
			s, p, err := readRecipeServices(r)
			if err != nil {
				return errors.Wrapf(err, "failed to read services for recipe %s", r.Name)
			}
			serviceConfigMap[r.Name] = s
			playlistsMap[r.Name] = p
		}

		serviceConfig, playlists, err = mergeServiceConfigs(serviceConfigMap, playlistsMap)
		if err != nil {
			return errors.Wrap(err, "failed to merge services and playlists")
		}
	}

	if opts.LoadApps {
		appConfigMap := make(map[string]AppConfig)
		for _, r := range tbrc.Recipes {
			a, err := readRecipeApps(r)
			if err != nil {
				return errors.Wrapf(err, "failed to read apps for recipe %s", r.Name)
			}
			appConfigMap[r.Name] = a
		}

		appConfig, err = mergeAppConfigs(appConfigMap)
		if err != nil {
			return errors.Wrap(err, "failed to merge apps")
		}
	}
	log.Debugln("Successfully generated docker-compose.yml")

	// TODO Dump lazydocker config if it doesn't exist

	return nil
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
