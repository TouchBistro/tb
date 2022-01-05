package config

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/engine"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/integrations/git"
	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/TouchBistro/tb/registry"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// ErrRegistryExists indicates that the registry being added already exists.
var ErrRegistryExists errors.String = "registry already exists"

const (
	tbrcName      = ".tbrc.yml"
	rootDir       = ".tb"
	registriesDir = "registries"
)

//go:embed template.yml
var tbrcTemplate []byte

// Some leftover global state.
// TODO(@cszatmary): figure out a way to clean this up.

var registries []registry.Registry

// Config represents a tbrc config file used to provide custom configuration for a user.
type Config struct {
	Debug            *bool                              `yaml:"debug"`
	ExperimentalMode bool                               `yaml:"experimental"`
	Playlists        map[string]playlist.Playlist       `yaml:"playlists"`
	Overrides        map[string]service.ServiceOverride `yaml:"overrides"`
	Registries       []registry.Registry                `yaml:"registries"`
}

// NOTE: This is deprecated and is only here for backwards compatibility.
func (c Config) DebugEnabled() bool {
	if c.Debug == nil {
		return false
	}
	return *c.Debug
}

// Read reads the config file using the given home directory.
// If a config file does not exist in homedir, one will be created.
func Load(homedir string) (Config, error) {
	const op = errors.Op("config.Read")
	if homedir == "" {
		var err error
		homedir, err = os.UserHomeDir()
		if err != nil {
			return Config{}, errors.Wrap(err, errors.Meta{
				Kind:   errkind.Internal,
				Reason: "unable to find user home directory",
				Op:     op,
			})
		}
	}
	configPath := filepath.Join(homedir, tbrcName)

	// Create default tbrc if it doesn't exist
	if !file.Exists(configPath) {
		if err := os.WriteFile(configPath, tbrcTemplate, 0o644); err != nil {
			return Config{}, errors.Wrap(err, errors.Meta{
				Kind:   errkind.IO,
				Reason: fmt.Sprintf("couldn't create default tbrc at %s", configPath),
				Op:     op,
			})
		}
	}

	f, err := os.Open(configPath)
	if err != nil {
		return Config{}, errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to open file %s", configPath),
			Op:     op,
		})
	}
	defer f.Close()

	var config Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return config, errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("couldn't read yaml file at %s", configPath),
			Op:     op,
		})
	}

	// Triple state bools suck but we need this so we can tell if the user set it explicitly.
	// TODO(@cszatmary): Remove this when we do a breaking change.
	if config.Debug != nil {
		if *config.Debug {
			logrus.SetLevel(logrus.DebugLevel)
			logrus.SetFormatter(&logrus.TextFormatter{
				DisableTimestamp: true,
				ForceColors:      true,
			})
			fatal.PrintDetailedError(true)
		}
		// This prints a warning sign
		logrus.Warn("\u26A0\uFE0F  Using the 'debug' field in tbrc.yml is deprecated. Use the '--verbose' or '-v' flag instead.")
	}

	if config.ExperimentalMode {
		logrus.Info(color.Yellow("🚧 Experimental mode enabled 🚧"))
		logrus.Info(color.Yellow("If you find any bugs please report them in an issue: https://github.com/TouchBistro/tb/issues"))
	}

	// Resolve registry paths
	for i, r := range config.Registries {
		// Set true path for usage later
		if r.LocalPath != "" {
			// Remind people they are using a local version in case they forgot
			logrus.Infof("❗ Using a local version of the %s registry ❗", color.Cyan(r.Name))

			// Local paths can be prefixed with ~ for convenience
			if strings.HasPrefix(r.LocalPath, "~") {
				r.Path = filepath.Join(homedir, strings.TrimPrefix(r.LocalPath, "~"))
			} else {
				path, err := filepath.Abs(r.LocalPath)
				if err != nil {
					return config, errors.Wrap(err, errors.Meta{
						Kind:   errkind.IO,
						Reason: fmt.Sprintf("failed to resolve absolute path to local registry %s", r.Name),
						Op:     op,
					})
				}
				r.Path = path
			}
		} else {
			r.Path = filepath.Join(homedir, registriesDir, r.Name)
		}
		config.Registries[i] = r
	}
	registries = config.Registries

	// Make sure all overrides use the full name of the service
	for name := range config.Overrides {
		registryName, _, err := resource.ParseName(name)
		if err != nil {
			return config, errors.Wrap(err, errors.Meta{
				Reason: fmt.Sprintf("invalid service name to override %s", name),
				Op:     op,
			})
		}
		if registryName == "" {
			return config, errors.New(
				errkind.Invalid,
				fmt.Sprintf("invalid service override %s, overrides must use the full name <registry>/<service>", name),
				op,
			)
		}
	}
	return config, nil
}

type InitOptions struct {
	LoadServices     bool
	LoadApps         bool
	UpdateRegistries bool
}

func Init(ctx context.Context, config Config, opts InitOptions) (*engine.Engine, error) {
	const op = errors.Op("config.Init")
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Kind:   errkind.Internal,
			Reason: "unable to find user home directory",
			Op:     op,
		})
	}
	tbRoot := filepath.Join(homedir, rootDir)
	// Create ~/.tb directory if it doesn't exist
	if err := os.MkdirAll(tbRoot, 0o755); err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to create tb root directory at %s", tbRoot),
			Op:     op,
		})
	}

	// Handle registries
	if len(config.Registries) == 0 {
		return nil, errors.New(errkind.Invalid, "no registries defined", op)
	}
	err = progress.RunParallel(ctx, progress.RunParallelOptions{
		Message: "Cloning/updating registries",
		Count:   len(config.Registries),
	}, func(ctx context.Context, i int) error {
		r := config.Registries[i]
		tracker := progress.TrackerFromContext(ctx)
		if r.LocalPath != "" {
			tracker.Debugf("Skipping local registry %s", r.Name)
			return nil
		}

		// Clone if missing
		gitClient := git.New()
		if !file.Exists(r.Path) {
			tracker.Debugf("Registry %s is missing, cloning", r.Name)
			if err := gitClient.Clone(ctx, r.Name, r.Path); err != nil {
				return errors.Wrap(err, errors.Meta{
					Reason: fmt.Sprintf("failed to clone registry %s", r.Name),
					Op:     op,
				})
			}
			tracker.Debugf("Finished cloning registry %s", r.Name)
			return nil
		}
		if !opts.UpdateRegistries {
			return nil
		}

		tracker.Debugf("Updating registry %s", r.Name)
		path := filepath.Join(tbRoot, registriesDir, r.Name)
		if err := gitClient.Pull(ctx, path); err != nil {
			return errors.Wrap(err, errors.Meta{
				Reason: fmt.Sprintf("failed to update registry %s", r.Name),
				Op:     op,
			})
		}
		tracker.Debugf("Finished cloning/pulling registry %s", r.Name)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Reason: "failed to clone/update registries",
			Op:     op,
		})
	}

	tracker := progress.TrackerFromContext(ctx)
	registryResult, err := registry.ReadAll(config.Registries, registry.ReadAllOptions{
		ReadServices: opts.LoadServices,
		ReadApps:     opts.LoadApps,
		HomeDir:      homedir,
		RootPath:     tbRoot,
		ReposPath:    filepath.Join(tbRoot, "repos"),
		Overrides:    config.Overrides,
		Logger:       tracker,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Reason: "failed to read registries",
			Op:     op,
		})
	}

	if opts.LoadServices {
		// Add custom playlists
		for n, p := range config.Playlists {
			p.Name = n
			registryResult.Playlists.SetCustom(p)
		}

		// Create docker-compose.yml
		tracker.Debug("Generating docker-compose.yml file")
		composePath := filepath.Join(tbRoot, docker.ComposeFilename)
		f, err := os.OpenFile(composePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{
				Kind:   errkind.IO,
				Reason: fmt.Sprintf("failed to open file %s", composePath),
				Op:     op,
			})
		}
		defer f.Close()

		const header = "# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY\n\n"
		if _, err := io.WriteString(f, header); err != nil {
			return nil, errors.Wrap(err, errors.Meta{
				Kind:   errkind.IO,
				Reason: "failed to write header comment to docker-compose yaml file",
				Op:     op,
			})
		}

		composeConfig := service.ComposeConfig(registryResult.Services)
		if err := yaml.NewEncoder(f).Encode(composeConfig); err != nil {
			return nil, errors.Wrap(err, errors.Meta{
				Kind:   errkind.IO,
				Reason: "failed to encode docker-compose struct to yaml",
				Op:     op,
			})
		}
		tracker.Debug("Successfully generated docker-compose.yml")
	}
	var deviceList simulator.DeviceList
	if opts.LoadApps {
		deviceData, err := simulator.ListDevices(ctx)
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{Reason: "failed to get list of simulators", Op: op})
		}
		deviceList, err = simulator.ParseDevices(deviceData)
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{Reason: "failed to parse available simulators", Op: op})
		}
	}

	e, err := engine.New(engine.Options{
		Workdir:         tbRoot,
		Services:        registryResult.Services,
		Playlists:       registryResult.Playlists,
		IOSApps:         registryResult.IOSApps,
		DesktopApps:     registryResult.DesktopApps,
		BaseImages:      registryResult.BaseImages,
		LoginStrategies: registryResult.LoginStrategies,
		DeviceList:      deviceList,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{Reason: "failed to initialize engine", Op: op})
	}
	return e, nil
}

func AddRegistry(registryName, homedir string) error {
	const op = errors.Op("config.AddRegistry")
	// Check if registry already added
	for _, r := range registries {
		if r.Name == registryName {
			return ErrRegistryExists
		}
	}
	if homedir == "" {
		var err error
		homedir, err = os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.Internal,
				Reason: "unable to find user home directory",
				Op:     op,
			})
		}
	}

	tbrcPath := filepath.Join(homedir, tbrcName)
	f, err := os.OpenFile(tbrcPath, os.O_RDWR, 0644)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to open file %s", tbrcPath),
			Op:     op,
		})
	}
	defer f.Close()

	// Decode into a Node so we can manipulate the contents while
	// preserving comments and formatting
	tbrcDocumentNode := &yaml.Node{}
	err = yaml.NewDecoder(f).Decode(tbrcDocumentNode)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("couldn't read yaml file at %s", tbrcPath),
			Op:     op,
		})
	}

	// Create nodes for registry
	nameKeyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: "name",
	}
	nameValueNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: registryName,
	}
	registryNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: []*yaml.Node{nameKeyNode, nameValueNode},
	}

	// Find registries section
	registriesNode := findYamlNode(tbrcDocumentNode, "registries")

	// registries key doesn't exist
	// need to add it at the end of the document
	if registriesNode == nil {
		// Get top level map node
		contentLen := len(tbrcDocumentNode.Content)
		if contentLen != 1 {
			// This shouldn't happen so don't worry about it right now
			// If this becomes an issue we can better handle this later
			return errors.New(
				errkind.Internal,
				fmt.Sprintf("tbrc document has invalid content length %d", contentLen),
				op,
			)
		}

		registriesKeyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "registries",
		}
		registriesNode = &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
		}

		tbrcContentNode := tbrcDocumentNode.Content[0]
		tbrcContentNode.Content = append(tbrcContentNode.Content, registriesKeyNode, registriesNode)
	} else if registriesNode.Tag == "!!null" {
		// !!null means there are no registries defined, i.e. empty key
		// Update the registries node to be a sequence node
		// Then we can just append the new registry to it and
		// treat it the same as if there was already a list of registries
		registriesNode.Kind = yaml.SequenceNode
		registriesNode.Tag = "!!seq"
	}

	// Add new registries at the end of the list
	registriesNode.Content = append(registriesNode.Content, registryNode)

	// Make sure we overwrite the file instead of appending to it
	// Need to go back to the start and truncate it
	if _, err := f.Seek(0, 0); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to seek start of file %s", tbrcPath),
			Op:     op,
		})
	}

	if err := f.Truncate(0); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to truncate file %s", tbrcPath),
			Op:     op,
		})
	}

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	if err := encoder.Encode(tbrcDocumentNode); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to write %s", tbrcPath),
			Op:     op,
		})
	}

	return nil
}

func findYamlNode(node *yaml.Node, key string) *yaml.Node {
	foundKey := false
	for _, n := range node.Content {
		if foundKey {
			return n
		}
		if n.Value == key {
			foundKey = true
			continue
		}
		if len(n.Content) > 0 {
			foundNode := findYamlNode(n, key)
			if foundNode != nil {
				return foundNode
			}
		}
	}
	return nil
}
