package app_test

import (
	"errors"
	"testing"

	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/app"
	"github.com/matryer/is"
)

func TestDeviceType(t *testing.T) {
	tests := []struct {
		name           string
		app            app.App
		wantDeviceType simulator.DeviceType
	}{
		{
			"No device type provided",
			app.App{},
			simulator.DeviceTypeUnspecified,
		},
		{
			"All devices",
			app.App{RunsOn: "all"},
			simulator.DeviceTypeUnspecified,
		},
		{
			"Only iPads",
			app.App{RunsOn: "iPad"},
			simulator.DeviceTypeiPad,
		},
		{
			"Only iPhones",
			app.App{RunsOn: "iPhone"},
			simulator.DeviceTypeiPhone,
		},
		{
			"Unknown device type",
			app.App{RunsOn: "iPod"},
			simulator.DeviceTypeUnspecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			is.Equal(tt.app.DeviceType(), tt.wantDeviceType)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		app        app.App
		appType    app.Type
		wantErr    bool
		wantMsgLen int
	}{
		{
			name: "valid iOS app",
			app: app.App{
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
			appType: app.TypeiOS,
			wantErr: false,
		},
		{
			name: "invalid runsOn for iOS app",
			app: app.App{
				BundleID: "com.touchbistro.UIKitDemo",
				RunsOn:   "iFoot",
				Branch:   "master",
				GitRepo:  "TouchBistro/TBUIKit",
				Storage: app.Storage{
					Provider: "s3",
					Bucket:   "tb-ios-builds",
				},
				Name:         "UIKitDemo",
				RegistryName: "TouchBistro/tb-registry",
			},
			appType:    app.TypeiOS,
			wantErr:    true,
			wantMsgLen: 1,
		},
		{
			name: "valid desktop app",
			app: app.App{
				Branch:  "master",
				GitRepo: "TouchBistro/MenuBoard",
				Storage: app.Storage{
					Provider: "s3",
					Bucket:   "tb-mac-builds",
				},
				Name:         "MenuBoard",
				RegistryName: "TouchBistro/tb-registry",
			},
			appType: app.TypeDesktop,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			err := app.Validate(tt.app, tt.appType)
			if !tt.wantErr {
				is.NoErr(err)
				return
			}
			var validationErr *resource.ValidationError
			is.True(errors.As(err, &validationErr))
			is.Equal(validationErr.Resource, tt.app)
			is.Equal(len(validationErr.Messages), tt.wantMsgLen)
		})
	}
}
