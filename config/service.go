package config

import (
	"fmt"
	"path/filepath"

	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

/* Types */

type ServiceMap map[string]service.Service

type ServiceConfig struct {
	Global struct {
		BaseImages     []string          `yaml:"baseImages"`
		LoginStategies []string          `yaml:"loginStrategies"`
		Variables      map[string]string `yaml:"variables"`
	} `yaml:"global"`
	Services ServiceMap `yaml:"services"`
}

/* Private helpers */

func parseServices(config ServiceConfig) (ServiceMap, error) {
	parsedServices := make(ServiceMap)

	// Validate each service and perform any necessary actions
	for name, service := range config.Services {
		// Make sure either local or remote usage is specified
		if !service.CanBuild() && service.Remote.Image == "" {
			msg := fmt.Sprintf("Must specify at least one of 'build.dockerfilePath' or 'remote.image' for service %s", name)
			return nil, errors.New(msg)
		}

		// Make sure repo is specified if not using remote
		if !service.UseRemote() && !service.CanBuild() {
			msg := fmt.Sprintf("'remote.enabled: false' is set but 'build.dockerfilePath' was not provided for service %s", name)
			return nil, errors.New(msg)
		}

		// Set special service specific vars
		vars := config.Global.Variables
		vars["@ROOTPATH"] = TBRootPath()

		if service.HasGitRepo() {
			vars["@REPOPATH"] = filepath.Join(ReposPath(), service.GitRepo)
		}

		// Expand any vars
		service.Build.DockerfilePath = util.ExpandVars(service.Build.DockerfilePath, vars)
		service.EnvFile = util.ExpandVars(service.EnvFile, vars)
		service.Remote.Image = util.ExpandVars(service.Remote.Image, vars)

		for key, value := range service.EnvVars {
			service.EnvVars[key] = util.ExpandVars(value, vars)
		}

		for i, volume := range service.Build.Volumes {
			service.Build.Volumes[i].Value = util.ExpandVars(volume.Value, vars)
		}

		for i, volume := range service.Remote.Volumes {
			service.Remote.Volumes[i].Value = util.ExpandVars(volume.Value, vars)
		}

		parsedServices[name] = service
	}

	return parsedServices, nil
}

func applyOverrides(services ServiceMap, overrides map[string]ServiceOverride) (ServiceMap, error) {
	newServices := make(ServiceMap)
	for name, s := range services {
		newServices[name] = s
	}

	for name, override := range overrides {
		s, ok := services[name]
		if !ok {
			return nil, fmt.Errorf("%s is not a valid service", name)
		}

		// Validate overrides
		if override.Remote.Enabled && s.Remote.Image == "" {
			msg := fmt.Sprintf("remote.enabled is overridden to true for %s but it is not available from a remote source", name)
			return nil, errors.New(msg)
		} else if !override.Remote.Enabled && !s.HasGitRepo() {
			msg := fmt.Sprintf("remote.enabled is overridden to false but %s cannot be built locally", name)
			return nil, errors.New(msg)
		}

		// Apply overrides to service
		if override.Build.Command != "" {
			s.Build.Command = override.Build.Command
		}

		if override.Build.Target != "" {
			s.Build.Target = override.Build.Target
		}

		if override.EnvVars != nil {
			for v, val := range override.EnvVars {
				s.EnvVars[v] = val
			}
		}

		if override.PreRun != "" {
			s.PreRun = override.PreRun
		}

		if override.Remote.Command != "" {
			s.Remote.Command = override.Remote.Command
		}

		s.Remote.Enabled = override.Remote.Enabled
		if override.Remote.Tag != "" {
			s.Remote.Tag = override.Remote.Tag
		}

		newServices[name] = s
	}

	return newServices, nil
}
