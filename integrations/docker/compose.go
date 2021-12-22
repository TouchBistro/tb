package docker

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
)

const filename = "docker-compose.yml"

type LogsOptions struct {
	Follow bool
	Tail   int
}

type ExecOptions struct {
	// Cmd is the command to execute. It must have at
	// least one element which is the name of the command.
	// Any additional elements are args for the command.
	Cmd    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Compose represents functionality provided by docker-compose.
type Compose interface {
	Build(ctx context.Context, services []string) error
	Stop(ctx context.Context, services []string) error
	Rm(ctx context.Context, services []string) error
	Run(ctx context.Context, service, cmd string) error
	Up(ctx context.Context, services []string) error
	Logs(ctx context.Context, services []string, w io.Writer, opts LogsOptions) error
	Exec(ctx context.Context, service string, opts ExecOptions) (int, error)
}

// NewCompose returns a new Compose instance.
// projectDir is expected to contain a docker-compose.yml file.
// projectName is used for labeling resources created by Compose.
func NewCompose(projectDir, projectName string) Compose {
	// Determine what compose command to use.
	// v2 appears to be barfing for ridiculous reasons.
	// It's complaining about invalid syntax in .env files
	// https://github.com/docker/compose/issues/8763
	// Use v1 until they figure it out.
	cmd := "docker-compose"
	if _, err := exec.LookPath("docker-compose-v1"); err == nil {
		cmd = "docker-compose-v1"
	}
	return &compose{filepath.Join(projectDir, filename), cmd, projectName}
}

// compose is an implementation of Compose that uses the docker-compose command.
type compose struct {
	// composeFile is the path to the docker-compose.yml file.
	composeFile string
	cmd         string // the command to run
	projectName string
}

func (c *compose) Build(ctx context.Context, services []string) error {
	args := append([]string{"build", "--parallel"}, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Build", nil, args...)
}

func (c *compose) Stop(ctx context.Context, services []string) error {
	args := append([]string{"stop", "-t", "2"}, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Stop", nil, args...)
}

func (c *compose) Rm(ctx context.Context, services []string) error {
	args := append([]string{"rm", "-f"}, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Rm", nil, args...)
}

func (c *compose) Run(ctx context.Context, service, cmd string) error {
	args := append([]string{"run", "--rm", NormalizeName(service)}, strings.Fields(cmd)...)
	return c.exec(ctx, "docker.Compose.Run", nil, args...)
}

func (c *compose) Up(ctx context.Context, services []string) error {
	args := append([]string{"up", "-d"}, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Up", nil, args...)
}

func (c *compose) Logs(ctx context.Context, services []string, w io.Writer, opts LogsOptions) error {
	tail := "all"
	if opts.Tail >= 0 {
		tail = strconv.Itoa(opts.Tail)
	}
	args := []string{"logs", "--tail", tail}
	if opts.Follow {
		args = append(args, "--follow")
	}
	args = append(args, normalizeNames(services)...)
	return c.exec(ctx, "docker.Compose.Up", w, args...)
}

func (c *compose) Exec(ctx context.Context, service string, opts ExecOptions) (int, error) {
	const op = errors.Op("docker.Compose.Exec")
	if len(opts.Cmd) == 0 {
		panic("ExecOptions.Cmd must have at least one element")
	}

	// Exec is special so we won't use c.exec but do it manually

	if opts.Stdout == nil || opts.Stderr == nil {
		tracker := progress.TrackerFromContext(ctx)
		w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
		defer w.Close()
		if opts.Stdout == nil {
			opts.Stdout = w
		}
		if opts.Stderr == nil {
			opts.Stderr = w
		}
	}

	args := []string{c.cmd, "--project-name", c.projectName, "--file", c.composeFile, "exec", NormalizeName(service)}
	args = append(args, opts.Cmd...)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	err := cmd.Run()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// This isn't actually an error with tb, it means the underlying command that was executed
		// was unsuccessful so don't treat it as an error but signal the status code.
		return exitErr.ExitCode(), nil
	}
	if err != nil {
		return -1, errors.Wrap(err, errors.Meta{
			Kind:   errkind.DockerCompose,
			Reason: fmt.Sprintf("failed to run %q", strings.Join(args, " ")),
			Op:     op,
		})
	}
	return 0, nil
}

func (c *compose) exec(ctx context.Context, op errors.Op, stdout io.Writer, args ...string) error {
	tracker := progress.TrackerFromContext(ctx)
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
	defer w.Close()
	if stdout == nil {
		stdout = w
	}

	finalArgs := append([]string{c.cmd, "--project-name", c.projectName, "--file", c.composeFile}, args...)
	cmd := exec.CommandContext(ctx, finalArgs[0], finalArgs[1:]...)
	cmd.Stdout = stdout
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

// normalizeNames calls normalizeName on each name.
func normalizeNames(names []string) []string {
	nn := make([]string, len(names))
	for i, n := range names {
		nn[i] = NormalizeName(n)
	}
	return nn
}

// ComposeConfig represents the configuration for docker compose
// that is specified in a docker-compose.yml file.
type ComposeConfig struct {
	Version  string                          `yaml:"version"`
	Services map[string]ComposeServiceConfig `yaml:"services"`

	// Top level named volumes are an empty field, i.e. `postgres:`
	// There's no way to create an empty field with go-yaml
	// so we use interface{} and set it to nil which produces `postgres: null`
	// docker-compose seems cool with this

	Volumes map[string]interface{} `yaml:"volumes,omitempty"`
}

type ComposeServiceConfig struct {
	Build         ComposeBuildConfig `yaml:"build,omitempty"` // non-remote
	Command       string             `yaml:"command,omitempty"`
	ContainerName string             `yaml:"container_name"`
	DependsOn     []string           `yaml:"depends_on,omitempty"`
	Entrypoint    []string           `yaml:"entrypoint,omitempty"`
	EnvFile       []string           `yaml:"env_file,omitempty"`
	Environment   map[string]string  `yaml:"environment,omitempty"`
	Image         string             `yaml:"image,omitempty"` // remote
	Ports         []string           `yaml:"ports,omitempty"`
	Volumes       []string           `yaml:"volumes,omitempty"`
}

type ComposeBuildConfig struct {
	Args    map[string]string `yaml:"args,omitempty"`
	Context string            `yaml:"context,omitempty"`
	Target  string            `yaml:"target,omitempty"`
}
