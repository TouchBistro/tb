package registry

import (
	"testing"

	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/service"
	"github.com/stretchr/testify/assert"
)

func TestReadServices(t *testing.T) {
	assert := assert.New(t)

	services, conf, err := ReadServices(Registry{
		Name: "TouchBistro/tb-registry",
		Path: "testdata/registry-1",
	}, "/home/test/.tb", "/home/test/.tb/repos")

	expectedServices := []service.Service{
		service.Service{
			EnvVars: map[string]string{
				"POSTGRES_USER":     "core",
				"POSTGRES_PASSWORD": "localdev",
			},
			Mode:  service.ModeRemote,
			Ports: []string{"5432:5432"},
			Remote: service.Remote{
				Image: "postgres",
				Tag:   "10.6-alpine",
				Volumes: []service.Volume{
					service.Volume{
						Value:   "postgres:/var/lib/postgresql/data",
						IsNamed: true,
					},
				},
			},
			Name:         "postgres",
			RegistryName: "TouchBistro/tb-registry",
		},
		service.Service{
			Dependencies: []string{
				"touchbistro-tb-registry-postgres",
			},
			EnvFile: "/home/test/.tb/repos/TouchBistro/venue-core-service/.env.example",
			EnvVars: map[string]string{
				"HTTP_PORT": "8080",
				"DB_HOST":   "touchbistro-tb-registry-postgres",
			},
			Mode:    service.ModeRemote,
			Ports:   []string{"8081:8080"},
			PreRun:  "yarn db:prepare",
			GitRepo: "TouchBistro/venue-core-service",
			Build: service.Build{
				Args: map[string]string{
					"NODE_ENV":  "development",
					"NPM_TOKEN": "$NPM_TOKEN",
				},
				Command:        "yarn start",
				DockerfilePath: "/home/test/.tb/repos/TouchBistro/venue-core-service",
				Target:         "dev",
				Volumes: []service.Volume{
					service.Volume{
						Value: "/home/test/.tb/repos/TouchBistro/venue-core-service:/home/node/app:delegated",
					},
				},
			},
			Remote: service.Remote{
				Command: "yarn serve",
				Image:   "12345.dkr.ecr.us-east-1.amazonaws.com/venue-core-service",
				Tag:     "master",
			},
			Name:         "venue-core-service",
			RegistryName: "TouchBistro/tb-registry",
		},
	}
	expectedConf := GlobalConfig{
		BaseImages: []string{
			"swift",
			"touchbistro/alpine-node:12-runtime",
		},
		LoginStrategies: []string{"ecr", "npm"},
	}

	assert.ElementsMatch(expectedServices, services)
	assert.Equal(expectedConf, conf)
	assert.NoError(err)
}

func TestReadPlaylists(t *testing.T) {
	assert := assert.New(t)

	playlists, err := ReadPlaylists(Registry{
		Name: "TouchBistro/tb-registry",
		Path: "testdata/registry-1",
	})

	expectedPlaylists := []playlist.Playlist{
		playlist.Playlist{
			Services: []string{
				"TouchBistro/tb-registry/postgres",
			},
			Name:         "db",
			RegistryName: "TouchBistro/tb-registry",
		},
		playlist.Playlist{
			Extends: "TouchBistro/tb-registry/db",
			Services: []string{
				"TouchBistro/tb-registry/venue-core-service",
			},
			Name:         "core",
			RegistryName: "TouchBistro/tb-registry",
		},
	}

	assert.ElementsMatch(expectedPlaylists, playlists)
	assert.NoError(err)
}
