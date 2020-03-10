package registry

import (
	"testing"

	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/service"
	"github.com/stretchr/testify/assert"
)

func TestReadRegistries(t *testing.T) {
	assert := assert.New(t)

	registries := []Registry{
		Registry{
			Name: "TouchBistro/tb-registry",
			Path: "testdata/registry-1",
		},
		Registry{
			Name: "ExampleZone/tb-registry",
			Path: "testdata/registry-2",
		},
	}
	result, err := ReadRegistries(registries, ReadOptions{
		ShouldReadServices: true,
		RootPath:           "/home/test/.tb",
		ReposPath:          "/home/test/.tb/repos",
		Overrides:          nil,
		CustomPlaylists:    nil,
	})

	assert.NoError(err)

	expectedBaseImages := []string{
		"swift",
		"touchbistro/alpine-node:12-runtime",
		"alpine-node",
	}
	expectedLoginStrategies := []string{"ecr", "npm"}

	assert.ElementsMatch(expectedBaseImages, result.BaseImages)
	assert.ElementsMatch(expectedLoginStrategies, result.LoginStrategies)

	tbPostgres, err := result.Services.Get("TouchBistro/tb-registry/postgres")
	if err != nil {
		assert.FailNow("Failed to get service TouchBistro/tb-registry/postgres")
	}

	assert.Equal(service.Service{
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
	}, tbPostgres)

	vcs, err := result.Services.Get("venue-core-service")
	if err != nil {
		assert.FailNow("Failed to get service venue-core-service")
	}

	assert.Equal(service.Service{
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
	}, vcs)

	dbPlaylist, err := result.Playlists.Get("db")
	if err != nil {
		assert.FailNow("Failed to get db playlist")
	}

	assert.Equal(playlist.Playlist{
		Services: []string{
			"TouchBistro/tb-registry/postgres",
		},
		Name:         "db",
		RegistryName: "TouchBistro/tb-registry",
	}, dbPlaylist)

	tbCorePlayist, err := result.Playlists.Get("TouchBistro/tb-registry/core")
	if err != nil {
		assert.FailNow("Failed to get TouchBistro/tb-registry/core playlist")
	}

	assert.Equal(playlist.Playlist{
		Extends: "TouchBistro/tb-registry/db",
		Services: []string{
			"TouchBistro/tb-registry/venue-core-service",
		},
		Name:         "core",
		RegistryName: "TouchBistro/tb-registry",
	}, tbCorePlayist)
}
