package config

import (
	"github.com/pkg/errors"
)

type Playlist struct {
	Extends    string   `yaml:"extends"`
	Services   []string `yaml:"services"`
	RecipeName string   `yaml:"-"`
}

type PlaylistMap map[string]Playlist

func getPlaylistServices(name string, deps map[string]bool) ([]string, error) {
	var playlist Playlist
	recipeName, playlistName, err := recipeNameParts(name)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid playlist name %s", name)
	}

	// Check custom playlists first
	if p, ok := tbrc.Playlists[playlistName]; ok {
		playlist = p
	} else if list, ok := playlists[playlistName]; ok {
		// Handle shorthand case
		if recipeName == "" {
			if len(list) > 1 {
				return nil, errors.Errorf("Multiple playlists named %s found. Please specify the recipe the playlist belongs to.", playlistName)
			}

			playlist = list[0]
			name = joinNameParts(playlist.RecipeName, playlistName)
		} else {
			// Find playlist with matching recipe
			found := false
			for _, p := range list {
				if p.RecipeName == recipeName {
					playlist = p
					found = true
					break
				}
			}

			if !found {
				return nil, errors.Errorf("No such playlist %s", name)
			}
		}
	} else {
		return nil, errors.Errorf("No such playlist %s", playlistName)
	}

	if playlist.Extends == "" {
		return playlist.Services, nil
	}

	// Resolve parent playlist defined in extends
	deps[name] = true
	if deps[playlist.Extends] {
		return nil, errors.Errorf("Circular dependency of services, %s and %s", playlist.Extends, name)
	}

	parentPlaylist, err := getPlaylistServices(playlist.Extends, deps)
	return append(parentPlaylist, playlist.Services...), err
}

func GetPlaylist(name string) ([]string, error) {
	services, err := getPlaylistServices(name, make(map[string]bool))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get services of playlist %s", name)
	}

	// Remove duplicate services
	seenServices := make(map[string]bool)
	uniqueServices := make([]string, 0)
	for _, serviceName := range services {
		if _, ok := seenServices[serviceName]; ok {
			continue
		}

		seenServices[serviceName] = true
		uniqueServices = append(uniqueServices, serviceName)
	}

	return uniqueServices, nil
}

func PlaylistNames() []string {
	names := make([]string, 0)
	for n, list := range playlists {
		for _, p := range list {
			names = append(names, joinNameParts(p.RecipeName, n))
		}
	}

	return names
}
