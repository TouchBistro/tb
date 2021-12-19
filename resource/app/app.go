package app

import (
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/TouchBistro/tb/resource"
)

// Type specifies the type of app.
type Type int

const (
	TypeiOS Type = iota
	TypeDesktop
)

// App specifies the configuration for an app that can be run by tb.
type App struct {
	// These fields are iOS specific

	BundleID string `yaml:"bundleID"`
	RunsOn   string `yaml:"runsOn"`

	// General fields

	Branch  string            `yaml:"branch"`
	GitRepo string            `yaml:"repo"`
	EnvVars map[string]string `yaml:"envVars"`
	Storage Storage           `yaml:"storage"`
	// Not part of yaml, set at runtime
	Name         string `yaml:"-"`
	RegistryName string `yaml:"-"`
}

type Storage struct {
	Provider string `yaml:"provider"`
	Bucket   string `yaml:"bucket"`
}

func (App) Type() resource.Type {
	return resource.TypeApp
}

// FullName returns the app name prefixed with the registry name,
// i.e. '<registry>/<app>'.
func (a App) FullName() string {
	return resource.FullName(a.RegistryName, a.Name)
}

// DeviceType returns the device type that the app runs on if it is an iOS app.
func (a App) DeviceType() simulator.DeviceType {
	runsOn := strings.ToLower(a.RunsOn)
	switch runsOn {
	case "ipad":
		return simulator.DeviceTypeiPad
	case "iphone":
		return simulator.DeviceTypeiPhone
	default:
		return simulator.DeviceTypeUnspecified
	}
}

// Validate validates a. If a is invalid a resource.ValidationError will be returned.
// t is used to determine how a should be validated.
func Validate(a App, t Type) error {
	// No validations needed for desktop currently
	if t != TypeiOS {
		return nil
	}

	var msgs []string
	switch strings.ToLower(a.RunsOn) {
	case "", "all", "ipad", "iphone":
	default:
		msgs = append(msgs, "'runsOn' value is invalid, must be 'all', 'ipad', or 'iphone'")
	}
	if msgs == nil {
		return nil
	}
	return &resource.ValidationError{Resource: a, Messages: msgs}
}

// Collection stores a collection of apps.
// Collection allows for efficiently looking up an app by its
// short name (i.e. the name of the app without the registry).
//
// A zero value Collection is a valid collection ready for use.
type Collection struct {
	collection resource.Collection
}

// Get retrieves the app with the given name from the Collection.
// name can either be the full name or the short name of the app.
//
// If no app is found, resource.ErrNotFound is returned. If name is a short name
// and multiple apps are found, resource.ErrMultipleResources is returned.
func (c *Collection) Get(name string) (App, error) {
	r, err := c.collection.Get(name)
	if err != nil {
		return App{}, errors.Wrap(err, errors.Meta{Op: "app.Collection.Get"})
	}
	return r.(App), nil
}

// Set adds or replaces the app in the Collection.
// a.FullName() must return a valid full name or an error will be returned.
func (c *Collection) Set(a App) error {
	if err := c.collection.Set(a); err != nil {
		return errors.Wrap(err, errors.Meta{Op: "app.Collection.Set"})
	}
	return nil
}

// Names returns a list of the full names of all apps in the collection.
func (c *Collection) Names() []string {
	names := make([]string, 0, c.collection.Len())
	it := c.collection.Iter()
	for it.Next() {
		names = append(names, it.Value().FullName())
	}
	return names
}
