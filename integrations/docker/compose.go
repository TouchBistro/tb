package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/logutil"
	"github.com/TouchBistro/goutils/progress"
)

// This file should be kept as general purpose as possible.
// Hopefully in the future if a docker compose library is released
// we can switch to that instead and remove this.

const ComposeFilename = "docker-compose.yml"

// ComposeAPIClient represents the functionality provided by docker-compose.
type ComposeAPIClient interface {
	ComposeBuild(ctx context.Context, project ComposeProject, services []string) error
	ComposeUp(ctx context.Context, project ComposeProject, services []string) error
	ComposeRun(ctx context.Context, project ComposeProject, opts ComposeRunOptions) error
	ComposeExec(ctx context.Context, project ComposeProject, opts ComposeRunOptions) (int, error)
	ComposeLogs(ctx context.Context, project ComposeProject, opts ComposeLogsOptions) error
}

// ComposeProject encapsulates information on a docker-compose project.
type ComposeProject struct {
	// Name is the name of the project. It is used for labels on docker resources.
	Name string
	// Workdir is the directory where the project is located.
	// This directory is expected to contain a docker-compose.yml file.
	Workdir string
}

type ComposeRunOptions struct {
	// Service is the service to run the command on. It must not be empty.
	Service string
	// Cmd is the command to execute. It must have at
	// least one element which is the name of the command.
	// Any additional elements are args for the command.
	Cmd    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type ComposeLogsOptions struct {
	Services []string
	Out      io.Writer
	Follow   bool
	Tail     string
}

func (c *apiClient) ComposeBuild(ctx context.Context, project ComposeProject, services []string) error {
	return c.execCompose(ctx, execComposeOptions{
		project:        project,
		useComposeFile: true,
		args:           append([]string{"build", "--parallel"}, services...),
	})
}

func (c *apiClient) ComposeUp(ctx context.Context, project ComposeProject, services []string) error {
	return c.execCompose(ctx, execComposeOptions{
		project:        project,
		useComposeFile: true,
		args:           append([]string{"up", "-d"}, services...),
	})
}

func (c *apiClient) ComposeRun(ctx context.Context, project ComposeProject, opts ComposeRunOptions) error {
	return c.execCompose(ctx, execComposeOptions{
		project:        project,
		useComposeFile: true,
		args:           append([]string{"run", "--rm", opts.Service}, opts.Cmd...),
		stdin:          opts.Stdin,
		stdout:         opts.Stdout,
		stderr:         opts.Stderr,
	})
}

func (c *apiClient) ComposeExec(ctx context.Context, project ComposeProject, opts ComposeRunOptions) (int, error) {
	err := c.execCompose(ctx, execComposeOptions{
		project: project,
		args:    append([]string{"exec", opts.Service}, opts.Cmd...),
		stdin:   opts.Stdin,
		stdout:  opts.Stdout,
		stderr:  opts.Stderr,
	})

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// This isn't actually an error with tb, it means the underlying command that was executed
		// was unsuccessful so don't treat it as an error but signal the status code.
		return exitErr.ExitCode(), nil
	}
	if err != nil {
		return -1, err
	}
	return 0, nil
}

func (c *apiClient) ComposeLogs(ctx context.Context, project ComposeProject, opts ComposeLogsOptions) error {
	if opts.Tail != "" {
		opts.Tail = "all"
	}
	args := []string{"logs", "--tail", opts.Tail}
	if opts.Follow {
		args = append(args, "--follow")
	}
	return c.execCompose(ctx, execComposeOptions{
		project: project,
		args:    append(args, opts.Services...),
		stdout:  opts.Out,
	})
}

type execComposeOptions struct {
	project        ComposeProject
	useComposeFile bool
	args           []string
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
}

func (c *apiClient) execCompose(ctx context.Context, opts execComposeOptions) error {
	var w io.Writer
	if opts.stdout == nil || opts.stderr == nil {
		tracker := progress.TrackerFromContext(ctx)
		op := fmt.Sprintf("docker-compose-%s", opts.args[0])
		wc := logutil.LogWriter(tracker.WithAttrs("op", op), slog.LevelDebug)
		defer wc.Close()
		w = wc
	}
	if opts.stdout == nil {
		opts.stdout = w
	}
	if opts.stderr == nil {
		opts.stderr = w
	}

	// Use compose v2 which is part of the docker CLI.
	args := []string{"docker", "compose", "--project-name", opts.project.Name}
	// In compose v2 not all commands require the compose file, many can work off docker labels.
	// Only provide the compose file if it is explicitly marked as required.
	if opts.useComposeFile {
		fp := filepath.Join(opts.project.Workdir, ComposeFilename)
		args = append(args, "--file", fp)
	}
	args = append(args, opts.args...)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = opts.stdin
	cmd.Stdout = opts.stdout
	cmd.Stderr = opts.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", strings.Join(args, " "), err)
	}
	return nil
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
