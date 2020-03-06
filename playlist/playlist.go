package playlist

import (
	"fmt"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

type Playlist struct {
	Extends  string   `yaml:"extends,omitempty"`
	Services []string `yaml:"services"`
	// Not part of yaml, set at runtime
	Name         string `yaml:"-"`
	RegistryName string `yaml:"-"`
}

func (p Playlist) FullName() string {
	return fmt.Sprintf("%s/%s", p.RegistryName, p.Name)
}

type PlaylistCollection struct {
	playlistMap     map[string][]Playlist
	customPlaylists map[string]Playlist
}

func NewPlaylistCollection(playlists []Playlist, customPlaylists map[string]Playlist) (*PlaylistCollection, error) {
	// Copy custom playlists
	cp := make(map[string]Playlist)
	for n, p := range customPlaylists {
		cp[n] = p
	}

	pc := &PlaylistCollection{
		playlistMap:     make(map[string][]Playlist),
		customPlaylists: cp,
	}

	for _, p := range playlists {
		err := pc.set(p)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add playlist %s to PlaylistCollection", p.FullName())
		}
	}

	return pc, nil
}

func (pc *PlaylistCollection) resolveServiceNames(name string, deps map[string]bool) ([]string, error) {
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

	parentServices, err := pc.resolveServiceNames(playlist.Extends, deps)
	return append(parentServices, playlist.Services...), err
}

func (pc *PlaylistCollection) Get(name string) (Playlist, error) {
	// Check custom playlists first
	if p, ok := pc.customPlaylists[name]; ok {
		return p, nil
	}

	registryName, playlistName, err := util.SplitNameParts(name)
	if err != nil {
		return Playlist{}, errors.Wrapf(err, "invalid playlist name %s", name)
	}

	bucket, ok := pc.playlistMap[playlistName]
	if !ok {
		return Playlist{}, errors.Errorf("No such playlist %s", playlistName)
	}

	// Handle shorthand syntax
	if registryName == "" {
		if len(bucket) > 1 {
			return Playlist{}, errors.Errorf("Muliple playlists named %s found", playlistName)
		}

		return bucket[0], nil
	}

	// Handle long syntax
	for _, p := range bucket {
		if p.RegistryName == registryName {
			return p, nil
		}
	}

	return Playlist{}, errors.Errorf("No such playlist %s", name)
}

func (pc *PlaylistCollection) set(value Playlist) error {
	if value.Name == "" || value.RegistryName == "" {
		return errors.Errorf("Name and RegistryName fields must not be empty to set Service")
	}

	fullName := value.FullName()
	registryName, playlistName, err := util.SplitNameParts(fullName)
	if err != nil {
		return errors.Wrapf(err, "invalid playlist name %s", fullName)
	}

	bucket, ok := pc.playlistMap[playlistName]
	if !ok {
		pc.playlistMap[playlistName] = []Playlist{value}
		return nil
	}

	// Check for existing playlist to update
	for i, p := range bucket {
		if p.RegistryName == registryName {
			pc.playlistMap[playlistName][i] = value
			return nil
		}
	}

	// No matching playlist found, add a new one
	pc.playlistMap[playlistName] = append(bucket, value)
	return nil
}

func (pc *PlaylistCollection) ServiceNames(playlistName string) ([]string, error) {
	serviceNames, err := pc.resolveServiceNames(playlistName, make(map[string]bool))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get services of plylist %s", playlistName)
	}

	return util.UniqueStrings(serviceNames), nil
}

func (pc *PlaylistCollection) Names() []string {
	names := make([]string, 0)
	for _, bucket := range pc.playlistMap {
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
