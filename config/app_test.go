package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func fakeAppConfig() AppConfig {
	return AppConfig{
		IOSApps: map[string][]IOSApp{
			"TouchBistro": []IOSApp{
				IOSApp{
					BundleID: "com.touchbistro.TouchBistro",
					Branch:   "develop",
					Repo:     "TouchBistro/tb-pos",
					EnvVars: map[string]string{
						"debug": "true",
					},
					RecipeName: "TouchBistro/tb-recipe-services",
				},
			},
			"ExampleApp": []IOSApp{
				IOSApp{
					BundleID: "com.touchbistro.ExampleApp",
					Branch:   "master",
					Repo:     "TouchBistro/example-app",
					EnvVars: map[string]string{
						"production": "true",
					},
					RecipeName: "TouchBistro/tb-recipe-services",
				},
				IOSApp{
					BundleID: "com.example-zone.ExampleApp",
					Branch:   "master",
					Repo:     "ExampleZone/ExampleApp",
					EnvVars: map[string]string{
						"example": "true",
					},
					RecipeName: "ExampleZone/tb-recipe-examples",
				},
			},
		},
	}
}

func TestIOSBuildPath(t *testing.T) {
	tbRoot = "/User/garlic-zone/.tb"
	assert.Equal(t, "/User/garlic-zone/.tb/ios", IOSBuildPath())
}

func TestGetIOSAppImplicit(t *testing.T) {
	assert := assert.New(t)
	appConfig = fakeAppConfig()

	name, app, err := GetIOSApp("TouchBistro")

	assert.Equal("TouchBistro/tb-recipe-services/TouchBistro", name)
	assert.Equal(IOSApp{
		BundleID: "com.touchbistro.TouchBistro",
		Branch:   "develop",
		Repo:     "TouchBistro/tb-pos",
		EnvVars: map[string]string{
			"debug": "true",
		},
		RecipeName: "TouchBistro/tb-recipe-services",
	}, app)
	assert.NoError(err)
}

func TestGetIOSAppExplicit(t *testing.T) {
	assert := assert.New(t)
	appConfig = fakeAppConfig()

	name, app, err := GetIOSApp("TouchBistro/tb-recipe-services/ExampleApp")

	assert.Equal("TouchBistro/tb-recipe-services/ExampleApp", name)
	assert.Equal(IOSApp{
		BundleID: "com.touchbistro.ExampleApp",
		Branch:   "master",
		Repo:     "TouchBistro/example-app",
		EnvVars: map[string]string{
			"production": "true",
		},
		RecipeName: "TouchBistro/tb-recipe-services",
	}, app)
	assert.NoError(err)
}

func TestGetIOSAppImplicitError(t *testing.T) {
	assert := assert.New(t)
	appConfig = fakeAppConfig()

	name, app, err := GetIOSApp("ExampleApp")

	assert.Empty(name)
	assert.Zero(app)
	assert.Error(err)
}

func TestGetIOSAppNonexistant(t *testing.T) {
	assert := assert.New(t)
	appConfig = fakeAppConfig()

	name, app, err := GetIOSApp("not-an-app")

	assert.Empty(name)
	assert.Zero(app)
	assert.Error(err)
}

func TestGetIOSAppInvalidName(t *testing.T) {
	assert := assert.New(t)
	appConfig = fakeAppConfig()

	name, app, err := GetIOSApp("malformed/not-an-app")

	assert.Empty(name)
	assert.Zero(app)
	assert.Error(err)
}

func TestGetIOSAppNonexistantRecipe(t *testing.T) {
	assert := assert.New(t)
	appConfig = fakeAppConfig()

	name, app, err := GetIOSApp("DeadZone/tb-recipe-apps/ExampleApp")

	assert.Empty(name)
	assert.Zero(app)
	assert.Error(err)
}
