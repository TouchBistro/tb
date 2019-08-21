package docker

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

const (
	stopTimeoutSecs = 2
)

func ComposeFile() string {
	return fmt.Sprintf("-f %s/docker-compose.yml", config.TBRootPath())
}

func ComposeStop() error {
	stopArgs := fmt.Sprintf("%s stop -t %d", ComposeFile(), stopTimeoutSecs)
	err := util.Exec("docker-compose-stop", "docker-compose", strings.Fields(stopArgs)...)

	return errors.Wrap(err, "could not exec docker-compose stop")
}
