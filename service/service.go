package service

import (
	"fmt"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

const (
	ModeRemote = "remote"
	ModeBuild  = "build"
)

type Volume struct {
	Value   string `yaml:"value"`
	IsNamed bool   `yaml:"named"`
}

type Build struct {
	Args           map[string]string `yaml:"args"`
	Command        string            `yaml:"command"`
	DockerfilePath string            `yaml:"dockerfilePath"`
	Target         string            `yaml:"target"`
	Volumes        []Volume          `yaml:"volumes"`
}

type GitRepo struct {
	Name string `yaml:"name"`
}

type Remote struct {
	Command string   `yaml:"command"`
	Image   string   `yaml:"image"`
	Tag     string   `yaml:"tag"`
	Volumes []Volume `yaml:"volumes"`
}

type Service struct {
	Build        Build             `yaml:"build"`
	Dependencies []string          `yaml:"dependencies"`
	Entrypoint   []string          `yaml:"entrypoint"`
	EnvFile      string            `yaml:"envFile"`
	EnvVars      map[string]string `yaml:"envVars"`
	GitRepo      GitRepo           `yaml:"repo"`
	Mode         string            `yaml:"mode"`
	Ports        []string          `yaml:"ports"`
	PreRun       string            `yaml:"preRun"`
	Remote       Remote            `yaml:"remote"`
	// Not part of yaml, set at runtime
	Name         string `yaml:"-"`
	RegistryName string `yaml:"-"`
}

func (s Service) HasGitRepo() bool {
	return s.GitRepo.Name != ""
}

func (s Service) UseRemote() bool {
	return s.Mode == ModeRemote
}

func (s Service) CanBuild() bool {
	return s.Build.DockerfilePath != ""
}

func (s Service) ImageURI() string {
	if s.Remote.Tag == "" {
		return s.Remote.Image
	}

	return fmt.Sprintf("%s:%s", s.Remote.Image, s.Remote.Tag)
}

// FullName returns the service name prefixed with the registry name.
// i.e. <registry>/<service>
func (s Service) FullName() string {
	return fmt.Sprintf("%s/%s", s.RegistryName, s.Name)
}

// DockerName returns a variation of FullName that
// has been modified to meet docker naming requirements.
func (s Service) DockerName() string {
	return util.DockerName(s.FullName())
}

type BuildOverride struct {
	Command string `yaml:"command"`
	Target  string `yaml:"target"`
}

type RemoteOverride struct {
	Command string `yaml:"command"`
	Tag     string `yaml:"tag"`
}

type ServiceOverride struct {
	Build   BuildOverride     `yaml:"build"`
	EnvVars map[string]string `yaml:"envVars"`
	Mode    string            `yaml:"mode"`
	PreRun  string            `yaml:"preRun"`
	Remote  RemoteOverride    `yaml:"remote"`
}

func (s Service) applyOverride(o ServiceOverride) (Service, error) {
	// Validate overrides
	if o.Mode != "" {
		// Make sure mode is a valid value
		if o.Mode != ModeRemote && o.Mode != ModeBuild {
			return s, errors.Errorf("'%s.mode' value is invalid must be 'remote' or 'build'", s.FullName())
		}

		if o.Mode == ModeRemote && s.Remote.Image == "" {
			msg := fmt.Sprintf("%s.mode is overridden to 'remote' but it is not available from a remote source", s.FullName())
			return s, errors.New(msg)
		} else if o.Mode == ModeBuild && !s.CanBuild() {
			msg := fmt.Sprintf("%s.mode is overridden to 'build' but it cannot be built locally", s.FullName())
			return s, errors.New(msg)
		}
	}

	// Apply overrides to service
	if o.Build.Command != "" {
		s.Build.Command = o.Build.Command
	}

	if o.Build.Target != "" {
		s.Build.Target = o.Build.Target
	}

	if o.EnvVars != nil {
		for v, val := range o.EnvVars {
			s.EnvVars[v] = val
		}
	}

	if o.PreRun != "" {
		s.PreRun = o.PreRun
	}

	if o.Remote.Command != "" {
		s.Remote.Command = o.Remote.Command
	}

	if o.Mode != "" {
		s.Mode = o.Mode
	}

	if o.Remote.Tag != "" {
		s.Remote.Tag = o.Remote.Tag
	}

	return s, nil
}
