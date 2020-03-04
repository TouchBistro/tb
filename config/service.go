package config

import (
	"fmt"
	"path/filepath"

	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

/* Types */

// Legacy, don't use!
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

	vars := config.Global.Variables
	vars["@ROOTPATH"] = TBRootPath()

	// Add vars for each service name
	for name := range config.Services {
		vars["@"+name] = "touchbistro-tb-registry-" + name
	}

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
		if service.HasGitRepo() {
			vars["@REPOPATH"] = filepath.Join(ReposPath(), service.GitRepo)
		} else {
			vars["@REPOPATH"] = ""
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
