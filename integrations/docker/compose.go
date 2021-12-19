package docker

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
)

const filename = "docker-compose.yml"

// Compose represents functionality provided by docker-compose.
type Compose interface {
	Build(ctx context.Context, services []string) error
	Stop(ctx context.Context, services []string) error
	Rm(ctx context.Context, services []string) error
	Run(ctx context.Context, service, cmd string) error
	Up(ctx context.Context, services []string) error
}

// NewCompose returns a new Compose instance.
// projectDir is expected to contain a docker-compose.yml file.
func NewCompose(projectDir string) Compose {
	// Determine what compose command to use.
	// v2 appears to be barfing for ridiculous reasons.
	// It's complaining about invalid syntax in .env files
	// https://github.com/docker/compose/issues/8763
	// Use v1 until they figure it out.
	cmd := "docker-compose"
	if _, err := exec.LookPath("docker-compose-v1"); err == nil {
		cmd = "docker-compose-v1"
	}
	return compose{composeFile: filepath.Join(projectDir, filename), cmd: cmd}
}

// compose is an implementation of Compose that uses the docker-compose command.
type compose struct {
	// composeFile is the path to the docker-compose.yml file.
	composeFile string
	cmd         string // the command to run
}

func (c compose) Build(ctx context.Context, services []string) error {
	args := append([]string{"--parallel"}, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Build", "build", args...)
}

func (c compose) Stop(ctx context.Context, services []string) error {
	args := append([]string{"-t", "2"}, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Stop", "stop", args...)
}

func (c compose) Rm(ctx context.Context, services []string) error {
	args := append([]string{"-f"}, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Rm", "rm", args...)
}

func (c compose) Run(ctx context.Context, service, cmd string) error {
	args := append([]string{"--rm", normalizeName(service)}, strings.Fields(cmd)...)
	return c.exec(ctx, "docker.Compose.Run", "run", args...)
}

func (c compose) Up(ctx context.Context, services []string) error {
	args := append([]string{"-d"}, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Up", "up", args...)
}

func (c compose) exec(ctx context.Context, op errors.Op, subcmd string, args ...string) error {
	tracker := progress.TrackerFromContext(ctx)
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
	defer w.Close()

	finalArgs := append([]string{c.cmd, "-f", c.composeFile, subcmd}, args...)
	cmd := exec.CommandContext(ctx, finalArgs[0], finalArgs[1:]...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.DockerCompose,
			Reason: fmt.Sprintf("failed to run %q", strings.Join(finalArgs, " ")),
			Op:     op,
		})
	}
	return nil
}

// normalizeName ensures that name is allowed by docker-compose.
// docker-compose does not allow slashes or upper case letters in service names,
// they are replaced with dashes and lower case letters respectively.
func normalizeName(name string) string {
	sanitized := strings.ReplaceAll(name, "/", "-")
	return strings.ToLower(sanitized)
}

// normalizeNames calls normalizeName on each name.
func normalizeNames(names []string) []string {
	nn := make([]string, len(names))
	for i, n := range names {
		nn[i] = normalizeName(n)
	}
	return nn
}
