package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createServiceCollection(t *testing.T) *ServiceCollection {
	// Creates two services with the same name but different registries
	// and one service that's a unique name
	sc, err := NewServiceCollection([]Service{
		Service{
			EnvVars: map[string]string{
				"POSTGRES_USER":     "user",
				"POSTGRES_PASSWORD": "password",
			},
			Mode: ModeRemote,
			Remote: Remote{
				Image: "postgres",
				Tag:   "12",
			},
			Name:         "postgres",
			RegistryName: "ExampleZone/tb-registry",
		},
		Service{
			EnvVars: map[string]string{
				"POSTGRES_USER":     "core",
				"POSTGRES_PASSWORD": "localdev",
			},
			Mode: ModeRemote,
			Remote: Remote{
				Image: "postgres",
				Tag:   "12-alpine",
			},
			Name:         "postgres",
			RegistryName: "TouchBistro/tb-registry",
		},
		Service{
			EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
			EnvVars: map[string]string{
				"HTTP_PORT": "8080",
			},
			Mode: ModeBuild,
			Ports: []string{
				"8081:8080",
			},
			PreRun:  "yarn db:prepare:dev",
			GitRepo: "TouchBistro/venue-core-service",
			Build: Build{
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
	}, nil)
	if err != nil {
		assert.FailNow(t, "Failed to create ServiceCollection")
	}

	return sc
}

func TestServiceMethods(t *testing.T) {
	assert := assert.New(t)

	s := Service{
		GitRepo: "TouchBistro/venue-core-service",
		Mode:    ModeRemote,
		Build: Build{
			DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
		},
		Remote: Remote{
			Image: "venue-core-service",
			Tag:   "master",
		},
		Name:         "venue-core-service",
		RegistryName: "TouchBistro/tb-registry",
	}

	assert.True(s.HasGitRepo())
	assert.True(s.UseRemote())
	assert.True(s.CanBuild())
	assert.Equal("venue-core-service:master", s.ImageURI())
	assert.Equal("TouchBistro/tb-registry/venue-core-service", s.FullName())
	assert.Equal("touchbistro-tb-registry-venue-core-service", s.DockerName())
}

func TestNewServiceCollectionOverrides(t *testing.T) {
	assert := assert.New(t)

	sc, err := NewServiceCollection([]Service{
		Service{
			EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
			EnvVars: map[string]string{
				"HTTP_PORT": "8080",
			},
			Mode: ModeBuild,
			Ports: []string{
				"8081:8080",
			},
			PreRun:  "yarn db:prepare:dev",
			GitRepo: "TouchBistro/venue-core-service",
			Build: Build{
				Args: map[string]string{
					"NODE_ENV": "development",
				},
				Command:        "yarn start",
				DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
				Target:         "release",
			},
			Remote: Remote{
				Image: "venue-core-service",
			},
			Name:         "venue-core-service",
			RegistryName: "TouchBistro/tb-registry",
		},
	}, map[string]ServiceOverride{
		"TouchBistro/tb-registry/venue-core-service": ServiceOverride{
			EnvVars: map[string]string{
				"LOGGER_LEVEL": "debug",
			},
			PreRun: "yarn db:prepare",
			Build: BuildOverride{
				Command: "yarn start:dev",
				Target:  "dev",
			},
			Remote: RemoteOverride{
				Command: "tail -f /dev/null",
				Enabled: true,
				Tag:     "master",
			},
		},
	})
	if err != nil {
		assert.FailNow("Failed to create ServiceCollection")
	}

	s, err := sc.Get("TouchBistro/tb-registry/venue-core-service")

	assert.Equal(Service{
		EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
		EnvVars: map[string]string{
			"HTTP_PORT":    "8080",
			"LOGGER_LEVEL": "debug",
		},
		Mode: ModeRemote,
		Ports: []string{
			"8081:8080",
		},
		PreRun:  "yarn db:prepare",
		GitRepo: "TouchBistro/venue-core-service",
		Build: Build{
			Args: map[string]string{
				"NODE_ENV": "development",
			},
			Command:        "yarn start:dev",
			DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
			Target:         "dev",
		},
		Remote: Remote{
			Command: "tail -f /dev/null",
			Image:   "venue-core-service",
			Tag:     "master",
		},
		Name:         "venue-core-service",
		RegistryName: "TouchBistro/tb-registry",
	}, s)
	assert.NoError(err)
}

func TestServiceCollectionGetFullName(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)

	s, err := sc.Get("TouchBistro/tb-registry/postgres")

	assert.Equal(Service{
		EnvVars: map[string]string{
			"POSTGRES_USER":     "core",
			"POSTGRES_PASSWORD": "localdev",
		},
		Mode: ModeRemote,
		Remote: Remote{
			Image: "postgres",
			Tag:   "12-alpine",
		},
		Name:         "postgres",
		RegistryName: "TouchBistro/tb-registry",
	}, s)
	assert.NoError(err)
}

func TestServiceCollectionGetShortName(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)

	s, err := sc.Get("venue-core-service")

	assert.Equal(Service{
		EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
		EnvVars: map[string]string{
			"HTTP_PORT": "8080",
		},
		Mode: ModeBuild,
		Ports: []string{
			"8081:8080",
		},
		PreRun:  "yarn db:prepare:dev",
		GitRepo: "TouchBistro/venue-core-service",
		Build: Build{
			Args: map[string]string{
				"NODE_ENV": "development",
			},
			Command:        "yarn start",
			DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
			Target:         "release",
		},
		Name:         "venue-core-service",
		RegistryName: "TouchBistro/tb-registry",
	}, s)
	assert.NoError(err)
}

func TestServiceCollectionGetShortError(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)

	s, err := sc.Get("postgres")

	assert.Zero(s)
	assert.Error(err)
}

func TestServiceCollectionGetNonexistent(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)

	s, err := sc.Get("TouchBistro/tb-registry/not-a-service")

	assert.Zero(s)
	assert.Error(err)
}

func TestServiceCollectionGetNoRecipe(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)

	s, err := sc.Get("ExampleZone/tb-registry/venue-core-service")

	assert.Zero(s)
	assert.Error(err)
}

func TestServiceCollectionGetInvalidName(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)

	s, err := sc.Get("Invalid/bad-name")

	assert.Zero(s)
	assert.Error(err)
}

func TestServiceCollectionLen(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)

	expectedLen := 3

	assert.Equal(expectedLen, sc.Len())
}

func TestServiceCollectionIter(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)
	it := sc.Iter()

	names := make([]string, 0)
	for it.HasNext() {
		s := it.Next()
		names = append(names, s.FullName())
	}

	expectedNames := []string{
		"ExampleZone/tb-registry/postgres",
		"TouchBistro/tb-registry/postgres",
		"TouchBistro/tb-registry/venue-core-service",
	}

	assert.ElementsMatch(expectedNames, names)
}
