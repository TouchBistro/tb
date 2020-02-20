package config

import "path/filepath"

type IOSApp struct {
	BundleID   string            `yaml:"bundleID"`
	Branch     string            `yaml:"branch"`
	Repo       string            `yaml:"repo"`
	EnvVars    map[string]string `yaml:"envVars"`
	RecipeName string            `yaml:"-"`
}

type RecipeAppConfig struct {
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

func Apps() map[string]IOSApp {
	return appConfig.IOSApps
}

func IOSBuildPath() string {
	return filepath.Join(tbRoot, "ios")
}
