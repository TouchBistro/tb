package docker

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/tb/config"
	"github.com/pkg/errors"
)

const (
	stopTimeoutSecs = 2
)

func ComposeFile() string {
	return fmt.Sprintf("-f %s/docker-compose.yml", config.TBRootPath())
}

func ComposeStop(services []string) error {
	namesArg := strings.Join(services, " ")
	stopArgs := fmt.Sprintf("%s stop -t %d %s", ComposeFile(), stopTimeoutSecs, namesArg)
	err := command.Exec("docker-compose", strings.Fields(stopArgs), "docker-compose-stop")

	return errors.Wrap(err, "could not exec docker-compose stop")
}

func ComposeRm(services []string) error {
	namesArg := strings.Join(services, " ")
	rmArgs := fmt.Sprintf("%s rm -f %s", ComposeFile(), namesArg)
	err := command.Exec("docker-compose", strings.Fields(rmArgs), "docker-compose-rm")

	return errors.Wrap(err, "could not exec docker-compose rm")
}
