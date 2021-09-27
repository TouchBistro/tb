package simulator

import (
	"io/ioutil"
	"testing"

	"github.com/TouchBistro/tb/resource/app"
	"github.com/stretchr/testify/assert"
)

func TestGetDeviceUDID(t *testing.T) {
	assert := assert.New(t)

	deviceData, err := ioutil.ReadFile("testdata/devices.json")
	if err != nil {
		assert.FailNow("Failed to read devices json file", err)
	}

	deviceList, err := ParseSimulators(deviceData)
	if err != nil {
		assert.FailNow("Failed to parse simulators", err)
	}

	deviceUDID, err := deviceList.GetDeviceUDID("iOS 13.5", "iPad Pro (9.7-inch)")

	assert.NoError(err)
	assert.Equal("A01ADC2E-6AAE-401C-A5B4-5CC5B165E8A1", deviceUDID)
}

func TestGetLatestIOSVersion(t *testing.T) {
	assert := assert.New(t)

	deviceData, err := ioutil.ReadFile("testdata/devices.json")
	if err != nil {
		assert.FailNow("Failed to read devices json file", err)
	}

	deviceList, err := ParseSimulators(deviceData)
	if err != nil {
		assert.FailNow("Failed to parse simulators", err)
	}

	version, err := deviceList.GetLatestIOSVersion()

	assert.NoError(err)
	assert.Equal("13.5", version)
}

func TestGetDefaultDevice(t *testing.T) {
	tests := []struct {
		name           string
		osVersion      string
		deviceType     app.DeviceType
		expectedDevice string
	}{
		{
			"Any device",
			"iOS 13.5",
			app.DeviceTypeAll,
			"iPhone 8",
		},
		{
			"Only iPad",
			"iOS 13.5",
			app.DeviceTypeiPad,
			"iPad Pro (9.7-inch)",
		},
		{
			"Only iPhone",
			"iOS 13.5",
			app.DeviceTypeiPhone,
			"iPhone 8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			deviceData, err := ioutil.ReadFile("testdata/devices.json")
			if err != nil {
				assert.FailNow("Failed to read devices json file", err)
			}

			deviceList, err := ParseSimulators(deviceData)
			if err != nil {
				assert.FailNow("Failed to parse simulators", err)
			}

			device, err := deviceList.GetDefaultDevice(tt.osVersion, tt.deviceType)

			assert.NoError(err)
			assert.Equal(tt.expectedDevice, device)
		})
	}
}

func TestIsValidDevice(t *testing.T) {
	tests := []struct {
		name            string
		deviceName      string
		deviceType      app.DeviceType
		expectedIsValid bool
	}{
		{
			"Any device type valid",
			"iPhone 11 Pro Max",
			app.DeviceTypeAll,
			true,
		},
		{
			"Valid iPad",
			"iPad Pro (11-inch) (2nd generation)",
			app.DeviceTypeiPad,
			true,
		},
		{
			"Invalid iPad",
			"iPhone 11 Pro Max",
			app.DeviceTypeiPad,
			false,
		},
		{
			"Valid iPhone",
			"iPhone 11 Pro Max",
			app.DeviceTypeiPhone,
			true,
		},
		{
			"Invalid iPhone",
			"iPad Pro (11-inch) (2nd generation)",
			app.DeviceTypeiPhone,
			false,
		},
		{
			"No type valid",
			"iPad Pro (11-inch) (2nd generation)",
			app.DeviceTypeUnknown,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := IsValidDevice(tt.deviceName, tt.deviceType)

			assert.Equal(t, tt.expectedIsValid, isValid)
		})
	}
}
