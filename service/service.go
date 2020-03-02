package service

import (
	"fmt"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
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
	Name       string `yaml:"-"`
	RecipeName string `yaml:"-"`
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

func (s Service) FullName() string {
	return util.JoinNameParts(s.RecipeName, s.Name)
}

type ServiceCollection struct {
	sm  map[string][]Service
	len int
}

func NewServiceCollection() *ServiceCollection {
	return &ServiceCollection{sm: make(map[string][]Service)}
}

func (sc *ServiceCollection) Get(name string) (Service, error) {
	recipeName, serviceName, err := util.SplitNameParts(name)
	if err != nil {
		return Service{}, errors.Wrapf(err, "invalid service name %s", name)
	}

	bucket, ok := sc.sm[serviceName]
	if !ok {
		return Service{}, errors.Errorf("No such service %s", serviceName)
	}

	// Handle shorthand syntax
	if recipeName == "" {
		if len(bucket) > 1 {
			return Service{}, errors.Errorf("Muliple services named %s found", serviceName)
		}

		return bucket[0], nil
	}

	// Handle long syntax
	for _, s := range bucket {
		if s.RecipeName == recipeName {
			return s, nil
		}
	}

	return Service{}, errors.Errorf("No such service %s", name)
}

func (sc *ServiceCollection) Set(value Service) error {
	if value.Name == "" || value.RecipeName == "" {
		return errors.Errorf("Name and RecipeName fields must not be empty to set Service")
	}

	fullName := value.FullName()
	recipeName, serviceName, err := util.SplitNameParts(fullName)
	if err != nil {
		return errors.Wrapf(err, "invalid service name %s", fullName)
	}

	bucket, ok := sc.sm[serviceName]
	if !ok {
		sc.sm[serviceName] = []Service{value}
		sc.len++
		return nil
	}

	// Check for existing playlist to update
	for i, s := range bucket {
		if s.RecipeName == recipeName {
			sc.sm[serviceName][i] = value
			return nil
		}
	}

	// No matching playlist found, add a new one
	sc.sm[serviceName] = append(bucket, value)
	sc.len++
	return nil
}

func (sc *ServiceCollection) Len() int {
	return sc.len
}

type iterator struct {
	sc   *ServiceCollection
	keys []string
	// Index of the current map key
	i int
	// Index of the current element in the bucket
	j int
}

func (sc *ServiceCollection) Iter() *iterator {
	it := iterator{
		sc:   sc,
		keys: make([]string, len(sc.sm)),
	}

	i := 0
	for k := range sc.sm {
		it.keys[i] = k
		i++
	}

	return &it
}

// Next returns the next element in the ServiceCollection currently being iterated.
// NOTE: This does not check if a next element exists. It is the callers responsibilty
// to ensure there is a next element using the HasNext() method.
func (it *iterator) Next() Service {
	// Get value at current index
	key := it.keys[it.i]
	bucket := it.sc.sm[key]
	val := bucket[it.j]

	// Update indices
	if it.j == len(bucket)-1 {
		// Move to next bucket
		it.i++
		it.j = 0
	} else {
		it.j++
	}

	return val
}

func (it *iterator) HasNext() bool {
	sm := it.sc.sm
	// Another element exists if i isn't on the last map key
	// and j isn't on the last bucket index
	return it.i < len(sm) && it.j < len(sm[it.keys[it.i]])
}
