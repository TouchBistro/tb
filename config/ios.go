package config

type IOSApp struct {
	BundleID string
	Branch   string
}

var apps = map[string]IOSApp{
	"TouchBistro": {
		BundleID: "com.touchbistro.TouchBistro",
		Branch:   "develop",
	},
}

func Apps() map[string]IOSApp {
	return apps
}
