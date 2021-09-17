package docker

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/tb/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	stopTimeoutSecs = 2
)

var composeCmd string

func composeCommand() string {
	if composeCmd != "" {
		return composeCmd
	}

	_, err := exec.LookPath("docker-compose-v1")

	if err == nil {
		log.Debugf("docker-compose-v1 exists, using it to avoid race condition issue with Docker Compose version v2.0.0-rc.3...")
		composeCmd = "docker-compose-v1"
	} else {
		log.Debugf("docker-compose-v1 does not exist, falling back to docker-compose (this might break if using Docker Compose version v2.0.0-rc.3)...")
		composeCmd = "docker-compose"
	}

	return composeCmd
}

func composeFile() string {
	return fmt.Sprintf("%s/docker-compose.yml", config.TBRootPath())
}

func ComposeExec(serviceName string, execArgs []string, cmd *command.Command) error {
	args := []string{"-f", composeFile(), "exec", serviceName}
	args = append(args, execArgs...)
	err := cmd.Exec(composeCommand(), args...)
	if err != nil {
		return errors.Wrap(err, "failed to run docker-compose exec")
	}
	return nil
}

func ComposeLogs(services []string, cmd *command.Command) error {
	args := []string{"-f", composeFile(), "logs", "-f"}
	args = append(args, services...)
	err := cmd.Exec(composeCommand(), args...)
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
	args := append([]string{"--rm", serviceName}, strings.Fields(cmd)...)
	return execDockerCompose("run", args...)
}

func execDockerCompose(subcmd string, args ...string) error {
	w := log.WithField("id", "docker-compose-"+subcmd).WriterLevel(log.DebugLevel)
	defer w.Close()
	cmd := command.New(command.WithStdout(w), command.WithStderr(w))
	args = append([]string{"-f", composeFile(), subcmd}, args...)
	err := cmd.Exec(composeCommand(), args...)
	if err != nil {
		return errors.Wrapf(err, "failed to run docker-compose %s", subcmd)
	}
	return nil
}
