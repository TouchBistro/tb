package playlist_test

import (
	"sort"
	"testing"

	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/matryer/is"
)

func newCollection(t *testing.T, playlists, customPlaylists []playlist.Playlist) *playlist.Collection {
	t.Helper()
	var c playlist.Collection
	for _, p := range playlists {
		if err := c.Set(p); err != nil {
			t.Fatalf("failed to add playlist %s to collection: %v", p.FullName(), err)
		}
	}
	for _, p := range customPlaylists {
		c.SetCustom(p)
	}
	return &c
}

func TestServiceNames(t *testing.T) {
	is := is.New(t)
	c := newCollection(t, []playlist.Playlist{
		{
			Services: []string{
				"postgres",
				"venue-core-service",
			},
			Name:         "core",
			RegistryName: "TouchBistro/tb-registry",
		},
		{
			Services: []string{
				"redis",
				"postgres",
				"node",
			},
			Name:         "core",
			RegistryName: "ExampleZone/tb-registry",
		},
		{
			Extends: "TouchBistro/tb-registry/core",
			Services: []string{
				"venue-admin-frontend",
				"partners-config-service",
			},
			Name:         "vaf-core",
			RegistryName: "TouchBistro/tb-registry",
		},
	}, []playlist.Playlist{
		{
			Extends: "vaf-core",
			Services: []string{
				"legacy-bridge-cloud-service",
				"loyalty-gateway-service",
				"postgres",
			},
			Name: "my-core",
		},
	})

	list, err := c.ServiceNames("my-core")
	expectedList := []string{
		"postgres",
		"venue-core-service",
		"venue-admin-frontend",
		"partners-config-service",
		"legacy-bridge-cloud-service",
		"loyalty-gateway-service",
	}
	is.Equal(list, expectedList)
	is.NoErr(err)
}

func TestServiceNamesCircularDependency(t *testing.T) {
	is := is.New(t)
	c := newCollection(t, []playlist.Playlist{
		{
			Extends: "core-2",
			Services: []string{
				"postgres",
			},
			Name:         "core",
			RegistryName: "TouchBistro/tb-registry",
		},
		{
			Extends: "TouchBistro/tb-registry/core",
			Services: []string{
				"localstack",
			},
			Name:         "core-2",
			RegistryName: "TouchBistro/tb-registry",
		},
	}, nil)

	list, err := c.ServiceNames("core-2")
	is.Equal(len(list), 0)
	is.True(err != nil)
}

func TestServiceNamesNonexistent(t *testing.T) {
	is := is.New(t)
	c := newCollection(t, nil, nil)

	list, err := c.ServiceNames("core")
	is.Equal(len(list), 0)
	is.True(err != nil)
}

func TestNames(t *testing.T) {
	is := is.New(t)
	c := newCollection(t, []playlist.Playlist{
		{
			Services: []string{
				"postgres",
				"venue-core-service",
			},
			Name:         "core",
			RegistryName: "TouchBistro/tb-registry",
		},
		{
			Services: []string{
				"redis",
				"postgres",
				"node",
			},
			Name:         "core",
			RegistryName: "ExampleZone/tb-registry",
		},
		{
			Extends: "TouchBistro/tb-registry/core",
			Services: []string{
				"venue-admin-frontend",
				"partners-config-service",
			},
			Name:         "vaf-core",
			RegistryName: "TouchBistro/tb-registry",
		},
	}, nil)

	names := c.Names()
	sort.Strings(names)
	is.Equal(names, []string{
		"ExampleZone/tb-registry/core",
		"TouchBistro/tb-registry/core",
		"TouchBistro/tb-registry/vaf-core",
	})
}

func TestCustomNames(t *testing.T) {
	is := is.New(t)
	c := newCollection(t, nil, []playlist.Playlist{
		{
			Extends: "TouchBistro/tb-registry/core",
			Services: []string{
				"partners-config-service",
			},
			Name: "my-core",
		},
		{
			Services: []string{
				"postgres",
			},
			Name: "db",
		},
	})

	names := c.CustomNames()
	sort.Strings(names)
	is.Equal(names, []string{"db", "my-core"})
}
