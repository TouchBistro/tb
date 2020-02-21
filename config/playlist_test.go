package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlaylist(t *testing.T) {
	assert := assert.New(t)

	playlists = map[string][]Playlist{
		"core": []Playlist{
			Playlist{
				Services: []string{
					"postgres",
					"venue-core-service",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
			Playlist{
				Services: []string{
					"redis",
					"postgres",
					"node",
				},
				RecipeName: "ExampleZone/tb-recipe-examples",
			},
		},
		"vaf-core": []Playlist{
			Playlist{
				Extends: "TouchBistro/tb-recipe-services/core",
				Services: []string{
					"venue-admin-frontend",
					"partners-config-service",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
		},
	}

	tbrc.Playlists = map[string]Playlist{
		"my-core": {
			Extends: "vaf-core",
			Services: []string{
				"legacy-bridge-cloud-service",
				"loyalty-gateway-service",
				"postgres",
			},
		},
	}

	list, err := GetPlaylist("my-core")

	assert.ElementsMatch([]string{
		"postgres",
		"venue-core-service",
		"venue-admin-frontend",
		"partners-config-service",
		"legacy-bridge-cloud-service",
		"loyalty-gateway-service",
	}, list)
	assert.NoError(err)
}

func TestGetPlaylistCircularDependency(t *testing.T) {
	assert := assert.New(t)

	playlists = map[string][]Playlist{
		"core": []Playlist{
			Playlist{
				Extends: "core-2",
				Services: []string{
					"postgres",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
		},
		"core-2": []Playlist{
			Playlist{
				Extends: "TouchBistro/tb-recipe-services/core",
				Services: []string{
					"localstack",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
		},
	}

	list, err := GetPlaylist("core-2")

	assert.Empty(list)
	assert.Error(err)
}

func TestGetPlaylistNonexistent(t *testing.T) {
	assert := assert.New(t)

	playlists = map[string][]Playlist{}

	list, err := GetPlaylist("core")

	assert.Empty(list)
	assert.Error(err)
}

func TestGetPlaylistImplicitError(t *testing.T) {
	assert := assert.New(t)

	playlists = map[string][]Playlist{
		"core": []Playlist{
			Playlist{
				Services: []string{
					"postgres",
					"venue-core-service",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
			Playlist{
				Services: []string{
					"redis",
					"postgres",
					"node",
				},
				RecipeName: "ExampleZone/tb-recipe-examples",
			},
		},
	}

	list, err := GetPlaylist("core")

	assert.Empty(list)
	assert.Error(err)
}

func TestGetPlaylistInvalidName(t *testing.T) {
	assert := assert.New(t)

	playlists = map[string][]Playlist{
		"core": []Playlist{
			Playlist{
				Services: []string{
					"postgres",
					"venue-core-service",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
			Playlist{
				Services: []string{
					"redis",
					"postgres",
					"node",
				},
				RecipeName: "ExampleZone/tb-recipe-examples",
			},
		},
	}

	list, err := GetPlaylist("malformed/core")

	assert.Empty(list)
	assert.Error(err)
}

func TestGetPlaylistNonexistantRecipe(t *testing.T) {
	assert := assert.New(t)

	playlists = map[string][]Playlist{
		"core": []Playlist{
			Playlist{
				Services: []string{
					"postgres",
					"venue-core-service",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
			Playlist{
				Services: []string{
					"redis",
					"postgres",
					"node",
				},
				RecipeName: "ExampleZone/tb-recipe-examples",
			},
		},
	}

	list, err := GetPlaylist("DeadZone/tb-recipe-apps/core")

	assert.Empty(list)
	assert.Error(err)
}

func TestPlaylistNames(t *testing.T) {
	assert := assert.New(t)

	playlists = map[string][]Playlist{
		"core": []Playlist{
			Playlist{
				Services: []string{
					"postgres",
					"venue-core-service",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
		},
		"vaf-core": []Playlist{
			Playlist{
				Extends: "core",
				Services: []string{
					"venue-admin-frontend",
					"partners-config-service",
				},
				RecipeName: "TouchBistro/tb-recipe-services",
			},
		},
	}

	names := PlaylistNames()

	assert.ElementsMatch([]string{
		"TouchBistro/tb-recipe-services/core",
		"TouchBistro/tb-recipe-services/vaf-core",
	}, names)
}
