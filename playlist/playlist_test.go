package playlist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlaylist(t *testing.T) {
	assert := assert.New(t)

	pc := NewPlaylistCollection(map[string]Playlist{
		"my-core": {
			Extends: "vaf-core",
			Services: []string{
				"legacy-bridge-cloud-service",
				"loyalty-gateway-service",
				"postgres",
			},
		},
	})

	err := pc.Set(Playlist{
		Services: []string{
			"postgres",
			"venue-core-service",
		},
		Name:       "core",
		RecipeName: "TouchBistro/tb-recipe",
	})
	if err != nil {
		assert.FailNow("Failed to set playlist")
	}

	err = pc.Set(Playlist{
		Services: []string{
			"redis",
			"postgres",
			"node",
		},
		Name:       "core",
		RecipeName: "ExampleZone/tb-recipe",
	})
	if err != nil {
		assert.FailNow("Failed to set playlist")
	}

	err = pc.Set(Playlist{
		Extends: "TouchBistro/tb-recipe/core",
		Services: []string{
			"venue-admin-frontend",
			"partners-config-service",
		},
		Name:       "vaf-core",
		RecipeName: "TouchBistro/tb-recipe",
	})
	if err != nil {
		assert.FailNow("Failed to set playlist")
	}

	list, err := pc.GetServices("my-core")
	expectedList := []string{
		"postgres",
		"venue-core-service",
		"venue-admin-frontend",
		"partners-config-service",
		"legacy-bridge-cloud-service",
		"loyalty-gateway-service",
	}

	assert.ElementsMatch(expectedList, list)
	assert.NoError(err)
}

func TestGetPlaylistCircularDependency(t *testing.T) {
	assert := assert.New(t)

	pc := NewPlaylistCollection(nil)
	err := pc.Set(Playlist{
		Extends: "core-2",
		Services: []string{
			"postgres",
		},
		Name:       "core",
		RecipeName: "TouchBistro/tb-recipe",
	})
	if err != nil {
		assert.FailNow("Failed to set playlist")
	}

	err = pc.Set(Playlist{
		Extends: "TouchBistro/tb-recipe/core",
		Services: []string{
			"localstack",
		},
		Name:       "core-2",
		RecipeName: "TouchBistro/tb-recipe",
	})
	if err != nil {
		assert.FailNow("Failed to set playlist")
	}

	list, err := pc.GetServices("core-2")

	assert.Empty(list)
	assert.Error(err)
}

func TestGetPlaylistNonexistent(t *testing.T) {
	assert := assert.New(t)

	pc := NewPlaylistCollection(nil)

	list, err := pc.GetServices("core")

	assert.Empty(list)
	assert.Error(err)
}

func TestNames(t *testing.T) {
	assert := assert.New(t)

	pc := NewPlaylistCollection(nil)

	err := pc.Set(Playlist{
		Services: []string{
			"postgres",
			"venue-core-service",
		},
		Name:       "core",
		RecipeName: "TouchBistro/tb-recipe",
	})
	if err != nil {
		assert.FailNow("Failed to set playlist")
	}

	err = pc.Set(Playlist{
		Services: []string{
			"redis",
			"postgres",
			"node",
		},
		Name:       "core",
		RecipeName: "ExampleZone/tb-recipe",
	})
	if err != nil {
		assert.FailNow("Failed to set playlist")
	}

	err = pc.Set(Playlist{
		Extends: "TouchBistro/tb-recipe/core",
		Services: []string{
			"venue-admin-frontend",
			"partners-config-service",
		},
		Name:       "vaf-core",
		RecipeName: "TouchBistro/tb-recipe",
	})
	if err != nil {
		assert.FailNow("Failed to set playlist")
	}

	names := pc.Names()
	expectedNames := []string{
		"TouchBistro/tb-recipe/core",
		"ExampleZone/tb-recipe/core",
		"TouchBistro/tb-recipe/vaf-core",
	}

	assert.ElementsMatch(expectedNames, names)
}

func TestCustomNames(t *testing.T) {
	assert := assert.New(t)

	pc := NewPlaylistCollection(map[string]Playlist{
		"my-core": Playlist{
			Extends: "TouchBistro/tb-recipe/core",
			Services: []string{
				"partners-config-service",
			},
		},
		"db": Playlist{
			Services: []string{
				"postgres",
			},
		},
	})

	names := pc.CustomNames()
	expectedNames := []string{"my-core", "db"}

	assert.ElementsMatch(expectedNames, names)
}
