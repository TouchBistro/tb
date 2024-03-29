// Package playlist contains functionality for working with Playlist resources.
// A playlist is a collection of services that can be run together.
package playlist

import (
	"fmt"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/internal/util"
	"github.com/TouchBistro/tb/resource"
)

// Playlist specifies the configuration for a playlist.
// A playlist is a list of services that can be run together.
//
// Playlists can extend another playlist which effectively merges
// the lists of services together.
type Playlist struct {
	Extends  string   `yaml:"extends,omitempty"`
	Services []string `yaml:"services"`
	// Not part of yaml, set at runtime
	Name         string `yaml:"-"`
	RegistryName string `yaml:"-"`
}

func (Playlist) Type() resource.Type {
	return resource.TypePlaylist
}

// FullName returns the playlist name prefixed with the registry name,
// i.e. '<registry>/<playlist>'.
func (p Playlist) FullName() string {
	return resource.FullName(p.RegistryName, p.Name)
}

// Collection stores a collection of playlists.
// Collection allows for efficiently looking up a playlist by its
// short name (i.e. the name of the playlist without the registry).
//
// A zero value Collection is a valid collection ready for use.
type Collection struct {
	collection      resource.Collection[Playlist]
	customPlaylists map[string]Playlist
}

// Get retrieves the playlist with the given name from the Collection.
// name can either be the full name or the short name of the playlist.
//
// Get will first check for a matching custom playlist. That means, if a custom playlist
// has the same name as a playlist in a registry, the custom playlist will take precedence.
// In this case the full name of the registry playlist must be used to retrieve it.
//
// If no playlist is found, resource.ErrNotFound is returned. If name is a short name
// and multiple playlists are found, resource.ErrMultipleResources is returned.
func (c *Collection) Get(name string) (Playlist, error) {
	// Check custom playlists first. Since custom playlists exist outside of a registry
	// there is no such thing as a full name for a custom playlist so there is no other way
	// to resolve ambiguity if the name conflicts with a playlist from a registry.
	// Therefore, custom playlists take precedence so they can always be resolved, and users
	// need to use the full name for the registry playlist.
	if p, ok := c.customPlaylists[name]; ok {
		return p, nil
	}
	return c.collection.Get(name)
}

// Set adds or replaces the playlist in the Collection.
// p.FullName() must return a valid full name or an error will be returned.
func (c *Collection) Set(p Playlist) error {
	return c.collection.Set(p)
}

// SetCustom sets a custom playlist. Custom playlists exist outside of registries,
// and take priority over playlists within registries during lookup.
func (c *Collection) SetCustom(p Playlist) {
	if c.customPlaylists == nil {
		c.customPlaylists = make(map[string]Playlist)
	}
	if p.Name == "" {
		panic("custom playlist has no name")
	}
	c.customPlaylists[p.Name] = p
}

// ServiceNames returns all the service names contained in the playlist with playlistName.
// It will resolve any extends fields and merge the playlists.
//
// ServiceNames automatically removes any duplications as a result of merging playlists.
// For example, if playlist B specifies service S and extends playlist A which also specifies
// service S, the returned slice will only contain service S once not twice.
//
// If a dependency cycle is detected while resolving extends an error will be returned.
func (c *Collection) ServiceNames(playlistName string) ([]string, error) {
	const op = errors.Op("playlist.Collection.ServiceNames")
	serviceNames, err := c.resolveServiceNames(op, playlistName, make(map[string]bool))
	if err != nil {
		return nil, err
	}
	return util.UniqueStrings(serviceNames), nil
}

func (c *Collection) resolveServiceNames(op errors.Op, name string, deps map[string]bool) ([]string, error) {
	p, err := c.Get(name)
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{Op: op})
	}
	if p.Extends == "" {
		return p.Services, nil
	}
	// Check for dependency cycle
	if deps[p.Extends] {
		msg := fmt.Sprintf("circular dependency of services, %s and %s", p.Extends, name)
		return nil, errors.New(errkind.Invalid, msg, op)
	}
	// Resolve parent playlist defined in extends
	deps[name] = true
	parentServices, err := c.resolveServiceNames(op, p.Extends, deps)
	return append(parentServices, p.Services...), err
}

// Name returns a list of the full names of all playlists in the collection.
func (c *Collection) Names() []string {
	names := make([]string, 0, c.collection.Len())
	for it := c.collection.Iter(); it.Next(); {
		names = append(names, it.Value().FullName())
	}
	return names
}

// CustomNames returns a list of names of all the custom playlists in the collection.
func (c *Collection) CustomNames() []string {
	names := make([]string, 0, len(c.customPlaylists))
	for n := range c.customPlaylists {
		names = append(names, n)
	}
	return names
}
