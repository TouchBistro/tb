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
