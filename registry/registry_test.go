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

	ezPostgres, err := result.Services.Get("ExampleZone/tb-registry/postgres")
	if err != nil {
		assert.FailNow("Failed to get service ExampleZone/tb-registry/postgres")
	}

	assert.Equal(service.Service{
		EnvVars: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
		},
		Mode:  service.ModeRemote,
		Ports: []string{"5432:5432"},
		Remote: service.Remote{
			Image: "postgres",
			Tag:   "12",
			Volumes: []service.Volume{
				service.Volume{
					Value:   "postgres:/var/lib/postgresql/data",
					IsNamed: true,
				},
			},
		},
		Name:         "postgres",
		RegistryName: "ExampleZone/tb-registry",
	}, ezPostgres)

	ves, err := result.Services.Get("venue-example-service")
	if err != nil {
		assert.FailNow("Failed to get service venue-example-service")
	}

	assert.Equal(service.Service{
		Entrypoint: []string{"bash", "entrypoints/docker.sh"},
		EnvFile:    "/home/test/.tb/repos/ExampleZone/venue-example-service/.env.compose",
		EnvVars: map[string]string{
			"HTTP_PORT":     "8000",
			"POSTGRES_HOST": "examplezone-tb-registry-postgres",
		},
		Mode:    service.ModeRemote,
		Ports:   []string{"9000:8000"},
		PreRun:  "yarn db:prepare:dev",
		GitRepo: "ExampleZone/venue-example-service",
		Build: service.Build{
			Command:        "yarn start",
			DockerfilePath: "/home/test/.tb/repos/ExampleZone/venue-example-service",
			Target:         "build",
		},
		Remote: service.Remote{
			Command: "yarn serve",
			Image:   "98765.dkr.ecr.us-east-1.amazonaws.com/venue-example-service",
			Tag:     "staging",
		},
		Name:         "venue-example-service",
		RegistryName: "ExampleZone/tb-registry",
	}, ves)

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

	ezCorePlaylist, err := result.Playlists.Get("ExampleZone/tb-registry/core")
	if err != nil {
		assert.FailNow("Failed to get ExampleZone/tb-registry/core playlist")
	}

	assert.Equal(playlist.Playlist{
		Services: []string{
			"ExampleZone/tb-registry/postgres",
		},
		Name:         "core",
		RegistryName: "ExampleZone/tb-registry",
	}, ezCorePlaylist)

	ezExampleZonePlaylist, err := result.Playlists.Get("example-zone")
	if err != nil {
		assert.FailNow("Failed to get example-zone playlist")
	}

	assert.Equal(playlist.Playlist{
		Extends: "ExampleZone/tb-registry/core",
		Services: []string{
			"ExampleZone/tb-registry/venue-example-service",
		},
		Name:         "example-zone",
		RegistryName: "ExampleZone/tb-registry",
	}, ezExampleZonePlaylist)
}

func TestValidate(t *testing.T) {
	assert := assert.New(t)

	err := Validate("testdata/registry-1")

	assert.NoError(err)
}

func TestValidateInvalidService(t *testing.T) {
	assert := assert.New(t)

	err := Validate("testdata/invalid-registry-1")

	assert.Error(err)
}
