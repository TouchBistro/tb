package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/git"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var ErrRecipeExists = errors.New("recipe already exists")

// File names in recipe
const (
	appsFileName      = "apps.yml"
	playlistsFileName = "playlists.yml"
	servicesFileName  = "services.yml"
	staticDirName     = "static"
)

type Recipe struct {
	Name      string `yaml:"name"`
	LocalPath string `yaml:"localPath,omitempty"`
	Path      string `yaml:"-"`
}

func recipesPath() string {
	return filepath.Join(tbRoot, "recipes")
}

func recipeNameParts(name string) (string, string, error) {
	// Full form of item name in a recipe is
	// <org>/<repo>/<item> where an item is a service, playlist or app
	regex := regexp.MustCompile(`^(?:([\w-]+\/[\w-]+)\/)?([\w-]+)$`)
	matches := regex.FindStringSubmatch(name)

	if len(matches) == 0 {
		return "", "", errors.Errorf("%s is not a valid tb item name. Format is <org>/<repo>/<item>", name)
	} else if len(matches) == 2 {
		// No recipe name provided
		return "", matches[1], nil
	}

	return matches[1], matches[2], nil
}

func joinNameParts(recipeName, itemName string) string {
	// Doing this to abstract the notion of how the names are combined to form the full name
	if recipeName == "" {
		return itemName
	}

	return fmt.Sprintf("%s/%s", recipeName, itemName)
}

func resolveRecipe(r Recipe, shouldUpdate bool) (Recipe, error) {
	var path string
	isLocal := r.LocalPath != ""
	if isLocal {
		path = filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(r.LocalPath, "~"))
	} else {
		path = filepath.Join(recipesPath(), r.Name)
	}
	// Set true path for usage later
	r.Path = path

	// Clone if missing and not local
	if !isLocal && !file.FileOrDirExists(path) {
		log.Debugf("Recipe %s is missing, cloning...", r.Name)
		err := git.Clone(r.Name, recipesPath())
		if err != nil {
			return r, errors.Wrapf(err, "failed to clone recipe to %s", path)
		}
	}

	if !isLocal && shouldUpdate {
		log.Debugf("Updating recipe %s...", r.Name)
		err := git.Pull(r.Name, recipesPath())
		if err != nil {
			return r, errors.Wrapf(err, "failed to update recipe %s", r.Name)
		}
	}

	return r, nil
}

func readRecipeServices(r Recipe) (RecipeServiceConfig, PlaylistMap, error) {
	servicesPath := filepath.Join(r.Path, servicesFileName)
	playlistsPath := filepath.Join(r.Path, playlistsFileName)

	log.Debugf("Reading services from recipe %s", r.Name)
	serviceConf := RecipeServiceConfig{}

	// Read services.yml
	if !file.FileOrDirExists(servicesPath) {
		log.Debugf("recipe %s has no %s", r.Name, servicesFileName)
	} else {
		f, err := os.Open(servicesPath)
		if err != nil {
			return serviceConf, nil, errors.Wrapf(err, "failed to open file %s", servicesPath)
		}
		defer f.Close()

		err = yaml.NewDecoder(f).Decode(&serviceConf)
		if err != nil {
			return serviceConf, nil, errors.Wrapf(err, "failed to read %s in recipe %s", servicesFileName, r.Name)
		}
	}

	log.Debugf("Reading playlists from recipe %s", r.Name)
	playlists := make(PlaylistMap)

	// Read playlists.yml
	if !file.FileOrDirExists(playlistsPath) {
		log.Debugf("recipe %s has no %s", r.Name, playlistsFileName)
	} else {
		f, err := os.Open(playlistsPath)
		if err != nil {
			return serviceConf, nil, errors.Wrapf(err, "failed to open file %s", playlistsPath)
		}
		defer f.Close()

		err = yaml.NewDecoder(f).Decode(&playlists)
		if err != nil {
			return serviceConf, nil, errors.Wrapf(err, "failed to read %s in recipe %s", playlistsFileName, r.Name)
		}
	}

	return serviceConf, playlists, nil
}

func readRecipeApps(r Recipe) (RecipeAppConfig, error) {
	appsPath := filepath.Join(r.Path, appsFileName)

	log.Debugf("Reading apps from recipe %s", r.Name)
	appConf := RecipeAppConfig{}

	if !file.FileOrDirExists(appsPath) {
		log.Debugf("recipe %s has no %s", r.Name, appsFileName)
	} else {
		f, err := os.Open(appsPath)
		if err != nil {
			return appConf, errors.Wrapf(err, "failed to open file %s", appsPath)
		}
		defer f.Close()

		err = yaml.NewDecoder(f).Decode(&appConf)
		if err != nil {
			return appConf, errors.Wrapf(err, "failed to read %s in recipe %s", appsFileName, r.Name)
		}
	}

	return appConf, nil
}

func mergeServiceConfigs(serviceConfigMap map[string]RecipeServiceConfig, playlistsMap map[string]PlaylistMap) (ServiceConfig, map[string][]Playlist, error) {
	mergedServiceConfig := ServiceConfig{}
	mergedPlaylists := make(map[string][]Playlist)

	for recipeName, serviceConf := range serviceConfigMap {
		mergedServiceConfig.BaseImages = append(mergedServiceConfig.BaseImages, serviceConf.Global.BaseImages...)
		mergedServiceConfig.LoginStrategies = append(mergedServiceConfig.LoginStrategies, serviceConf.Global.LoginStrategies...)

		// Parse services to verify required properties exist and expand variables
		services, err := parseServices(serviceConf)
		if err != nil {
			return mergedServiceConfig, nil, errors.Wrapf(err, "failed to parse services from recipe %s", recipeName)
		}

		// Add all services to the merged config and set their recipe name for lookup later
		for serviceName, s := range services {
			s.RecipeName = recipeName
			mergedServiceConfig.Services[serviceName] = append(mergedServiceConfig.Services[serviceName], s)
		}
	}

	for recipeName, playlists := range playlistsMap {
		for playlistName, p := range playlists {
			if p.Extends != "" {
				// Add recipe name to playlist being extended if it's missing
				r, name, err := recipeNameParts(p.Extends)
				if err != nil {
					return mergedServiceConfig, nil, errors.Wrapf(err, "failed to resolve full name of playlist %s", p.Extends)
				}

				if r == "" {
					p.Extends = joinNameParts(recipeName, name)
				}
			}

			// Add recipe name to services if it's missing
			serviceList := make([]string, len(p.Services))
			for i, s := range p.Services {
				r, name, err := recipeNameParts(s)
				if err != nil {
					return mergedServiceConfig, nil, errors.Wrapf(err, "failed to resolve full name of service %s in playlist %s", s, playlistName)
				}

				if r == "" {
					r = recipeName
				}

				serviceList[i] = joinNameParts(r, name)
			}

			p.Services = serviceList
			p.RecipeName = recipeName
			mergedPlaylists[playlistName] = append(mergedPlaylists[playlistName], p)
		}
	}

	return mergedServiceConfig, mergedPlaylists, nil
}

func mergeAppConfigs(appConfigMap map[string]RecipeAppConfig) (AppConfig, error) {
	mergedAppConfig := AppConfig{}

	for recipeName, appConf := range appConfigMap {
		// TODO figure out s3 config

		for appName, app := range appConf.IOSApps {
			app.RecipeName = recipeName
			mergedAppConfig.IOSApps[appName] = append(mergedAppConfig.IOSApps[appName], app)
		}
	}

	return mergedAppConfig, nil
}

func AddRecipe(recipeName string) error {
	// Check if recipe is already added
	for _, recipe := range tbrc.Recipes {
		if recipe.Name == recipeName {
			return ErrRecipeExists
		}
	}

	// Check recipe name is valid, i.e. org/name
	regex := regexp.MustCompile(`^[\w-]+\/[\w-]+$`)
	if !regex.MatchString(recipeName) {
		return errors.Errorf("%s is not a valid recipe name", recipeName)
	}

	err := git.Clone(recipeName, recipesPath())
	if err != nil {
		return errors.Wrapf(err, "failed to clone recipe %s", recipeName)
	}

	tbrc.Recipes = append(tbrc.Recipes, Recipe{
		Name: recipeName,
	})

	err = saveTBRC(tbrc)
	return errors.Wrapf(err, "failed to save %s", tbrcFileName)
}
