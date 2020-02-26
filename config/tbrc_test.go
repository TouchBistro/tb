package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setup() (string, error) {
	tbrc = userConfig{}
	dir, err := ioutil.TempDir("", "tbrc_test")
	os.Setenv("HOME", dir)
	return dir, err
}

func TestInitTBRC(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup()
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	tbrcPath := filepath.Join(dir, tbrcName)
	err = ioutil.WriteFile(tbrcPath, []byte(`debug: true
experimental: true
playlists:
  my-core:
    extends: TouchBistro/tb-recipe/core
    services:
      - partners-config-service
overrides:
  TouchBistro/tb-recipe/postgres:
    remote:
      enabled: true
      tag: 12-alpine
  venue-core-service:
    envVars:
      AUTH_URL: http://localhost:8002/auth
    preRun: yarn db:prepare:dev
    build:
      command: 'tail -f /dev/null'
      target: release
    remote:
      command: 'tail -f /dev/null'
      enabled: true
      tag: feat/delete-menu
`), 0644)
	if err != nil {
		assert.FailNow("Failed to create tbrc file", err)
	}

	expectedTBRC := userConfig{
		DebugEnabled:        true,
		ExperimentalEnabled: true,
		Playlists: map[string]Playlist{
			"my-core": Playlist{
				Extends: "TouchBistro/tb-recipe/core",
				Services: []string{
					"partners-config-service",
				},
			},
		},
		Overrides: map[string]ServiceOverride{
			"TouchBistro/tb-recipe/postgres": {
				Remote: RemoteOverride{
					Enabled: true,
					Tag:     "12-alpine",
				},
			},
			"venue-core-service": {
				EnvVars: map[string]string{
					"AUTH_URL": "http://localhost:8002/auth",
				},
				PreRun: "yarn db:prepare:dev",
				Build: BuildOverride{
					Command: "tail -f /dev/null",
					Target:  "release",
				},
				Remote: RemoteOverride{
					Command: "tail -f /dev/null",
					Enabled: true,
					Tag:     "feat/delete-menu",
				},
			},
		},
	}

	err = InitTBRC()

	assert.NoError(err)
	assert.Equal(expectedTBRC, tbrc)
}

func TestInitTBRCDefault(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup()
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	tbrcPath := filepath.Join(dir, tbrcName)
	expectedTBRC := userConfig{}

	err = InitTBRC()

	assert.NoError(err)
	assert.Equal(expectedTBRC, tbrc)
	assert.FileExists(tbrcPath)
}

func TestInitTBRCInvalidYaml(t *testing.T) {
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

	err = InitTBRC()

	assert.Error(err)
}
