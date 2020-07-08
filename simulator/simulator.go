package simulator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Device struct {
	State       string `json:"state"`
	IsAvailable bool   `json:"isAvailable"`
	Name        string `json:"name"`
	UDID        string `json:"udid"`
}

type DeviceList struct {
	// A map of runtimes (OS versions) to devices (simulators)
	deviceMap map[string][]Device
}

func ParseSimulators(deviceData []byte) (DeviceList, error) {
	var rawDeviceList struct {
		Devices map[string][]Device `json:"devices"`
	}
	err := json.Unmarshal(deviceData, &rawDeviceList)
	if err != nil {
		return DeviceList{}, errors.Wrap(err, "Failed to parse device list JSON")
	}

	const osPrefix = "com.apple.CoreSimulator.SimRuntime."
	deviceList := DeviceList{
		deviceMap: make(map[string][]Device),
	}

	for osName, devices := range rawDeviceList.Devices {
		if len(devices) == 0 {
			continue
		}

		// Only care about iOS simulators, remove rest
		if !strings.HasPrefix(osName, osPrefix+"iOS") {
			continue
		}

		name := strings.TrimPrefix(osName, osPrefix)
		deviceList.deviceMap[name] = devices
	}

	return deviceList, nil
}

func (dl DeviceList) GetDeviceUDID(osVersion, deviceName string) (string, error) {
	// Replace all spaces, brackets, and periods with dashes
	// Ex: `iOS 13.5` will become `iOS-13-5`
	regex := regexp.MustCompile(`(\.|\(|\)|\s)`)
	osKey := regex.ReplaceAllString(osVersion, "-")

	devices, ok := dl.deviceMap[osKey]
	if !ok {
		return "", errors.Errorf("Unknown OS: %s", osVersion)
	}

	foundDevices := make([]Device, 0)
	for _, device := range devices {
		if device.Name == deviceName {
			foundDevices = append(foundDevices, device)
		}
	}

	numDevices := len(foundDevices)
	if numDevices == 0 {
		return "", errors.Errorf("No device with name %s and OS version %s", deviceName, osVersion)
	} else if numDevices > 1 {
		return "", errors.Errorf("More than 1 device with name %s and OS version %s", deviceName, osVersion)
	}

	return foundDevices[0].UDID, nil
}

func (dl DeviceList) GetLatestIOSVersion() (string, error) {
	type version struct {
		major int
		minor int
	}

	osVersions := make([]version, len(dl.deviceMap))
	for osVersion := range dl.deviceMap {
		// Format is `iOS-major-minor`
		parts := strings.Split(osVersion, "-")
		if len(parts) != 3 {
			return "", errors.Errorf("Invalid iOS version string %s", osVersion)
		}

		major, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", errors.Wrapf(err, "Failed to parse major version as an int %s", parts[1])
		}

		minor, err := strconv.Atoi(parts[2])
		if err != nil {
			return "", errors.Wrapf(err, "Failed to parse minor version as an int %s", parts[2])
		}

		osVersions = append(osVersions, version{major, minor})
	}

	sort.Slice(osVersions, func(i, j int) bool {
		if osVersions[i].major == osVersions[j].major {
			return osVersions[i].minor < osVersions[j].minor
		}

		return osVersions[i].major < osVersions[j].major
	})

	latestVersion := osVersions[len(osVersions)-1]
	return fmt.Sprintf("%d.%d", latestVersion.major, latestVersion.minor), nil
}
