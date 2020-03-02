package service

import (
	"fmt"

	"github.com/pkg/errors"
)

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

func (s Service) ApplyOverride(o ServiceOverride) (Service, error) {
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

func (sc *ServiceCollection) ApplyOverrides(overrides map[string]ServiceOverride) error {
	for name, override := range overrides {
		s, err := sc.Get(name)
		if err != nil {
			return errors.Errorf("failed to get service %s to apply override to", name)
		}

		s, err = s.ApplyOverride(override)
		if err != nil {
			return errors.Errorf("failed to apply override to service %s", s.FullName())
		}

		// Name and RecipeName are validated before a service is added to the ServiceCollection
		// so it's safe to use them to directly update the service
		for i, el := range sc.sm[s.Name] {
			if el.RecipeName == s.RecipeName {
				sc.sm[s.Name][i] = s
				break
			}
		}
	}

	return nil
}
