package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/login"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Package state for storing config info
var tbrc UserConfig
var serviceConfig ServiceConfig
var playlists map[string][]Playlist
var appConfig AppConfig
var tbRoot string

const lazydockerConfig = `
reporting: "off"
gui:
  wrapMainPanel: true
update:
  dockerRefreshInterval: 2000ms`

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

func Services() ServiceListMap {
	return serviceConfig.Services
}

func LoginStategies() ([]login.LoginStrategy, error) {
	s, err := login.ParseStrategies(serviceConfig.LoginStrategies)
	return s, errors.Wrap(err, "Failed to parse login strategies")
}

func BaseImages() []string {
	return serviceConfig.BaseImages
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

	rcFile, err := os.Open(rcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", rcPath)
	}
	defer rcFile.Close()

	err = yaml.NewDecoder(rcFile).Decode(&tbrc)
	if err != nil {
		return errors.Wrapf(err, "failed to decode %s", tbrcFileName)
	}

	// Configure anything debug related
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

	if !tbrc.ExperimentalMode {
		log.Debugln("Using legacy config init")
		return legacyInit()
	}

	// Create default lazydocker config if it doesn't exist
	ldDirPath := filepath.Join(os.Getenv("HOME"), "Library/Application Support/jesseduffield/lazydocker")
	err = os.MkdirAll(ldDirPath, 0766)
	if err != nil {
		return errors.Wrapf(err, "failed to create lazydocker config directory %s", ldDirPath)
	}

	ldConfigPath := filepath.Join(ldDirPath, "lazydocker.yml")
	if !file.FileOrDirExists(ldConfigPath) {
		err = ioutil.WriteFile(ldConfigPath, []byte(lazydockerConfig), 0644)
		if err != nil {
			return errors.Wrap(err, "failed to create lazydocker config file")
		}
	}

	// THE RECIPE ZONE

	log.Debugln("Resolving recipes...")
	// Make sure recipes exist and resolve path
	for i, r := range tbrc.Recipes {
		resolvedRecipe, err := resolveRecipe(r, opts.UpdateRecipes)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve recipe %s", r.Name)
		}
		tbrc.Recipes[i] = resolvedRecipe
	}

	if opts.LoadServices {
		log.Debugln("Loading services...")

		serviceConfigMap := make(map[string]RecipeServiceConfig)
		playlistsMap := make(map[string]PlaylistMap)
		for _, r := range tbrc.Recipes {
			log.Debugf("Reading services from recipe %s", r)

			s, p, err := readRecipeServices(r)
			if err != nil {
				return errors.Wrapf(err, "failed to read services for recipe %s", r.Name)
			}
			serviceConfigMap[r.Name] = s
			playlistsMap[r.Name] = p
		}

		log.Debugln("Merging services...")
		serviceConfig, playlists, err = mergeServiceConfigs(serviceConfigMap, playlistsMap)
		if err != nil {
			return errors.Wrap(err, "failed to merge services and playlists")
		}

		log.Debugln("Applying overrides to services...")
		serviceConfig.Services, err = applyOverrides(serviceConfig.Services, tbrc.Overrides)

		// Create docker-compose.yml
		log.Debugln("Generating docker-compose.yml file...")

		composePath := filepath.Join(tbRoot, dockerComposePath)
		file, err := os.OpenFile(composePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", composePath)
		}
		defer file.Close()

		err = CreateComposeFile(serviceConfig.Services.ServiceMap(), file)
		if err != nil {
			return errors.Wrap(err, "failed to generated docker-compose file")
		}
		log.Debugln("Successfully generated docker-compose.yml")
	}

	if opts.LoadApps {
		log.Debugln("Loading apps...")

		appConfigMap := make(map[string]RecipeAppConfig)
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

	return nil
}
