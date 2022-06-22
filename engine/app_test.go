package engine_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TouchBistro/tb/engine"
	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/app"
	"github.com/matryer/is"
)

func TestAppiOSListDevices(t *testing.T) {
	tests := []struct {
		name           string
		opts           engine.AppiOSListDevicesOptions
		wantDevices    []string
		wantiOSVersion string
	}{
		{
			name:           "no options",
			wantDevices:    []string{"iPhone 11", "iPhone 11 Pro", "iPad Pro (9.7-inch)", "iPad (7th generation)"},
			wantiOSVersion: "13.5",
		},
		{
			name: "app with any device type",
			opts: engine.AppiOSListDevicesOptions{
				AppName: "UIKitDemo",
			},
			wantDevices:    []string{"iPhone 11", "iPhone 11 Pro", "iPad Pro (9.7-inch)", "iPad (7th generation)"},
			wantiOSVersion: "13.5",
		},
		{
			name: "app with ipad only",
			opts: engine.AppiOSListDevicesOptions{
				AppName:    "TouchBistro",
				IOSVersion: "13.5",
			},
			wantDevices:    []string{"iPad Pro (9.7-inch)", "iPad (7th generation)"},
			wantiOSVersion: "13.5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := newAppCollection(t, []app.App{
				{
					BundleID: "com.touchbistro.TouchBistro",
					RunsOn:   "iPad",
					Branch:   "master",
					GitRepo:  "TouchBistro/tb-pos",
					EnvVars: map[string]string{
						"debug.autoAcceptTOS": "true",
					},
					Storage: app.Storage{
						Provider: "s3",
						Bucket:   "tb-ios-builds",
					},
					Name:         "TouchBistro",
					RegistryName: "TouchBistro/tb-registry",
				},
				{
					BundleID: "com.touchbistro.UIKitDemo",
					Branch:   "master",
					GitRepo:  "TouchBistro/TBUIKit",
					Storage: app.Storage{
						Provider: "s3",
						Bucket:   "tb-ios-builds",
					},
					Name:         "UIKitDemo",
					RegistryName: "TouchBistro/tb-registry",
				},
			})
			e := newEngine(t, engine.Options{
				IOSApps:    ac,
				DeviceList: newDeviceList(t),
			})

			ctx := context.Background()
			deviceNames, iosVersion, err := e.AppiOSListDevices(ctx, tt.opts)
			is := is.New(t)
			is.NoErr(err)

			is.Equal(deviceNames, tt.wantDevices)
			is.Equal(iosVersion, tt.wantiOSVersion)
		})
	}
}

func newAppCollection(t *testing.T, apps []app.App) *resource.Collection[app.App] {
	t.Helper()
	var ac resource.Collection[app.App]
	for _, a := range apps {
		if err := ac.Set(a); err != nil {
			t.Fatalf("failed to add app %s to collection: %v", a.FullName(), err)
		}
	}
	return &ac
}

func newDeviceList(t *testing.T) simulator.DeviceList {
	t.Helper()
	// DISCUSS(@cszatmary): Does this make sense to do?
	// I'd prefer to share the same test data but it feels wrong
	// for this package to be reaching into the simulator package.
	data, err := os.ReadFile(filepath.FromSlash("../integrations/simulator/testdata/devices.json"))
	if err != nil {
		t.Fatalf("failed to read devices.json: %v", err)
	}
	dl, err := simulator.ParseDevices(data)
	if err != nil {
		t.Fatalf("failed to parse devices: %v", err)
	}
	return dl
}
