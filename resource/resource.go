// Package resource provides support for working with resources managed by tb.
//
// The resource package contains general purpose functionality that is common to all resources.
// Resources are services, playlists, and apps. Specific functionality for each of these resource
// is provided by the subpackages. See each subpackage for more details.
package resource

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
)

// ErrInvalidName is returned when a resource name is provided that does not have
// the correct format.
const ErrInvalidName errors.String = "invalid resource name"

// ErrNotFound is returned when a resource is not found in a Collection.
const ErrNotFound errors.String = "resource not found"

// ErrMultipleResources is returned when multiple resources with the same short name
// were found in a Collection.
const ErrMultipleResources errors.String = "multiple resources found with the same name"

// Resource represents a resource managed by tb.
type Resource interface {
	Type() Type
	FullName() string
}

// Type identifies the type of a resource.
type Type int

const (
	TypeService Type = iota
	TypePlaylist
	TypeApp
)

func (t Type) String() string {
	return [...]string{"service", "playlist", "app"}[t]
}

// ValidationError represents a resource having failed validation.
// It contains the resource that failed validation and a list of validation failure messages.
type ValidationError struct {
	Resource Resource
	Messages []string
}

func (ve *ValidationError) Error() string {
	var sb strings.Builder
	sb.WriteString(ve.Resource.Type().String())
	sb.WriteString(": ")
	sb.WriteString(ve.Resource.FullName())
	sb.WriteString(": ")
	for i, msg := range ve.Messages {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(msg)
	}
	return sb.String()
}

// Full form of a resource name is <org>/<repo>/<resource>.
var nameRegex = regexp.MustCompile(`^(?:([\w-]+\/[\w-]+)\/)?([\w-]+)$`)

// ParseName parses name. If name is a full name, the registry name and resource name
// will be returned. If name is a short name, registry name will be empty.
//
// If name is not a valid name, ErrInvalidName is returned.
func ParseName(name string) (registryName, resourceName string, err error) {
	matches := nameRegex.FindStringSubmatch(name)
	if len(matches) == 0 {
		msg := fmt.Sprintf("%s should have format <org>/<repo>/<item>", name)
		err = errors.Wrap(ErrInvalidName, errors.Meta{Kind: errkind.Invalid, Reason: msg, Op: "resource.ParseName"})
		return
	}
	// If no registry name matches[1] will be an empty string so this is safe.
	return matches[1], matches[2], nil
}

// FullName returns the full name by joining registryName and resourceName.
// If registryName is empty, resourceName is returned.
func FullName(registryName, resourceName string) string {
	if registryName == "" {
		return resourceName
	}
	return registryName + "/" + resourceName
}

// TODO(@cszatmary): Change Collection and Iterator to be generic once go 1.18 is released.
// This is a perfect candidate for generics since we are dealing with general purpose
// data structures that are used for each type of resource.
// I already tried this out on a separate branch with the go 1.18 beta and it worked great.

// Collection stores a collection of resources.
// Collection allows for efficiently looking up a resource by its
// short name (i.e. the name of the resource without the registry).
//
// A zero value Collection is a valid collection ready for use.
type Collection struct {
	// resources stores the list of resources.
	// resources are expected to all have the same type.
	resources []Resource
	// nameMap is a map of short names to a list of indices
	// for each matching resource in resources.
	nameMap map[string][]int
}

// Len returns the number of resources stored in the Collection.
func (c *Collection) Len() int {
	return len(c.resources)
}

// Get retrieves the resource with the given name from the Collection.
// name can either be the full name or the short name of the resource.
//
// If no resource is found, ErrNotFound is returned. If name is a short name
// and multiple resources are found, ErrMultipleResources is returned.
func (c *Collection) Get(name string) (Resource, error) {
	const op = errors.Op("resource.Collection.Get")
	registryName, resourceName, err := ParseName(name)
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{Op: op})
	}
	bucket, ok := c.nameMap[resourceName]
	if !ok {
		return nil, errors.Wrap(ErrNotFound, errors.Meta{Kind: errkind.Invalid, Reason: name, Op: op})
	}

	// Handle short name
	if registryName == "" {
		if len(bucket) > 1 {
			return nil, errors.Wrap(ErrMultipleResources, errors.Meta{Kind: errkind.Invalid, Reason: name, Op: op})
		}
		return c.resources[bucket[0]], nil
	}
	for _, ri := range bucket {
		r := c.resources[ri]
		if r.FullName() == name {
			return r, nil
		}
	}
	return nil, errors.Wrap(ErrNotFound, errors.Meta{Kind: errkind.Invalid, Reason: name, Op: op})
}

// Set adds or replaces the resource in the Collection.
// r.FullName() must return a valid full name or an error will be returned.
func (c *Collection) Set(r Resource) error {
	const op = errors.Op("resource.Collection.Set")
	registryName, resourceName, err := ParseName(r.FullName())
	if err != nil {
		return errors.Wrap(err, errors.Meta{Kind: errkind.Internal, Op: op})
	}
	if registryName == "" {
		return errors.New(errkind.Internal, "registry name missing from resource", op)
	}
	if c.nameMap == nil {
		c.nameMap = make(map[string][]int)
	}

	bucket := c.nameMap[resourceName]
	// Check if the resource already exists
	foundIndex := -1
	for _, ri := range bucket {
		rr := c.resources[ri]
		if rr.FullName() == r.FullName() {
			foundIndex = ri
			break
		}
	}

	// If an existing resource was found then easy, just update it.
	if foundIndex != -1 {
		c.resources[foundIndex] = r
		return nil
	}
	// No existing one found, add new one.
	// New resource is always appended to the end so the index is easy.
	c.nameMap[resourceName] = append(bucket, len(c.resources))
	c.resources = append(c.resources, r)
	return nil
}

// Iterator allows for iteration over the resources in a Collection.
// An iterator provides two methods that can be used for iteration, Next and Value.
// Next advances the iterator to the next element and returns a bool indicating if
// it was successful. Value returns the value at the current index.
//
// The iteration order over a Collection is not specified and is not guaranteed to be the same
// from one iteration to the next.
//
// The API can easily be used with a while-style for loop:
//
//  for it := c.Iter(); it.Next(); {
//      r := it.Value()
//      // Do something with r...
//  }
type Iterator struct {
	c *Collection
	i int
}

// Iter creates a new Iterator that can be used to iterate over the resources in a Collection.
func (c *Collection) Iter() *Iterator {
	// Start at -1 since Next is required to be called before accessing the first element
	// so when we increment we get to 0 the first element.
	return &Iterator{c: c, i: -1}
}

// Next advances the iterator to the next element. Every call to Value, even the
// first one, must be preceded by a call to Next.
//
// Next returns a bool indicating whether or not a next element exists meaning
// it is safe to call Value.
func (it *Iterator) Next() bool {
	it.i++
	return it.i < len(it.c.resources)
}

// Value returns the current element in the iterator.
// Value will panic if iteration has finished.
func (it *Iterator) Value() Resource {
	// Do an explicit length check so that we can panic with a custom message to make it clearer.
	// We could just let accessing the underlying slice panic but the message would be confusing
	// and would leak implementation details. Callers shouldn't know or care about the underlying
	// resources slice.
	if it.i >= len(it.c.resources) {
		panic("resource.Iterator: out of bounds access")
	}
	return it.c.resources[it.i]
}
