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
