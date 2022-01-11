package simulator_test

import (
	"errors"
	"os"
	"testing"

	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/matryer/is"
)

func TestGetDevice(t *testing.T) {
	tests := []struct {
		name       string
		osVersion  string
		deviceName string
		wantDevice simulator.Device
	}{
		{
			name:       "basic usage",
			osVersion:  "iOS 13.5",
			deviceName: "iPad Pro (9.7-inch)",
			wantDevice: simulator.Device{
				State:       "Shutdown",
				IsAvailable: true,
				Name:        "iPad Pro (9.7-inch)",
				UDID:        "A01ADC2E-6AAE-401C-A5B4-5CC5B165E8A1",
				Type:        simulator.DeviceTypeiPad,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			dl := parseDevices(t)
			d, err := dl.GetDevice(tt.osVersion, tt.deviceName)
			is.NoErr(err)
			is.Equal(d, tt.wantDevice)
		})
	}
}

func TestGetDeviceError(t *testing.T) {
	tests := []struct {
		name       string
		osVersion  string
		deviceName string
		wantErr    error
	}{
		{
			name:       "no OS found",
			osVersion:  "iOS 12.5",
			deviceName: "iPad Pro (9.7-inch)",
			wantErr:    simulator.ErrOSNotFound,
		},
		{
			name:       "no device found",
			osVersion:  "iOS 13.5",
			deviceName: "iPad Pro Max (9.7-inch)",
			wantErr:    simulator.ErrDeviceNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			dl := parseDevices(t)
			_, err := dl.GetDevice(tt.osVersion, tt.deviceName)
			is.True(errors.Is(err, tt.wantErr))
		})
	}
}

func TestGetDefaultDevice(t *testing.T) {
	tests := []struct {
		name       string
		osVersion  string
		deviceType simulator.DeviceType
		wantDevice simulator.Device
	}{
		{
			name:       "Any device",
			osVersion:  "iOS 13.5",
			deviceType: simulator.DeviceTypeUnspecified,
			wantDevice: simulator.Device{
				State:       "Shutdown",
				IsAvailable: true,
				Name:        "iPhone 8",
				UDID:        "67D79B13-7D22-4CE4-A15C-0472EE48360D",
				Type:        simulator.DeviceTypeiPhone,
			},
		},
		{
			name:       "Only iPad",
			osVersion:  "iOS 13.5",
			deviceType: simulator.DeviceTypeiPad,
			wantDevice: simulator.Device{
				State:       "Shutdown",
				IsAvailable: true,
				Name:        "iPad Pro (9.7-inch)",
				UDID:        "A01ADC2E-6AAE-401C-A5B4-5CC5B165E8A1",
				Type:        simulator.DeviceTypeiPad,
			},
		},
		{
			name:       "Only iPhone",
			osVersion:  "iOS 13.5",
			deviceType: simulator.DeviceTypeiPhone,
			wantDevice: simulator.Device{
				State:       "Shutdown",
				IsAvailable: true,
				Name:        "iPhone 8",
				UDID:        "67D79B13-7D22-4CE4-A15C-0472EE48360D",
				Type:        simulator.DeviceTypeiPhone,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			dl := parseDevices(t)
			d, err := dl.GetDefaultDevice(tt.osVersion, tt.deviceType)
			is.NoErr(err)
			is.Equal(d, tt.wantDevice)
		})
	}
}

func TestGetDefaultDeviceError(t *testing.T) {
	tests := []struct {
		name       string
		osVersion  string
		deviceType simulator.DeviceType
		wantErr    error
	}{
		{
			name:       "no OS found",
			osVersion:  "iOS 12.5",
			deviceType: simulator.DeviceTypeiPad,
			wantErr:    simulator.ErrOSNotFound,
		},
		{
			name:       "no device found",
			osVersion:  "iOS 13.2",
			deviceType: simulator.DeviceTypeiPad,
			wantErr:    simulator.ErrDeviceNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			dl := parseDevices(t)
			_, err := dl.GetDefaultDevice(tt.osVersion, tt.deviceType)
			is.True(errors.Is(err, tt.wantErr))
		})
	}
}

func TestGetLatestiOSVersion(t *testing.T) {
	is := is.New(t)
	dl := parseDevices(t)
	version, err := dl.GetLatestiOSVersion()
	is.NoErr(err)
	is.Equal(version, "13.5")
}

func parseDevices(t *testing.T) simulator.DeviceList {
	t.Helper()
	data, err := os.ReadFile("testdata/devices.json")
	if err != nil {
		t.Fatalf("failed to read devices.json: %v", err)
	}
	dl, err := simulator.ParseDevices(data)
	if err != nil {
		t.Fatalf("failed to parse devices: %v", err)
	}
	return dl
}
