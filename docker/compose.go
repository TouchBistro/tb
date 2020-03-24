package docker

import (
	"fmt"
	"os/exec"
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

func ComposeExec(serviceName string, execArgs []string, opts ...func(*exec.Cmd)) error {
	composeCmd := fmt.Sprintf("%s exec %s", ComposeFile(), serviceName)
	composeArgs := strings.Split(composeCmd, " ")
	composeArgs = append(composeArgs, execArgs...)
	err := command.Exec("docker-compose", composeArgs, "docker-compose-exec", opts...)

	return errors.Wrap(err, "could not exec docker-compose exec")
}

func ComposeBuild(services []string) error {
	namesArg := strings.Join(services, " ")
	buildArgs := fmt.Sprintf("%s build --parallel %s", ComposeFile(), namesArg)
	err := command.Exec("docker-compose", strings.Fields(buildArgs), "docker-compose-build")

	return errors.Wrap(err, "could not exec docker-compose build")
}

func ComposeUp(services []string) error {
	namesArg := strings.Join(services, " ")
	upArgs := fmt.Sprintf("%s up -d %s", ComposeFile(), namesArg)
	err := command.Exec("docker-compose", strings.Fields(upArgs), "docker-compose-up")

	return errors.Wrap(err, "could not exec docker-compose up")
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

func ComposeRun(serviceName, cmd string) error {
	runArgs := fmt.Sprintf("%s run --rm %s %s", ComposeFile(), serviceName, cmd)
	err := command.Exec("docker-compose", strings.Fields(runArgs), "docker-compose-run")

	return errors.Wrap(err, "could not execute docker-compose run")
}
