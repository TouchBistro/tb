package util

import (
	"path/filepath"
	"strings"
)

func ComposeFiles() (string, error) {
	matches, err := filepath.Glob("./docker-compose.*.yml")

	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", nil
	}

	str := "-f " + strings.Join(matches, " -f ")

	return str, nil
}
