// Package app contains functionality for working with App resources.
// An app is an iOS or desktop application that can be run by tb.
package app

import (
	"strings"

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
	// Make RunsOn case insensitive so users can do things like "ipad" or "iPad".
	switch strings.ToLower(a.RunsOn) {
	case "ipad":
		return simulator.DeviceTypeiPad
	case "iphone":
		return simulator.DeviceTypeiPhone
	default:
		// Don't error here and just assume unspecified.
		// Validate handle's verifying that it is a valid value.
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
