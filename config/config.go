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
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/engine"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/integrations/git"
	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/TouchBistro/tb/internal/util"
	"github.com/TouchBistro/tb/registry"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
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

// Config represents a tbrc config file used to provide custom configuration for a user.
type Config struct {
	// Triple state bools suck but we need this so we can tell if the user set it explicitly.
	// TODO(@cszatmary): Remove this when we do a breaking change.
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
// If homedir is empty, it will be resolved from the environment.
// If a config file does not exist in homedir, one will be created.
func Read(homedir string) (Config, error) {
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
	return config, nil
}

type InitOptions struct {
	// If true, Init will load services and playlists from registries.
	// If false, no services or playlists will be available in the returned Engine instance.
	LoadServices bool
	// If true, Init will load apps from registries.
	// If false, no apps will be available in the returned Engine instance.
	LoadApps bool
	// If true, registries will be updated before being read, otherwise the existing version
	// will be read. Missing registries will always be cloned regardless of the value of this field.
	UpdateRegistries bool
}

// Init takes a config and initializes an engine.Engine for performing tb operations.
// opts can be used to customize how this Engine instance is constructed.
//
// Init will read all registries specified in config and use that to produce a list of
// services, playlists, and apps for the Engine to manage.
func Init(ctx context.Context, config Config, opts InitOptions) (*engine.Engine, error) {
	const op = errors.Op("config.Init")

	// Retrieve the user's home directory since it will be required for various operations.
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

	// We need at least one registry otherwise tb is pretty useless so let the user know.
	if len(config.Registries) == 0 {
		return nil, errors.New(errkind.Invalid, "no registries defined", op)
	}

	// Validate and normalize all registries.
	tracker := progress.TrackerFromContext(ctx)
	for i, r := range config.Registries {
		// Resolve true registry path
		if r.LocalPath != "" {
			// Remind people they are using a local version in case they forgot
			tracker.Infof("❗ Using a local version of the %s registry ❗", color.Cyan(r.Name))

			// Local paths can be prefixed with ~ for convenience
			if strings.HasPrefix(r.LocalPath, "~") {
				r.Path = filepath.Join(homedir, strings.TrimPrefix(r.LocalPath, "~"))
			} else {
				path, err := filepath.Abs(r.LocalPath)
				if err != nil {
					return nil, errors.Wrap(err, errors.Meta{
						Kind:   errkind.IO,
						Reason: fmt.Sprintf("failed to resolve absolute path to local registry %s", r.Name),
						Op:     op,
					})
				}
				r.Path = path
			}
		} else {
			// If not local, the path will be where the registry is/will be cloned.
			r.Path = filepath.Join(tbRoot, registriesDir, r.Name)
		}
		config.Registries[i] = r
	}

	// Go through each registry and make sure it is ready for use.
	err = progress.RunParallel(ctx, progress.RunParallelOptions{
		Message: "Cloning/updating registries",
		Count:   len(config.Registries),
	}, func(ctx context.Context, i int) error {
		r := config.Registries[i]
		if r.LocalPath != "" {
			// User's are responsible for local registries so we just assume they are good to go.
			tracker.Debugf("Skipping local registry %s", r.Name)
			return nil
		}

		// Clone if missing, otherwise we can't actually use it which would be pretty useless.
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
		if err := gitClient.Pull(ctx, r.Path); err != nil {
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

	// Validate service overrides.
	// Make sure all overrides use the full name of the service. This is necessary so
	// we can determine which service to override in which registry without ambiguity.
	for name := range config.Overrides {
		registryName, _, err := resource.ParseName(name)
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{
				Reason: fmt.Sprintf("invalid service name to override %s", name),
				Op:     op,
			})
		}
		if registryName == "" {
			return nil, errors.New(
				errkind.Invalid,
				fmt.Sprintf("invalid service override %s, overrides must use the full name <registry>/<service>", name),
				op,
			)
		}
	}

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

	// Only try loading devices if we are on macOS.
	// This way other commands like 'tb app list' can still function on linux.
	var deviceList simulator.DeviceList
	if opts.LoadApps && util.IsMacOS {
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

// AddRegistry adds the registry to the config file located in the given home directory.
// If homedir is empty, it will be resolved from the environment.
// If a config file does not exist in homedir, one will be created and the registry
// will then be added to it.
//
// If the registry already exists in the config file, ErrRegistryExists will be returned.
func AddRegistry(registryName, homedir string) error {
	const op = errors.Op("config.AddRegistry")
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

	// Check if registry already added
	// We need to read the config first so we can look at the registries
	config, err := Read(homedir)
	if err != nil {
		return errors.Wrap(err, errors.Meta{Op: op})
	}
	for _, r := range config.Registries {
		if r.Name == registryName {
			return ErrRegistryExists
		}
	}

	// Registry does not exist, we need to add it.
	// In order to do this we need to read the config file again but
	// unmarshal it unto a yaml node. This is necessary in order to
	// preserve comments in the file. Do this since tbrc is meant to be
	// a human-editable file so we want to allow comments and now
	// mess it up every time a registry is added.

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
	// preserving comments and ordering
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
