package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/registry"
	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const tbrcName = ".tbrc.yml"

var ErrRegistryExists = errors.New("registry already exists")

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

	// Make sure all overrides use the full name of the service
	for name := range tbrc.Overrides {
		registryName, _, err := util.SplitNameParts(name)
		if err != nil {
			return errors.Wrapf(err, "invalid service name to override %s", name)
		}

		if registryName == "" {
			return errors.Errorf("invalid service override %s, overrides must use the full name <registry>/<service>", name)
		}
	}

	return nil
}

func AddRegistry(registryName string) error {
	// Check if registry already added
	for _, r := range tbrc.Registries {
		if r.Name == registryName {
			return ErrRegistryExists
		}
	}

	// Check if registry name is valid, i.e. <org>/<repo>
	nameRegex := regexp.MustCompile(`^[\w-]+\/[\w-]+$`)
	if !nameRegex.MatchString(registryName) {
		return errors.Errorf("%s is not a valid registry name", registryName)
	}

	tbrc.Registries = append(tbrc.Registries, registry.Registry{Name: registryName})

	// Save registries
	// We want the tbrc file to be human readable and writable
	// Unfortantly this means we can't just write the yaml file because the yaml marshaler
	// doesn't preserve comments or file layout and style
	// So we manually write it by finding the registries region and replacing it

	// Marshal registries manually so it's nicely formatted
	builder := strings.Builder{}
	builder.WriteString("registries:\n")
	for _, r := range tbrc.Registries {
		builder.WriteString(fmt.Sprintf("  - name: %s\n", r.Name))
		if r.LocalPath != "" {
			builder.WriteString(fmt.Sprintf("    localPath: %s\n", r.LocalPath))
		}
	}

	// The tbrc file should be small enough that we can just do this the lazy way
	tbrcPath := filepath.Join(os.Getenv("HOME"), tbrcName)
	fileData, err := ioutil.ReadFile(tbrcPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", tbrcPath)
	}
	tbrcStr := string(fileData)

	// try to match the registries field with a regex so we can replace it
	sectionRegex := regexp.MustCompile(`(?m)^registries:\n(?:(?:\s|-).+\n)*`)
	indices := sectionRegex.FindStringIndex(tbrcStr)

	// nil means there was no match so there are no registries defined
	if indices == nil {
		tbrcStr += builder.String()
	} else {
		startIndex := indices[0]
		endIndex := indices[1]
		tbrcStr = tbrcStr[:startIndex] + builder.String() + tbrcStr[endIndex:]
	}

	err = ioutil.WriteFile(tbrcPath, []byte(tbrcStr), 0644)
	return errors.Wrapf(err, "failed to write tbrc file")
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
  # ExampleOrg/tb-registry/venue-admin-frontend:
    # mode: remote
    # remote:
      # tag: feat/new-menu
`
