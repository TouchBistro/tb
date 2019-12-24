package config

import (
	"fmt"
)

type IOSApp struct {
	BundleID     string
	Branch       string
	Repo         string
	Organisation string
	EnvVars      map[string]string
}

const Bucket = "tb-ios-builds"

var apps = map[string]IOSApp{
	"TouchBistro": {
		BundleID:     "com.touchbistro.TouchBistro",
		Branch:       "develop",
		Organisation: "TouchBistro",
		Repo:         "tb-pos",
		EnvVars: map[string]string{
			"debug.autoAcceptTOS": "true",
		},
	},
}

func Apps() map[string]IOSApp {
	return apps
}

func IOSBuildPath() string {
	return fmt.Sprintf("%s/%s", tbRoot, "ios")
}
