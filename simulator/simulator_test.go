package simulator

import (
	"io/ioutil"
	"testing"

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
