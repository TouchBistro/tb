package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type OSMap = map[string]DeviceMap
type DeviceMap = map[string]string

type DeviceSet struct {
	DefaultDevices map[string]interface{} `json:"DefaultDevices"`
}

var osMap OSMap

func getSimulators() (OSMap, error) {
	path := fmt.Sprintf("%s/Library/Developer/CoreSimulator/Devices/device_set.plist", os.Getenv("HOME"))
	cmd := exec.Command("plutil", "-convert", "json", "-o", "-", path)
	cmdOut := &bytes.Buffer{}
	cmd.Stdout = cmdOut

	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read set of iOS simulators")
	}

	var deviceSet DeviceSet
	err = json.Unmarshal(cmdOut.Bytes(), &deviceSet)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse simulator set JSON")
	}

	// Delete pesky `version` key
	delete(deviceSet.DefaultDevices, "version")
	osMap := make(OSMap, len(deviceSet.DefaultDevices))
	const osPrefix = "com.apple.CoreSimulator.SimRuntime."
	const devicePrefix = "com.apple.CoreSimulator.SimDeviceType."

	for osName, devices := range deviceSet.DefaultDevices {
		// Only care about iOS simulators
		if !strings.HasPrefix(osName, osPrefix+"iOS") {
			continue
		}

		deviceTypes, ok := devices.(map[string]interface{})
		if !ok {
			continue
		}

		deviceMap := make(DeviceMap, len(deviceTypes))

		for deviceName, value := range deviceTypes {
			deviceUUID, ok := value.(string)
			if !ok {
				continue
			}

			name := strings.TrimPrefix(deviceName, devicePrefix)
			deviceMap[name] = deviceUUID
		}

		name := strings.TrimPrefix(osName, osPrefix)
		osMap[name] = deviceMap
	}

	return osMap, nil
}

func dashEncode(str string) string {
	// Replace all spaces, brackets, and periods with dashes
	regex := regexp.MustCompile(`(\.|\(|\)|\s)`)
	return regex.ReplaceAllString(str, "-")
}

func InitIOS() error {
	var err error
	osMap, err = getSimulators()
	if err != nil {
		return errors.Wrap(err, "Failed to get available simulators")
	}

	return nil
}

func GetDeviceUUID(osVersion, name string) (string, error) {
	osKey := dashEncode(osVersion)
	nameKey := dashEncode(name)

	deviceMap, ok := osMap[osKey]
	if !ok {
		return "", errors.New(fmt.Sprintf("Unknown OS: %s", osVersion))
	}

	deviceUUID, ok := deviceMap[nameKey]
	if !ok {
		return "", errors.New(fmt.Sprintf("Unknown device: %s", name))
	}

	return deviceUUID, nil
}
