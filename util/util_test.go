package util_test

import (
	"testing"

	"github.com/TouchBistro/tb/util"
	"github.com/stretchr/testify/assert"
)

func TestExpandVars(t *testing.T) {
	assert := assert.New(t)

	str := `envPath: ${@env:HOME}/${@REPOPATH}/${name}`
	vars := map[string]string{
		"@REPOPATH": ".tb/repos",
		"name":      "node-boilerplate",
	}

	expanded, err := util.ExpandVars(str, vars)

	expected := `envPath: ${HOME}/.tb/repos/node-boilerplate`
	assert.Equal(expected, expanded)
	assert.NoError(err)
}

func TestExpandVarsMissingVar(t *testing.T) {
	assert := assert.New(t)

	str := `DB_HOST: ${@postgres}`
	vars := map[string]string{}

	expanded, err := util.ExpandVars(str, vars)

	assert.Empty(expanded)
	assert.Error(err)
}

func TestUnqiueStrings(t *testing.T) {
	assert := assert.New(t)

	s := []string{"npm", "ecr", "ecr", "gcp", "npm", "ecr"}
	expected := []string{"npm", "ecr", "gcp"}
	result := util.UniqueStrings(s)

	assert.Equal(expected, result)
}

func TestSplitNameParts(t *testing.T) {
	assert := assert.New(t)

	name := "TouchBistro/tb-registry/touchbistro-node-boilerplate"
	registryName, serviceName, err := util.SplitNameParts(name)

	assert.Equal("TouchBistro/tb-registry", registryName)
	assert.Equal("touchbistro-node-boilerplate", serviceName)
	assert.NoError(err)
}

func TestSplitNamePartsShortName(t *testing.T) {
	assert := assert.New(t)

	name := "touchbistro-node-boilerplate"
	registryName, serviceName, err := util.SplitNameParts(name)

	assert.Empty(registryName)
	assert.Equal("touchbistro-node-boilerplate", serviceName)
	assert.NoError(err)
}

func TestSplitNamePartsInvalid(t *testing.T) {
	assert := assert.New(t)

	name := "TouchBistro/touchbistro-node-boilerplate"
	registryName, serviceName, err := util.SplitNameParts(name)

	assert.Empty(registryName)
	assert.Empty(serviceName)
	assert.Error(err)
}

func TestDockerName(t *testing.T) {
	assert := assert.New(t)

	name := "TouchBistro/tb-registry/touchbistro-node-boilerplate"
	dockerName := util.DockerName(name)

	expected := "touchbistro-tb-registry-touchbistro-node-boilerplate"
	assert.Equal(expected, dockerName)
}
