package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createAppCollection(t *testing.T) *AppCollection {
	ac, err := NewAppCollection([]App{
		App{
			BundleID: "com.touchbistro.TouchBistro",
			Branch:   "develop",
			GitRepo:  "TouchBistro/tb-pos",
			EnvVars: map[string]string{
				"debug.autoAcceptTOS": "true",
			},
			Storage: Storage{
				Provider: "s3",
				Bucket:   "tb-ios-builds",
			},
			Name:         "TouchBistro",
			RegistryName: "TouchBistro/tb-registry",
		},
		App{
			BundleID: "com.touchbistro.UIKitDemo",
			Branch:   "master",
			GitRepo:  "TouchBistro/TBUIKit",
			Storage: Storage{
				Provider: "s3",
				Bucket:   "tb-ios-builds",
			},
			Name:         "UIKitDemo",
			RegistryName: "TouchBistro/tb-registry",
		},
		App{
			BundleID: "com.examplezone.UIKitDemo",
			Branch:   "master",
			GitRepo:  "ExampleZone/UIKitDemo",
			Storage: Storage{
				Provider: "cloud-storage",
				Bucket:   "ios-apps",
			},
			Name:         "UIKitDemo",
			RegistryName: "ExampleZone/tb-registry",
		},
	})
	if err != nil {
		assert.FailNow(t, "Failed to create AppCollection")
	}

	return ac
}

func TestAppCollectionGetFullName(t *testing.T) {
	assert := assert.New(t)
	ac := createAppCollection(t)

	a, err := ac.Get("TouchBistro/tb-registry/UIKitDemo")

	assert.Equal(App{
		BundleID: "com.touchbistro.UIKitDemo",
		Branch:   "master",
		GitRepo:  "TouchBistro/TBUIKit",
		Storage: Storage{
			Provider: "s3",
			Bucket:   "tb-ios-builds",
		},
		Name:         "UIKitDemo",
		RegistryName: "TouchBistro/tb-registry",
	}, a)
	assert.NoError(err)
}

func TestAppCollectionGetShortName(t *testing.T) {
	assert := assert.New(t)
	ac := createAppCollection(t)

	a, err := ac.Get("TouchBistro")

	assert.Equal(App{
		BundleID: "com.touchbistro.TouchBistro",
		Branch:   "develop",
		GitRepo:  "TouchBistro/tb-pos",
		EnvVars: map[string]string{
			"debug.autoAcceptTOS": "true",
		},
		Storage: Storage{
			Provider: "s3",
			Bucket:   "tb-ios-builds",
		},
		Name:         "TouchBistro",
		RegistryName: "TouchBistro/tb-registry",
	}, a)
	assert.NoError(err)
}

func TestAppCollectionGetShortError(t *testing.T) {
	assert := assert.New(t)
	ac := createAppCollection(t)

	a, err := ac.Get("UIKitDemo")

	assert.Zero(a)
	assert.Error(err)
}

func TestAppCollectionGetNonexistent(t *testing.T) {
	assert := assert.New(t)
	ac := createAppCollection(t)

	a, err := ac.Get("TouchBistro/tb-registry/not-an-app")

	assert.Zero(a)
	assert.Error(err)
}
