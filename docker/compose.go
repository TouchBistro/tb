package docker

import (
	"fmt"
	"strconv"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/tb/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	stopTimeoutSecs = 2
)

func composeFile() string {
	return fmt.Sprintf("%s/docker-compose.yml", config.TBRootPath())
}

func ComposeExec(serviceName string, execArgs []string, cmd *command.Command) error {
	args := []string{"-f", composeFile(), "exec", serviceName}
	args = append(args, execArgs...)
	err := cmd.Exec("docker-compose", args...)
	if err != nil {
		return errors.Wrap(err, "failed to run docker-compose exec")
	}
	return nil
}

func ComposeLogs(services []string, cmd *command.Command) error {
	args := []string{"-f", composeFile(), "logs", "-f"}
	args = append(args, services...)
	err := cmd.Exec("docker-compose", args...)
	if err != nil {
		return errors.Wrap(err, "failed to run docker-compose logs")
	}
	return nil
}

func ComposeBuild(services []string) error {
	args := append([]string{"--parallel"}, services...)
	return execDockerCompose("build", args...)
}

func ComposeUp(services []string) error {
	args := append([]string{"-d"}, services...)
	return execDockerCompose("up", args...)
}

func ComposeStop(services []string) error {
	args := append([]string{"-t", strconv.Itoa(stopTimeoutSecs)}, services...)
	return execDockerCompose("stop", args...)
}

func ComposeRm(services []string) error {
	args := append([]string{"-f"}, services...)
	return execDockerCompose("rm", args...)
}

func ComposeRun(serviceName, cmd string) error {
	return execDockerCompose("run", "--rm", serviceName, cmd)
}

func execDockerCompose(subcmd string, args ...string) error {
	w := log.WithField("id", "docker-compose-"+subcmd).WriterLevel(log.DebugLevel)
	defer w.Close()
	cmd := command.New(command.WithStdout(w), command.WithStderr(w))
	args = append([]string{"-f", composeFile(), subcmd}, args...)
	err := cmd.Exec("docker-compose", args...)
	if err != nil {
		return errors.Wrapf(err, "failed to run docker-compose %s", subcmd)
	}
	return nil
}
