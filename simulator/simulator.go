package simulator

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type Device struct {
	State       string `json:"state"`
	IsAvailable bool   `json:"isAvailable"`
	Name        string `json:"name"`
	UDID        string `json:"udid"`
}

// A map of runtimes (OS versions) to devices (simulators)
type DeviceMap map[string][]Device

type DeviceList struct {
	Devices DeviceMap `json:"devices"`
}

var deviceMap DeviceMap

func dashEncode(str string) string {
	// Replace all spaces, brackets, and periods with dashes
	regex := regexp.MustCompile(`(\.|\(|\)|\s)`)
	return regex.ReplaceAllString(str, "-")
}

func LoadSimulators() error {
	deviceData, err := ListDevices()
	if err != nil {
		return errors.Wrap(err, "Failed to get device list")
	}

	var deviceList DeviceList
	err = json.Unmarshal(deviceData, &deviceList)
	if err != nil {
		return errors.Wrap(err, "Failed to parse device list JSON")
	}

	const osPrefix = "com.apple.CoreSimulator.SimRuntime."
	deviceMap = make(DeviceMap)

	for osName, device := range deviceList.Devices {
		// Only care about iOS simulators, remove rest
		if !strings.HasPrefix(osName, osPrefix+"iOS") {
			continue
		}

		name := strings.TrimPrefix(osName, osPrefix)
		deviceMap[name] = device
	}

	return nil
}

func GetDeviceUDID(osVersion, name string) (string, error) {
	osKey := dashEncode(osVersion)

	deviceList, ok := deviceMap[osKey]
	if !ok {
		return "", errors.Errorf("Unknown OS: %s", osVersion)
	}

	devices := make([]Device, 0)
	for _, device := range deviceList {
		if device.Name == name {
			devices = append(devices, device)
		}
	}

	numDevices := len(devices)

	if numDevices == 0 {
		return "", errors.Errorf("No device with name %s and OS version %s", name, osVersion)
	} else if numDevices > 1 {
		return "", errors.Errorf("More than 1 device with name %s and OS version %s", name, osVersion)
	}

	return devices[0].UDID, nil
}

func GetLatestIOSVersion() string {
	osVersions := make([]string, len(deviceMap))

	for osVersion := range deviceMap {
		osVersions = append(osVersions, osVersion)
	}

	// Lexical order should be fine since iOS minor versions don't go past 4
	// Also Xcode usually only installs simulators for the latest iOS version by default
	sort.Strings(osVersions)
	latestVersion := osVersions[len(osVersions)-1]

	// Un-normalize the iOS version so it matches user input
	trimmed := strings.TrimLeft(latestVersion, "iOS-")
	return strings.ReplaceAll(trimmed, "-", ".")
}
