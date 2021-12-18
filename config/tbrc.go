package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/registry"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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
	if !file.Exists(tbrcPath) {
		err := ioutil.WriteFile(tbrcPath, []byte(rcTemplate), 0o644)
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

	logLevel := log.InfoLevel
	if tbrc.DebugEnabled {
		logLevel = log.DebugLevel
		fatal.PrintDetailedError(true)
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
		registryName, _, err := resource.ParseName(name)
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

	tbrcPath := filepath.Join(os.Getenv("HOME"), tbrcName)
	f, err := os.OpenFile(tbrcPath, os.O_RDWR, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", tbrcPath)
	}
	defer f.Close()

	// Decode into a Node so we can manipulate the contents while
	// preserving comments and formatting
	tbrcDocumentNode := &yaml.Node{}
	err = yaml.NewDecoder(f).Decode(tbrcDocumentNode)
	if err != nil {
		return errors.Wrapf(err, "couldn't read yaml file at %s", tbrcPath)
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
	registriesNode := findNode(tbrcDocumentNode, "registries")

	// registries key doesn't exist
	// need to add it at the end of the document
	if registriesNode == nil {
		// Get top level map node
		contentLen := len(tbrcDocumentNode.Content)
		if contentLen != 1 {
			// This shouldn't happen so don't worry about it right now
			// If this becomes an issue we can better handle this later
			return errors.Wrapf(err, "tbrc document has invalid content length %d", contentLen)
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
	_, err = f.Seek(0, 0)
	if err != nil {
		return errors.Wrapf(err, "failed to seek start of file %s", tbrcPath)
	}

	err = f.Truncate(0)
	if err != nil {
		return errors.Wrapf(err, "failed to truncate file %s", tbrcPath)
	}

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	err = encoder.Encode(tbrcDocumentNode)
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", tbrcName)
	}

	return nil
}

func findNode(node *yaml.Node, key string) *yaml.Node {
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
			foundNode := findNode(n, key)
			if foundNode != nil {
				return foundNode
			}
		}
	}

	return nil
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
