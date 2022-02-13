package simulator_test

import (
	"errors"
	"os"
	"testing"

	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/matryer/is"
)

func TestGetDefaultDevice(t *testing.T) {
	tests := []struct {
		name        string
		osVersion   string
		deviceType  simulator.DeviceType
		wantDevices []simulator.Device
	}{
		{
			name:       "All devices",
			osVersion:  "iOS 13.5",
			deviceType: simulator.DeviceTypeUnspecified,
			wantDevices: []simulator.Device{
				{
					State:       "Shutdown",
					IsAvailable: true,
					Name:        "iPhone 11",
					UDID:        "E199247A-BCBE-431A-B909-16C3A58EFC89",
					LogPath:     "/Users/admin/Library/Logs/CoreSimulator/E199247A-BCBE-431A-B909-16C3A58EFC89",
					Type:        simulator.DeviceTypeiPhone,
				},
				{
					State:       "Shutdown",
					IsAvailable: true,
					Name:        "iPhone 11 Pro",
					UDID:        "6BEB23FC-447E-4557-A1C2-AD7919F89728",
					LogPath:     "/Users/admin/Library/Logs/CoreSimulator/6BEB23FC-447E-4557-A1C2-AD7919F89728",
					Type:        simulator.DeviceTypeiPhone,
				},
				{
					State:       "Shutdown",
					IsAvailable: true,
					Name:        "iPad Pro (9.7-inch)",
					UDID:        "A01ADC2E-6AAE-401C-A5B4-5CC5B165E8A1",
					LogPath:     "/Users/admin/Library/Logs/CoreSimulator/A01ADC2E-6AAE-401C-A5B4-5CC5B165E8A1",
					Type:        simulator.DeviceTypeiPad,
				},
				{
					State:       "Shutdown",
					IsAvailable: true,
					Name:        "iPad (7th generation)",
					UDID:        "F8D16DB4-8639-4BBA-9E84-0ECCB73F43E0",
					LogPath:     "/Users/admin/Library/Logs/CoreSimulator/F8D16DB4-8639-4BBA-9E84-0ECCB73F43E0",
					Type:        simulator.DeviceTypeiPad,
				},
			},
		},
		{
			name:       "Only iPads",
			osVersion:  "iOS 13.5",
			deviceType: simulator.DeviceTypeiPad,
			wantDevices: []simulator.Device{
				{
					State:       "Shutdown",
					IsAvailable: true,
					Name:        "iPad Pro (9.7-inch)",
					UDID:        "A01ADC2E-6AAE-401C-A5B4-5CC5B165E8A1",
					LogPath:     "/Users/admin/Library/Logs/CoreSimulator/A01ADC2E-6AAE-401C-A5B4-5CC5B165E8A1",
					Type:        simulator.DeviceTypeiPad,
				},
				{
					State:       "Shutdown",
					IsAvailable: true,
					Name:        "iPad (7th generation)",
					UDID:        "F8D16DB4-8639-4BBA-9E84-0ECCB73F43E0",
					LogPath:     "/Users/admin/Library/Logs/CoreSimulator/F8D16DB4-8639-4BBA-9E84-0ECCB73F43E0",
					Type:        simulator.DeviceTypeiPad,
				},
			},
		},
		{
			name:       "Only iPhones",
			osVersion:  "iOS 13.5",
			deviceType: simulator.DeviceTypeiPhone,
			wantDevices: []simulator.Device{
				{
					State:       "Shutdown",
					IsAvailable: true,
					Name:        "iPhone 11",
					UDID:        "E199247A-BCBE-431A-B909-16C3A58EFC89",
					LogPath:     "/Users/admin/Library/Logs/CoreSimulator/E199247A-BCBE-431A-B909-16C3A58EFC89",
					Type:        simulator.DeviceTypeiPhone,
				},
				{
					State:       "Shutdown",
					IsAvailable: true,
					Name:        "iPhone 11 Pro",
					UDID:        "6BEB23FC-447E-4557-A1C2-AD7919F89728",
					LogPath:     "/Users/admin/Library/Logs/CoreSimulator/6BEB23FC-447E-4557-A1C2-AD7919F89728",
					Type:        simulator.DeviceTypeiPhone,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			dl := parseDevices(t)
			d, err := dl.ListDevices(tt.osVersion, tt.deviceType)
			is.NoErr(err)
			is.Equal(d, tt.wantDevices)
		})
	}
}

func TestListDevicesError(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			dl := parseDevices(t)
			_, err := dl.ListDevices(tt.osVersion, tt.deviceType)
			is.True(errors.Is(err, tt.wantErr))
		})
	}
}

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
				LogPath:     "/Users/admin/Library/Logs/CoreSimulator/A01ADC2E-6AAE-401C-A5B4-5CC5B165E8A1",
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
