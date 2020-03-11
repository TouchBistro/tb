package config

import "path/filepath"

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
	"TBUIKitDemo": {
		BundleID:     "com.touchbistro.TBUIKitDemo",
		Branch:       "master",
		Organisation: "TouchBistro",
		Repo:         "TBUIKit",
	},
}

func Apps() map[string]IOSApp {
	return apps
}

func IOSBuildPath() string {
	return filepath.Join(TBRootPath(), "ios")
}
