package service

import (
	"errors"
	"fmt"

	"github.com/TouchBistro/tb/util"
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

type Remote struct {
	Command string   `yaml:"command"`
	Enabled bool     `yaml:"enabled"`
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
	GitRepo      string            `yaml:"repo"`
	Ports        []string          `yaml:"ports"`
	PreRun       string            `yaml:"preRun"`
	Remote       Remote            `yaml:"remote"`
	// Not part of yaml, set at runtime
	Name         string `yaml:"-"`
	RegistryName string `yaml:"-"`
}

func (s Service) HasGitRepo() bool {
	return s.GitRepo != ""
}

func (s Service) UseRemote() bool {
	return s.Remote.Enabled
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
	Enabled bool   `yaml:"enabled"`
	Tag     string `yaml:"tag"`
}

type ServiceOverride struct {
	Build   BuildOverride     `yaml:"build"`
	EnvVars map[string]string `yaml:"envVars"`
	PreRun  string            `yaml:"preRun"`
	Remote  RemoteOverride    `yaml:"remote"`
}

func (s Service) applyOverride(o ServiceOverride) (Service, error) {
	// Validate overrides
	if o.Remote.Enabled && s.Remote.Image == "" {
		msg := fmt.Sprintf("remote.enabled is overridden to true for %s but it is not available from a remote source", s.FullName())
		return s, errors.New(msg)
	} else if !o.Remote.Enabled && !s.HasGitRepo() {
		msg := fmt.Sprintf("remote.enabled is overridden to false but %s cannot be built locally", s.FullName())
		return s, errors.New(msg)
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

	s.Remote.Enabled = o.Remote.Enabled
	if o.Remote.Tag != "" {
		s.Remote.Tag = o.Remote.Tag
	}

	return s, nil
}
