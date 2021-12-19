// Package service contains the types for defining services.
package service

import (
	"fmt"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
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

// Collection stores a collection of services.
// Collection allows for efficiently looking up a service by its
// short name (i.e. the name of the service without the registry).
//
// A zero value Collection is a valid collection ready for use.
type Collection struct {
	collection resource.Collection
}

// Len returns the number of services stored in the Collection.
func (c *Collection) Len() int {
	return c.collection.Len()
}

// Get retrieves the service with the given name from the Collection.
// name can either be the full name or the short name of the service.
//
// If no service is found, resource.ErrNotFound is returned. If name is a short name
// and multiple services are found, resource.ErrMultipleResources is returned.
func (c *Collection) Get(name string) (Service, error) {
	r, err := c.collection.Get(name)
	if err != nil {
		return Service{}, errors.Wrap(err, errors.Meta{Op: errors.Op("service.Collection.Get")})
	}
	return r.(Service), nil
}

// Set adds or replaces the service in the Collection.
// s.FullName() must return a valid full name or an error will be returned.
func (c *Collection) Set(s Service) error {
	if err := c.collection.Set(s); err != nil {
		return errors.Wrap(err, errors.Meta{Op: errors.Op("service.Collection.Set")})
	}
	return nil
}

// Iterator allows for iteration over the services in a Collection.
// An iterator provides two methods that can be used for iteration, Next and Value.
// Next advances the iterator to the next element and returns a bool indicating if
// it was successful. Value returns the value at the current index.
//
// The iteration order over a Collection is not specified and is not guaranteed to be the same
// from one iteration to the next.
type Iterator struct {
	*resource.Iterator
}

// Iter creates a new Iterator that can be used to iterate over the services in a Collection.
func (c *Collection) Iter() *Iterator {
	return &Iterator{c.collection.Iter()}
}

// Value returns the current element in the iterator.
// Value will panic if iteration has finished.
func (it *Iterator) Value() Service {
	return it.Iterator.Value().(Service)
}