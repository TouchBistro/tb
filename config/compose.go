package config

import (
	"strings"
)

type composeBuild struct {
	Args    map[string]string `yaml:"args,omitempty"`
	Context string            `yaml:"context,omitempty"`
	Target  string            `yaml:"target,omitempty"`
}

type composeService struct {
	Build         composeBuild      `yaml:"build,omitempty"` // non-remote
	Command       string            `yaml:"command,omitempty"`
	ContainerName string            `yaml:"container_name"`
	DependsOn     []string          `yaml:"depends_on,omitempty"`
	Entrypoint    []string          `yaml:"entrypoint,omitempty"`
	EnvFile       []string          `yaml:"env_file,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Image         string            `yaml:"image,omitempty"` // remote
	Ports         []string          `yaml:"ports,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
}

type composeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]composeService `yaml:"services"`
	Volumes  map[string]interface{}    `yaml:"volumes,omitempty"`
}

func generateComposeService(name string, service Service) composeService {
	s := composeService{
		Command:       service.Command,
		ContainerName: name,
		DependsOn:     service.Dependencies,
		Entrypoint:    service.Entrypoint,
		EnvFile:       []string{},
		Environment:   service.EnvVars,
		Ports:         service.Ports,
	}

	if service.EnvFile != "" {
		s.EnvFile = append(s.EnvFile, service.EnvFile)
	}

	if service.UseRemote() {
		s.Image = service.ImageURI()
	} else {
		s.Build = composeBuild{
			Args:    service.Build.Args,
			Context: service.Build.DockerfilePath,
			Target:  service.Build.Target,
		}

		// Override command if custom command is set for build
		if service.Build.Command != "" {
			s.Command = service.Build.Command
		}
	}

	for _, volume := range service.Volumes {
		if service.UseRemote() && !volume.IsForRemote {
			continue
		}

		s.Volumes = append(s.Volumes, volume.Value)
	}

	return s
}

func generateComposeFile(services ServiceMap) composeFile {
	composeServices := make(map[string]composeService)
	volumes := make(map[string]interface{})

	for name, service := range services {
		composeServices[name] = generateComposeService(name, service)

		// Add named volumes
		for _, volume := range service.Volumes {
			if !volume.IsNamed {
				continue
			}

			namedVolume := strings.Split(volume.Value, ":")[0]
			volumes[namedVolume] = nil
		}
	}

	return composeFile{
		Version:  "3.7",
		Services: composeServices,
		Volumes:  volumes,
	}
}
