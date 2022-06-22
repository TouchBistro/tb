// Package service contains functionality for working with Service resources.
// A service is an application that can be run in a docker container.
package service

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/resource"
)

const (
	ModeRemote = "remote"
	ModeBuild  = "build"
)

// Service specifies the configuration for a service that can be run by tb.
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

type Volume struct {
	Value   string `yaml:"value"`
	IsNamed bool   `yaml:"named"`
}

func (s Service) HasGitRepo() bool {
	return s.GitRepo.Name != ""
}

func (s Service) CanBuild() bool {
	return s.Build.DockerfilePath != ""
}

// ImageURI returns the image optionally with the tag if one exists.
func (s Service) ImageURI() string {
	if s.Remote.Tag == "" {
		return s.Remote.Image
	}
	return s.Remote.Image + ":" + s.Remote.Tag
}

func (Service) Type() resource.Type {
	return resource.TypeService
}

// FullName returns the service name prefixed with the registry name,
// i.e. '<registry>/<service>'.
func (s Service) FullName() string {
	return resource.FullName(s.RegistryName, s.Name)
}

// Validate validates s. If s is invalid a resource.ValidationError will be returned.
func Validate(s Service) error {
	var msgs []string
	if s.Mode != ModeRemote && s.Mode != ModeBuild {
		msg := fmt.Sprintf("invalid 'mode' value %q, must be 'remote' or 'build'", s.Mode)
		msgs = append(msgs, msg)
	}
	if s.Mode == ModeRemote && s.Remote.Image == "" {
		msgs = append(msgs, "'mode' is set to 'remote' but 'remote.image' was not provided")
	}
	if s.Mode == ModeBuild && s.Build.DockerfilePath == "" {
		msgs = append(msgs, "'mode' is set to 'build' but 'build.dockerfilePath' was not provided")
	}
	if msgs == nil {
		return nil
	}
	return &resource.ValidationError{Resource: s, Messages: msgs}
}

// ServiceOverride defines the overrides that should be applied to a Service.
// It is a subset of the fields of Service, since not all fields are allowed to
// be overridden.
type ServiceOverride struct {
	Build   BuildOverride     `yaml:"build"`
	EnvVars map[string]string `yaml:"envVars"`
	GitRepo GitRepoOverride   `yaml:"repo"`
	Mode    string            `yaml:"mode"`
	PreRun  string            `yaml:"preRun"`
	Remote  RemoteOverride    `yaml:"remote"`
}

type BuildOverride struct {
	Command string `yaml:"command"`
	Target  string `yaml:"target"`
}

type GitRepoOverride struct {
	Path string `yaml:"path"`
}

type RemoteOverride struct {
	Command string `yaml:"command"`
	Tag     string `yaml:"tag"`
}

// Override applies the overrides from o to s. If applying the override
// results in an invalid configuration, Override will return an error.
func Override(s Service, o ServiceOverride) (Service, error) {
	const op = errors.Op("service.Override")
	// Validate overrides
	if o.Mode != "" {
		if o.Mode != ModeRemote && o.Mode != ModeBuild {
			msg := fmt.Sprintf("invalid override value for '%s.mode', must be 'remote' or 'build'", s.FullName())
			return s, errors.New(errkind.Invalid, msg, op)
		}
		if o.Mode == ModeRemote && s.Remote.Image == "" {
			msg := fmt.Sprintf("'%s.mode' is overridden to 'remote' but it is not available from a remote source", s.FullName())
			return s, errors.New(errkind.Invalid, msg, op)
		} else if o.Mode == ModeBuild && !s.CanBuild() {
			msg := fmt.Sprintf("'%s.mode' is overridden to 'build' but it cannot be built locally", s.FullName())
			return s, errors.New(errkind.Invalid, msg, op)
		}
	}

	// Apply overrides
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

// DISCUSS(@cszatmary): Does this make sense here? I honestly struggled with where to put this the most.
// I considerered the following:
// config: Does not seem like config's business though as config deals with the higher level glue code.
// docker: It would make the most sense in docker, but integrations shouldn't import from resources.
// services: This logic has a lot to do with services so we can justify that services supports mapping to
// other formats.

// ComposeConfig maps the Collection to a docker compose config.
func ComposeConfig(c *resource.Collection[Service]) docker.ComposeConfig {
	composeConfig := docker.ComposeConfig{
		Version:  "3.7",
		Services: make(map[string]docker.ComposeServiceConfig),
		Volumes:  make(map[string]interface{}),
	}
	for it := c.Iter(); it.Next(); {
		s := it.Value()
		dockerName := docker.NormalizeName(s.FullName())
		cs := docker.ComposeServiceConfig{
			ContainerName: dockerName,
			DependsOn:     s.Dependencies,
			Entrypoint:    s.Entrypoint,
			Environment:   s.EnvVars,
			Ports:         s.Ports,
		}
		if s.EnvFile != "" {
			cs.EnvFile = append(cs.EnvFile, s.EnvFile)
		}

		var volumes []Volume
		if s.Mode == ModeRemote {
			cs.Command = s.Remote.Command
			cs.Image = s.ImageURI()
			volumes = s.Remote.Volumes
		} else {
			cs.Build = docker.ComposeBuildConfig{
				Args:    s.Build.Args,
				Context: s.Build.DockerfilePath,
				Target:  s.Build.Target,
			}
			cs.Command = s.Build.Command
			volumes = s.Build.Volumes
		}

		for _, v := range volumes {
			cs.Volumes = append(cs.Volumes, v.Value)
			if v.IsNamed {
				namedVolume := strings.Split(v.Value, ":")[0]
				composeConfig.Volumes[namedVolume] = nil
			}
		}
		composeConfig.Services[dockerName] = cs
	}
	return composeConfig
}
