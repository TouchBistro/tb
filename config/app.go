package config

import (
	"path/filepath"

	"github.com/pkg/errors"
)

type IOSApp struct {
	BundleID   string            `yaml:"bundleID"`
	Branch     string            `yaml:"branch"`
	Repo       string            `yaml:"repo"`
	EnvVars    map[string]string `yaml:"envVars"`
	RecipeName string            `yaml:"-"`
}

type RecipeAppConfig struct {
	// TODO what is a good way to make storage generic & have strategies
	// i.e. s3 strategy for us
	Global struct {
		Storage struct {
			Type string `yaml:"type"`
			Name string `yaml:"name"`
		} `yaml:"storage"`
	} `yaml:"global"`
	IOSApps map[string]IOSApp `yaml:"ios"`
}

type AppConfig struct {
	IOSApps map[string][]IOSApp
}

// TODO legacy - remove
const Bucket = "tb-ios-builds"

func IOSBuildPath() string {
	return filepath.Join(tbRoot, "ios")
}

func GetIOSApp(name string) (string, IOSApp, error) {
	recipeName, appName, err := recipeNameParts(name)
	if err != nil {
		return "", IOSApp{}, errors.Wrapf(err, "invalid iOS app name %s", name)
	}

	list, ok := appConfig.IOSApps[appName]
	if !ok {
		return "", IOSApp{}, errors.Errorf("No such iOS app %s", appName)
	}

	if recipeName == "" {
		if len(list) > 1 {
			return "", IOSApp{}, errors.Errorf("Multiple iOS apps named %s found. Please specify the recipe the item belongs to.", appName)
		}

		a := list[0]
		return joinNameParts(a.RecipeName, appName), a, nil
	}

	for _, app := range list {
		if app.RecipeName == recipeName {
			return name, app, nil
		}
	}

	return "", IOSApp{}, errors.Errorf("No such iOS app %s", name)
}
