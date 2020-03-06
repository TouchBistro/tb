package playlist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlaylist(t *testing.T) {
	assert := assert.New(t)

	pc, err := NewPlaylistCollection([]Playlist{
		Playlist{
			Services: []string{
				"postgres",
				"venue-core-service",
			},
			Name:         "core",
			RegistryName: "TouchBistro/tb-registry",
		},
		Playlist{
			Services: []string{
				"redis",
				"postgres",
				"node",
			},
			Name:         "core",
			RegistryName: "ExampleZone/tb-registry",
		},
		Playlist{
			Extends: "TouchBistro/tb-registry/core",
			Services: []string{
				"venue-admin-frontend",
				"partners-config-service",
			},
			Name:         "vaf-core",
			RegistryName: "TouchBistro/tb-registry",
		},
	}, map[string]Playlist{
		"my-core": {
			Extends: "vaf-core",
			Services: []string{
				"legacy-bridge-cloud-service",
				"loyalty-gateway-service",
				"postgres",
			},
		},
	})
	if err != nil {
		assert.FailNow("Failed to create PlaylistCollection")
	}

	list, err := pc.ServiceNames("my-core")
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

	pc, err := NewPlaylistCollection([]Playlist{
		Playlist{
			Extends: "core-2",
			Services: []string{
				"postgres",
			},
			Name:         "core",
			RegistryName: "TouchBistro/tb-registry",
		},
		Playlist{
			Extends: "TouchBistro/tb-registry/core",
			Services: []string{
				"localstack",
			},
			Name:         "core-2",
			RegistryName: "TouchBistro/tb-registry",
		},
	}, nil)
	if err != nil {
		assert.FailNow("Failed to create PlaylistCollection")
	}

	list, err := pc.ServiceNames("core-2")

	assert.Empty(list)
	assert.Error(err)
}

func TestGetPlaylistNonexistent(t *testing.T) {
	assert := assert.New(t)

	pc, err := NewPlaylistCollection(nil, nil)
	if err != nil {
		assert.FailNow("Failed to create PlaylistCollection")
	}

	list, err := pc.ServiceNames("core")

	assert.Empty(list)
	assert.Error(err)
}

func TestNames(t *testing.T) {
	assert := assert.New(t)

	pc, err := NewPlaylistCollection([]Playlist{
		Playlist{
			Services: []string{
				"postgres",
				"venue-core-service",
			},
			Name:         "core",
			RegistryName: "TouchBistro/tb-registry",
		},
		Playlist{
			Services: []string{
				"redis",
				"postgres",
				"node",
			},
			Name:         "core",
			RegistryName: "ExampleZone/tb-registry",
		},
		Playlist{
			Extends: "TouchBistro/tb-registry/core",
			Services: []string{
				"venue-admin-frontend",
				"partners-config-service",
			},
			Name:         "vaf-core",
			RegistryName: "TouchBistro/tb-registry",
		},
	}, nil)
	if err != nil {
		assert.FailNow("Failed to create PlaylistCollection")
	}

	names := pc.Names()
	expectedNames := []string{
		"TouchBistro/tb-registry/core",
		"ExampleZone/tb-registry/core",
		"TouchBistro/tb-registry/vaf-core",
	}

	assert.ElementsMatch(expectedNames, names)
}

func TestCustomNames(t *testing.T) {
	assert := assert.New(t)

	pc, err := NewPlaylistCollection(nil, map[string]Playlist{
		"my-core": Playlist{
			Extends: "TouchBistro/tb-registry/core",
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
	if err != nil {
		assert.FailNow("Failed to create PlaylistCollection")
	}

	names := pc.CustomNames()
	expectedNames := []string{"my-core", "db"}

	assert.ElementsMatch(expectedNames, names)
}
