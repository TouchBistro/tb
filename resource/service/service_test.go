package service_test

import (
	"errors"
	"testing"

	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/service"
	"github.com/matryer/is"
)

func TestServiceMethods(t *testing.T) {
	is := is.New(t)
	s := service.Service{
		GitRepo: service.GitRepo{
			Name: "TouchBistro/venue-core-service",
		},
		Mode: service.ModeRemote,
		Build: service.Build{
			DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
		},
		Remote: service.Remote{
			Image: "venue-core-service",
			Tag:   "master",
		},
		Name:         "venue-core-service",
		RegistryName: "TouchBistro/tb-registry",
	}
	is.True(s.HasGitRepo())
	is.True(s.CanBuild())
	is.Equal(s.ImageURI(), "venue-core-service:master")
	is.Equal(s.Type(), resource.TypeService)
	is.Equal(s.FullName(), "TouchBistro/tb-registry/venue-core-service")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		service    service.Service
		wantErr    bool
		wantMsgLen int
	}{
		{
			name: "valid",
			service: service.Service{
				EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
				EnvVars: map[string]string{
					"HTTP_PORT": "8080",
				},
				Mode: service.ModeBuild,
				Ports: []string{
					"8081:8080",
				},
				PreRun: "yarn db:prepare:dev",
				GitRepo: service.GitRepo{
					Name: "TouchBistro/venue-core-service",
				},
				Build: service.Build{
					Args: map[string]string{
						"NODE_ENV": "development",
					},
					Command:        "yarn start",
					DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
					Target:         "release",
				},
				Name:         "venue-core-service",
				RegistryName: "TouchBistro/tb-registry",
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			service: service.Service{
				Mode: "local",
				Build: service.Build{
					Args: map[string]string{
						"NODE_ENV": "development",
					},
					Command:        "yarn start",
					DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
					Target:         "release",
				},
				Name:         "venue-core-service",
				RegistryName: "TouchBistro/tb-registry",
			},
			wantErr:    true,
			wantMsgLen: 1,
		},
		{
			name: "no image for remote",
			service: service.Service{
				Mode: service.ModeRemote,
				Remote: service.Remote{
					Tag: "12-alpine",
				},
				Name:         "postgres",
				RegistryName: "TouchBistro/tb-registry",
			},
			wantErr:    true,
			wantMsgLen: 1,
		},
		{
			name: "no dockerfile path for build",
			service: service.Service{
				Mode:         service.ModeBuild,
				Name:         "postgres",
				RegistryName: "TouchBistro/tb-registry",
			},
			wantErr:    true,
			wantMsgLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			err := service.Validate(tt.service)
			if !tt.wantErr {
				is.NoErr(err)
				return
			}
			var validationErr *resource.ValidationError
			is.True(errors.As(err, &validationErr))
			is.Equal(validationErr.Resource, tt.service)
			is.Equal(len(validationErr.Messages), tt.wantMsgLen)
		})
	}
}

func TestOverride(t *testing.T) {
	is := is.New(t)
	s := service.Service{
		EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
		EnvVars: map[string]string{
			"HTTP_PORT": "8080",
		},
		Mode: service.ModeBuild,
		Ports: []string{
			"8081:8080",
		},
		PreRun: "yarn db:prepare:dev",
		GitRepo: service.GitRepo{
			Name: "TouchBistro/venue-core-service",
		},
		Build: service.Build{
			Args: map[string]string{
				"NODE_ENV": "development",
			},
			Command:        "yarn start",
			DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
			Target:         "release",
		},
		Remote: service.Remote{
			Image: "venue-core-service",
		},
		Name:         "venue-core-service",
		RegistryName: "TouchBistro/tb-registry",
	}
	o := service.ServiceOverride{
		EnvVars: map[string]string{
			"LOGGER_LEVEL": "debug",
		},
		Mode:   service.ModeRemote,
		PreRun: "yarn db:prepare",
		Build: service.BuildOverride{
			Command: "yarn start:dev",
			Target:  "dev",
		},
		Remote: service.RemoteOverride{
			Command: "tail -f /dev/null",
			Tag:     "master",
		},
	}

	overridden, err := service.Override(s, o)
	is.NoErr(err)
	is.Equal(overridden, service.Service{
		EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
		EnvVars: map[string]string{
			"HTTP_PORT":    "8080",
			"LOGGER_LEVEL": "debug",
		},
		Mode: service.ModeRemote,
		Ports: []string{
			"8081:8080",
		},
		PreRun: "yarn db:prepare",
		GitRepo: service.GitRepo{
			Name: "TouchBistro/venue-core-service",
		},
		Build: service.Build{
			Args: map[string]string{
				"NODE_ENV": "development",
			},
			Command:        "yarn start:dev",
			DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
			Target:         "dev",
		},
		Remote: service.Remote{
			Command: "tail -f /dev/null",
			Image:   "venue-core-service",
			Tag:     "master",
		},
		Name:         "venue-core-service",
		RegistryName: "TouchBistro/tb-registry",
	})
}

func TestOverrideError(t *testing.T) {
	tests := []struct {
		name     string
		service  service.Service
		override service.ServiceOverride
	}{
		{
			name: "invalid mode",
			service: service.Service{
				Mode: service.ModeRemote,
				Remote: service.Remote{
					Image: "postgres",
					Tag:   "12-alpine",
				},
				Name:         "postgres",
				RegistryName: "TouchBistro/tb-registry",
			},
			override: service.ServiceOverride{
				Mode: "local",
			},
		},
		{
			name: "cannot override to remote",
			service: service.Service{
				Mode:         service.ModeBuild,
				Name:         "postgres",
				RegistryName: "TouchBistro/tb-registry",
			},
			override: service.ServiceOverride{
				Mode: service.ModeRemote,
			},
		},
		{
			name: "cannot override to build",
			service: service.Service{
				EnvVars: map[string]string{
					"POSTGRES_USER":     "core",
					"POSTGRES_PASSWORD": "localdev",
				},
				Mode:         service.ModeRemote,
				Name:         "postgres",
				RegistryName: "TouchBistro/tb-registry",
			},
			override: service.ServiceOverride{
				Mode: service.ModeBuild,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			_, err := service.Override(tt.service, tt.override)
			is.True(err != nil)
		})
	}
}

func TestComposeConfig(t *testing.T) {
	services := []service.Service{
		{
			EnvVars: map[string]string{
				"POSTGRES_USER":     "user",
				"POSTGRES_PASSWORD": "password",
			},
			Mode: service.ModeRemote,
			Ports: []string{
				"5432:5432",
			},
			Remote: service.Remote{
				Image: "postgres",
				Tag:   "12",
				Volumes: []service.Volume{
					{
						Value:   "postgres:/var/lib/postgresql/data",
						IsNamed: true,
					},
				},
			},
			Name:         "postgres",
			RegistryName: "ExampleZone/tb-registry",
		},
		{
			EnvVars: map[string]string{
				"DB_USER":     "core",
				"DB_PASSWORD": "localdev",
			},
			Mode: service.ModeRemote,
			Ports: []string{
				"5432:5432",
			},
			Remote: service.Remote{
				Image: "postgres",
				Tag:   "10.6-alpine",
				Volumes: []service.Volume{
					{
						Value:   "postgres:/var/lib/postgresql/data",
						IsNamed: true,
					},
				},
			},
			Name:         "postgres",
			RegistryName: "TouchBistro/tb-registry",
		},
		{
			Dependencies: []string{
				"touchbistro-tb-registry-postgres",
			},
			EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
			EnvVars: map[string]string{
				"HTTP_PORT": "8080",
				"DB_HOST":   "touchbistro-tb-registry-postgres",
			},
			Mode: service.ModeBuild,
			Ports: []string{
				"8081:8080",
			},
			PreRun: "yarn db:prepare",
			GitRepo: service.GitRepo{
				Name: "TouchBistro/venue-core-service",
			},
			Build: service.Build{
				Args: map[string]string{
					"NODE_ENV":       "development",
					"NPM_READ_TOKEN": "$NPM_READ_TOKEN",
				},
				Command:        "yarn start",
				DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
				Target:         "dev",
				Volumes: []service.Volume{
					{
						Value: ".tb/repos/TouchBistro/venue-core-service:/home/node/app:delegated",
					},
				},
			},
			Name:         "venue-core-service",
			RegistryName: "TouchBistro/tb-registry",
		},
	}
	var c resource.Collection[service.Service]
	for _, s := range services {
		if err := c.Set(s); err != nil {
			t.Fatalf("failed to add service %s to collection: %v", s.FullName(), err)
		}
	}

	wantComposeConfig := docker.ComposeConfig{
		Version: "3.7",
		Services: map[string]docker.ComposeServiceConfig{
			"examplezone-tb-registry-postgres": {
				ContainerName: "examplezone-tb-registry-postgres",
				Environment: map[string]string{
					"POSTGRES_PASSWORD": "password",
					"POSTGRES_USER":     "user",
				},
				Image:   "postgres:12",
				Ports:   []string{"5432:5432"},
				Volumes: []string{"postgres:/var/lib/postgresql/data"},
			},
			"touchbistro-tb-registry-postgres": {
				ContainerName: "touchbistro-tb-registry-postgres",
				Environment: map[string]string{
					"DB_PASSWORD": "localdev",
					"DB_USER":     "core",
				},
				Image:   "postgres:10.6-alpine",
				Ports:   []string{"5432:5432"},
				Volumes: []string{"postgres:/var/lib/postgresql/data"},
			},
			"touchbistro-tb-registry-venue-core-service": {
				Build: docker.ComposeBuildConfig{
					Args: map[string]string{
						"NODE_ENV":       "development",
						"NPM_READ_TOKEN": "$NPM_READ_TOKEN",
					},
					Context: ".tb/repos/TouchBistro/venue-core-service",
					Target:  "dev",
				},
				Command:       "yarn start",
				ContainerName: "touchbistro-tb-registry-venue-core-service",
				DependsOn:     []string{"touchbistro-tb-registry-postgres"},
				EnvFile:       []string{".tb/repos/TouchBistro/venue-core-service/.env.example"},
				Environment: map[string]string{
					"DB_HOST":   "touchbistro-tb-registry-postgres",
					"HTTP_PORT": "8080",
				},
				Ports:   []string{"8081:8080"},
				Volumes: []string{".tb/repos/TouchBistro/venue-core-service:/home/node/app:delegated"},
			},
		},
		Volumes: map[string]interface{}{"postgres": nil},
	}
	composeConfig := service.ComposeConfig(&c)
	is := is.New(t)
	is.Equal(composeConfig, wantComposeConfig)
}
