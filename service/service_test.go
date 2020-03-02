package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createServiceCollection(t *testing.T) *ServiceCollection {
	sc := NewServiceCollection()
	err := sc.Set(Service{
		EnvVars: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
		},
		Remote: Remote{
			Enabled: true,
			Image:   "postgres",
			Tag:     "12",
		},
		Name:       "postgres",
		RecipeName: "ExampleZone/tb-recipe",
	})
	if err != nil {
		assert.FailNow(t, "Failed to set playlist")
	}

	err = sc.Set(Service{
		EnvVars: map[string]string{
			"POSTGRES_USER":     "core",
			"POSTGRES_PASSWORD": "localdev",
		},
		Remote: Remote{
			Enabled: true,
			Image:   "postgres",
			Tag:     "12-alpine",
		},
		Name:       "postgres",
		RecipeName: "TouchBistro/tb-recipe",
	})
	if err != nil {
		assert.FailNow(t, "Failed to set playlist")
	}

	err = sc.Set(Service{
		EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
		EnvVars: map[string]string{
			"HTTP_PORT": "8080",
		},
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
		Name:       "venue-core-service",
		RecipeName: "TouchBistro/tb-recipe",
	})
	if err != nil {
		assert.FailNow(t, "Failed to set playlist")
	}

	return sc
}

func TestServiceCollectionGetFullName(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)

	s, err := sc.Get("TouchBistro/tb-recipe/postgres")

	assert.Equal(Service{
		EnvVars: map[string]string{
			"POSTGRES_USER":     "core",
			"POSTGRES_PASSWORD": "localdev",
		},
		Remote: Remote{
			Enabled: true,
			Image:   "postgres",
			Tag:     "12-alpine",
		},
		Name:       "postgres",
		RecipeName: "TouchBistro/tb-recipe",
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
		Name:       "venue-core-service",
		RecipeName: "TouchBistro/tb-recipe",
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

	s, err := sc.Get("TouchBistro/tb-recipe/not-a-service")

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

func TestServiceCollectionSetUpdate(t *testing.T) {
	assert := assert.New(t)
	sc := createServiceCollection(t)
	name := "TouchBistro/tb-recipe/postgres"

	s, err := sc.Get(name)
	if err != nil {
		assert.FailNow("Failed to get service")
	}
	s.PreRun = "setup_db.sh -v"

	err = sc.Set(s)

	assert.NoError(err)

	s, err = sc.Get(name)

	assert.Equal(Service{
		EnvVars: map[string]string{
			"POSTGRES_USER":     "core",
			"POSTGRES_PASSWORD": "localdev",
		},
		PreRun: "setup_db.sh -v",
		Remote: Remote{
			Enabled: true,
			Image:   "postgres",
			Tag:     "12-alpine",
		},
		Name:       "postgres",
		RecipeName: "TouchBistro/tb-recipe",
	}, s)
	assert.NoError(err)
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
		"ExampleZone/tb-recipe/postgres",
		"TouchBistro/tb-recipe/postgres",
		"TouchBistro/tb-recipe/venue-core-service",
	}

	assert.ElementsMatch(expectedNames, names)
}
