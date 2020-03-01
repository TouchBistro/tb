package playlist

import (
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

type Playlist struct {
	Extends  string   `yaml:"extends,omitempty"`
	Services []string `yaml:"services"`
	// Not part of yaml, set at runtime
	Name       string `yaml:"-"`
	RecipeName string `yaml:"-"`
}

func (p Playlist) FullName() string {
	return util.JoinNameParts(p.RecipeName, p.Name)
}

type PlaylistCollection struct {
	playlists       map[string][]Playlist
	customPlaylists map[string]Playlist
}

func NewPlaylistCollection(customPlaylists map[string]Playlist) *PlaylistCollection {
	cp := make(map[string]Playlist)
	for n, p := range customPlaylists {
		cp[n] = p
	}

	return &PlaylistCollection{
		playlists:       make(map[string][]Playlist),
		customPlaylists: cp,
	}
}

func (pc *PlaylistCollection) getServices(name string, deps map[string]bool) ([]string, error) {
	playlist, err := pc.Get(name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get playlist %s", name)
	}

	if playlist.Extends == "" {
		return playlist.Services, nil
	}

	// Resolve parent playlist defined in extends
	deps[name] = true
	if deps[playlist.Extends] {
		return nil, errors.Errorf("Circular dependency of services, %s and %s", playlist.Extends, name)
	}

	parentServices, err := pc.getServices(playlist.Extends, deps)
	return append(parentServices, playlist.Services...), err
}

func (pc *PlaylistCollection) Get(name string) (Playlist, error) {
	// Check custom playlists first
	if p, ok := pc.customPlaylists[name]; ok {
		return p, nil
	}

	recipeName, playlistName, err := util.SplitNameParts(name)
	if err != nil {
		return Playlist{}, errors.Wrapf(err, "invalid playlist name %s", name)
	}

	bucket, ok := pc.playlists[playlistName]
	if !ok {
		return Playlist{}, errors.Errorf("No such playlist %s", playlistName)
	}

	// Handle shorthand syntax
	if recipeName == "" {
		if len(bucket) > 1 {
			return Playlist{}, errors.Errorf("Muliple playlists named %s found", playlistName)
		}

		return bucket[0], nil
	}

	// Handle long syntax
	for _, p := range bucket {
		if p.RecipeName == recipeName {
			return p, nil
		}
	}

	return Playlist{}, errors.Errorf("No such playlist %s", name)
}

func (pc *PlaylistCollection) Set(name string, value Playlist) error {
	// Check custom playlists first
	if _, ok := pc.customPlaylists[name]; ok {
		pc.customPlaylists[name] = value
		return nil
	}

	recipeName, playlistName, err := util.SplitNameParts(name)
	if err != nil {
		return errors.Wrapf(err, "invalid playlist name %s", name)
	}

	bucket, ok := pc.playlists[playlistName]
	if !ok {
		pc.playlists[playlistName] = []Playlist{value}
		return nil
	}

	// Check for existing playlist to update
	for i, p := range bucket {
		if p.RecipeName == recipeName {
			pc.playlists[playlistName][i] = value
			return nil
		}
	}

	// No matching playlist found, add a new one
	pc.playlists[playlistName] = append(bucket, value)
	return nil
}

func (pc *PlaylistCollection) GetServices(name string) ([]string, error) {
	serviceNames, err := pc.getServices(name, make(map[string]bool))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get services of plylist %s", name)
	}

	return util.UniqueStrings(serviceNames), nil
}

func (pc *PlaylistCollection) Names() []string {
	names := make([]string, 0)
	for _, bucket := range pc.playlists {
		for _, p := range bucket {
			names = append(names, p.FullName())
		}
	}

	return names
}

func (pc *PlaylistCollection) CustomNames() []string {
	names := make([]string, 0)
	for n := range pc.customPlaylists {
		names = append(names, n)
	}

	return names
}
