package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/TouchBistro/tb/registry"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setup() (string, error) {
	tbrc = userConfig{}
	dir, err := ioutil.TempDir("", "tbrc_test")
	os.Setenv("HOME", dir)
	return dir, err
}

func TestLoadTBRC(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup()
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	tbrcPath := filepath.Join(dir, tbrcName)
	err = ioutil.WriteFile(tbrcPath, []byte("debug: true\nexperimental: true"), 0644)
	if err != nil {
		assert.FailNow("Failed to create tbrc file", err)
	}

	err = LoadTBRC()

	assert.NoError(err)
	assert.True(IsExperimentalEnabled())
	assert.Equal(log.GetLevel(), log.DebugLevel)
}

func TestLoadTBRCDefault(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup()
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	tbrcPath := filepath.Join(dir, tbrcName)

	err = LoadTBRC()

	assert.NoError(err)
	assert.False(IsExperimentalEnabled())
	assert.Equal(log.GetLevel(), log.InfoLevel)
	assert.FileExists(tbrcPath)
}

func TestLoadTBRCInvalidYaml(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup()
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	tbrcPath := filepath.Join(dir, tbrcName)
	err = ioutil.WriteFile(tbrcPath, []byte(`debug: true:`), 0644)
	if err != nil {
		assert.FailNow("Failed to create tbrc file", err)
	}

	err = LoadTBRC()

	assert.Error(err)
}

func TestResolveRegistries(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup()
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	tbrcPath := filepath.Join(dir, tbrcName)
	err = ioutil.WriteFile(tbrcPath, []byte(`registries:
  - name: TouchBistro/tb-registry
  - name: ExampleZone/tb-registry
    localPath: ~/tools/tb-registry
`), 0644)
	if err != nil {
		assert.FailNow("Failed to create tbrc file", err)
	}

	expectedRegistries := []registry.Registry{
		registry.Registry{
			Name:      "TouchBistro/tb-registry",
			LocalPath: "",
			Path:      filepath.Join(RegistriesPath(), "TouchBistro/tb-registry"),
		},
		registry.Registry{
			Name:      "ExampleZone/tb-registry",
			LocalPath: "~/tools/tb-registry",
			Path:      filepath.Join(dir, "tools/tb-registry"),
		},
	}

	err = LoadTBRC()

	assert.NoError(err)
	assert.ElementsMatch(expectedRegistries, Registries())
}

type addRegistryTest struct {
	name         string
	registryName string
	existingTBRC string
	expectedTBRC string
	err          error
}

var addRegistryTests = []addRegistryTest{
	{
		name:         "no existing registries",
		registryName: "TouchBistro/tb-registry",
		existingTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		expectedTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
registries:
  - name: TouchBistro/tb-registry
`,
		err: nil,
	},
	{
		name:         "adding a new registry",
		registryName: "TouchBistro/tb-registry-example",
		existingTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
    localPath: ~/registries/TouchBistro/tb-registry
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		expectedTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
    localPath: ~/registries/TouchBistro/tb-registry
  - name: TouchBistro/tb-registry-example
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		err: nil,
	},
	{
		name:         "registry already exists",
		registryName: "TouchBistro/tb-registry",
		existingTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		expectedTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		err: ErrRegistryExists,
	},
}

func TestAddRegistry(t *testing.T) {
	for _, test := range addRegistryTests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			dir, err := setup()
			if err != nil {
				assert.FailNow("Failed to setup tmp dir", err)
			}
			defer os.RemoveAll(dir)

			tbrcPath := filepath.Join(dir, tbrcName)
			err = ioutil.WriteFile(tbrcPath, []byte(test.existingTBRC), 0644)
			if err != nil {
				assert.FailNow("Failed to create tbrc file", err)
			}

			err = LoadTBRC()
			if err != nil {
				assert.FailNow("Failed to load tbrc", err)
			}

			err = AddRegistry(test.registryName)
			assert.Equal(test.err, err)

			fileData, err := ioutil.ReadFile(tbrcPath)
			if err != nil {
				assert.FailNow("Failed to read tbrc file", err)
			}

			assert.Equal(test.expectedTBRC, string(fileData))
		})
	}
}
