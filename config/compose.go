package config

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
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

func createComposeService(name string, service Service) composeService {
	s := composeService{
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

	var volumes []volume
	if service.UseRemote() {
		s.Command = service.Remote.Command
		s.Image = service.ImageURI()
		volumes = service.Remote.Volumes
	} else {
		s.Build = composeBuild{
			Args:    service.Build.Args,
			Context: service.Build.DockerfilePath,
			Target:  service.Build.Target,
		}

		s.Command = service.Build.Command
		volumes = service.Build.Volumes
	}

	for _, v := range volumes {
		s.Volumes = append(s.Volumes, v.Value)
	}

	return s
}

func CreateComposeFile(services ServiceMap, w io.Writer) error {
	composeServices := make(map[string]composeService)
	// Top level named volumes are an empty field, i.e. `postgres:`
	// There's no way to create an empty field with go-yaml
	// so we use interface{} and set it to nil which produces `postgres: null`
	// docker-compose seems cool with this
	composeVolumes := make(map[string]interface{})

	for name, service := range services {
		composeServices[name] = createComposeService(name, service)

		// Add named volumes
		var volumes []volume
		if service.UseRemote() {
			volumes = service.Remote.Volumes
		} else {
			volumes = service.Build.Volumes
		}
		for _, v := range volumes {
			if !v.IsNamed {
				continue
			}

			namedVolume := strings.Split(v.Value, ":")[0]
			composeVolumes[namedVolume] = nil
		}
	}

	composeFile := composeFile{
		Version:  "3.7",
		Services: composeServices,
		Volumes:  composeVolumes,
	}

	_, err := w.Write([]byte("# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY\n\n"))
	if err != nil {
		return errors.Wrap(err, "failed to write header comment to docker-compose yaml file")
	}

	err = yaml.NewEncoder(w).Encode(&composeFile)
	return errors.Wrap(err, "failed to encode docker-compose struct to yaml")
}
