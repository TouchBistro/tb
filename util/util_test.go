package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandVars(t *testing.T) {
	assert := assert.New(t)

	str := `envPath: ${@env:HOME}/${@REPOPATH}/${name}`
	vars := map[string]string{
		"@REPOPATH": ".tb/repos",
		"name":      "node-boilerplate",
	}

	expanded := ExpandVars(str, vars)

	expected := `envPath: ${HOME}/.tb/repos/node-boilerplate`
	assert.Equal(expected, expanded)
}
