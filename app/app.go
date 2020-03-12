package app

import (
	"fmt"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

type Storage struct {
	Provider string `yaml:"provider"`
	Bucket   string `yaml:"bucket"`
}

type App struct {
	BundleID string            `yaml:"bundleID"`
	Branch   string            `yaml:"branch"`
	GitRepo  string            `yaml:"repo"`
	EnvVars  map[string]string `yaml:"envVars"`
	Storage  Storage           `yaml:"storage"`
	// Not part of yaml, set at runtime
	Name         string `yaml:"-"`
	RegistryName string `yaml:"-"`
}

func (a App) FullName() string {
	return fmt.Sprintf("%s/%s", a.RegistryName, a.Name)
}

type AppCollection struct {
	am map[string][]App
}

func NewAppCollection(apps []App) (*AppCollection, error) {
	ac := &AppCollection{
		am: make(map[string][]App),
	}

	for _, a := range apps {
		err := ac.set(a)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add app %s to AppCollection", a.FullName())
		}
	}

	return ac, nil
}

func (ac *AppCollection) Get(name string) (App, error) {
	registryName, appName, err := util.SplitNameParts(name)
	if err != nil {
		return App{}, errors.Wrapf(err, "invalid app name %s", name)
	}

	bucket, ok := ac.am[appName]
	if !ok {
		return App{}, errors.Errorf("No such app %s", appName)
	}

	// Handle short syntax
	if registryName == "" {
		if len(bucket) > 1 {
			return App{}, errors.Errorf("Multiple apps named %s found", appName)
		}

		return bucket[0], nil
	}

	// Handle long syntax
	for _, a := range bucket {
		if a.RegistryName == registryName {
			return a, nil
		}
	}

	return App{}, errors.Errorf("No such app %s", name)
}

func (ac *AppCollection) set(value App) error {
	if value.Name == "" || value.RegistryName == "" {
		return errors.Errorf("Name and RegistryName fields must not be empty to set App")
	}

	fullName := value.FullName()
	registryName, appName, err := util.SplitNameParts(fullName)
	if err != nil {
		return errors.Wrapf(err, "invalid app name %s", fullName)
	}

	bucket, ok := ac.am[appName]
	if !ok {
		ac.am[appName] = []App{value}
		return nil
	}

	// Check for existing app to update
	for i, a := range bucket {
		if a.RegistryName == registryName {
			ac.am[appName][i] = value
			return nil
		}
	}

	// No matching app found, add a new one
	ac.am[appName] = append(bucket, value)
	return nil
}
