package docker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/tb/util"
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

func ComposeStop() error {
	composeFiles, err := ComposeFiles()

	if err != nil {
		return err
	}

	stopArgs := fmt.Sprintf("%s stop", composeFiles)
	_, err = util.Exec("docker-compose", strings.Fields(stopArgs)...)

	return err
}
