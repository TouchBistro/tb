package config

import "path/filepath"

type IOSApp struct {
	BundleID string `yaml:"bundleID"`
	Branch   string `yaml:"branch"`
	Repo     string `yaml:"repo"`
	EnvVars  map[string]string
}

const Bucket = "tb-ios-builds"

var apps = map[string]IOSApp{
	"TouchBistro": {
		BundleID: "com.touchbistro.TouchBistro",
		Branch:   "develop",
		Repo:     "TouchBistro/tb-pos",
		EnvVars: map[string]string{
			"debug.autoAcceptTOS": "true",
		},
	},
}

func Apps() map[string]IOSApp {
	return apps
}

func IOSBuildPath() string {
	return filepath.Join(tbRoot, "ios")
}
