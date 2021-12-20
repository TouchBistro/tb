package simulator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
)

// ErrOSNotFound indicates a desired OS version was not found.
var ErrOSNotFound errors.String = "os not found"

// ErrDeviceNotFound indicates a specific device was not found.
var ErrDeviceNotFound errors.String = "device not found"

// ErrMultipleDevices indicates multiple devices were found when only one was expected.
var ErrMultipleDevices errors.String = "multiple devices found"

// DeviceType specifies the type of a simulator device.
type DeviceType int

const (
	DeviceTypeUnspecified DeviceType = iota
	DeviceTypeiPad
	DeviceTypeiPhone
)

func (dt DeviceType) String() string {
	return [...]string{"unspecified", "iPad", "iPhone"}[dt]
}

// Device contains the configuration for an iOS device.
type Device struct {
	State       string     `json:"state"`
	IsAvailable bool       `json:"isAvailable"`
	Name        string     `json:"name"`
	UDID        string     `json:"udid"`
	Type        DeviceType `json:"-"`
}

// DeviceList contains all device configurations.
type DeviceList struct {
	// A map of runtimes (OS versions) to devices (simulators)
	deviceMap map[string][]Device
}

// ParseDevices returns a DeviceList by parsing data.
// data is expected to contain JSON encoded device data.
func ParseDevices(data []byte) (DeviceList, error) {
	const op = errors.Op("simulator.ParseDevices")
	var rawDeviceList struct {
		Devices map[string][]Device `json:"devices"`
	}
	if err := json.Unmarshal(data, &rawDeviceList); err != nil {
		return DeviceList{}, errors.Wrap(err, errors.Meta{
			Reason: "failed to parse device list JSON",
			Op:     op,
		})
	}

	// Perform any normalizations and transformations on the simulator data to make it easier to work with.
	const osPrefix = "com.apple.CoreSimulator.SimRuntime."
	deviceList := DeviceList{deviceMap: make(map[string][]Device)}
	for osName, devices := range rawDeviceList.Devices {
		// Only care about iOS simulators, remove rest (i.e watchOS, tvOS).
		if !strings.HasPrefix(osName, osPrefix+"iOS") {
			continue
		}
		parsedDevices := make([]Device, len(devices))
		for i, d := range devices {
			// Determine device type
			switch {
			case strings.Contains(d.Name, "iPad"):
				d.Type = DeviceTypeiPad
			case strings.Contains(d.Name, "iPhone"):
				d.Type = DeviceTypeiPhone
			default:
				// This should never happen. If it does we either have a bug, or apple has
				// added a new device type. In either case it means a fix is needed.
				return DeviceList{}, errors.New(
					errkind.Internal,
					fmt.Sprintf("device has unknown type: %q", d.Name),
					op,
				)
			}
			parsedDevices[i] = d
		}
		name := strings.TrimPrefix(osName, osPrefix)
		deviceList.deviceMap[name] = parsedDevices
	}
	return deviceList, nil
}

// GetDevice returns the device with the given OS version and name.
//
// If the OS version does not exist, ErrOSNotFound will be returned.
// If no device with the given name is found, ErrDeviceNotFound will be returned.
// If multiple devices are found with the same OS version and name, ErrMultipleDevices
// will be returned.
func (dl DeviceList) GetDevice(osVersion, deviceName string) (Device, error) {
	const op = errors.Op("DeviceList.GetDevice")
	devices, err := dl.getDevices(osVersion, op)
	if err != nil {
		return Device{}, err
	}
	var foundDevices []Device
	for _, d := range devices {
		if d.Name == deviceName {
			foundDevices = append(foundDevices, d)
		}
	}
	switch len(foundDevices) {
	case 0:
		return Device{}, errors.Wrap(ErrDeviceNotFound, errors.Meta{
			Kind:   errkind.Invalid,
			Reason: fmt.Sprintf("OS version %q, name %q", osVersion, deviceName),
			Op:     op,
		})
	case 1:
		return foundDevices[0], nil
	default:
		return Device{}, errors.Wrap(ErrMultipleDevices, errors.Meta{
			Kind:   errkind.Invalid,
			Reason: fmt.Sprintf("OS version %q, name %q", osVersion, deviceName),
			Op:     op,
		})
	}
}

// GetDefaultDevice returns the default device to be used with the given OS version and device type.
// If deviceType is DeviceTypeUnspecified the first device found will be returned.
//
// If the OS version does not exist, ErrOSNotFound will be returned.
// If no devices with the given device type are found, ErrDeviceNotFound will be returned.
func (dl DeviceList) GetDefaultDevice(osVersion string, deviceType DeviceType) (Device, error) {
	const op = errors.Op("DeviceList.GetDefaultDevice")
	devices, err := dl.getDevices(osVersion, op)
	if err != nil {
		return Device{}, err
	}
	// Find the first available device the matches the deviceType.
	for _, d := range devices {
		// If unspecified just pick the first one we find.
		if deviceType == DeviceTypeUnspecified {
			return d, nil
		}
		if d.Type == deviceType {
			return d, nil
		}
	}
	return Device{}, errors.Wrap(ErrDeviceNotFound, errors.Meta{
		Kind:   errkind.Invalid,
		Reason: fmt.Sprintf("OS version %q, device type %q", osVersion, deviceType),
		Op:     op,
	})
}

// getDevices returns a list of devices with the given os version.
func (dl DeviceList) getDevices(osVersion string, op errors.Op) ([]Device, error) {
	// Normalize osVersion. This allows calls to specify a friendly name for the OS.
	// Replace all spaces, brackets, and periods with dashes
	// Ex: `iOS 13.5` will become `iOS-13-5`
	regex := regexp.MustCompile(`(\.|\(|\)|\s)`)
	osKey := regex.ReplaceAllString(osVersion, "-")
	// Add iOS prefix if missing
	if !strings.HasPrefix(osKey, "iOS-") {
		osKey = "iOS-" + osKey
	}
	devices, ok := dl.deviceMap[osKey]
	if !ok {
		return nil, errors.Wrap(ErrOSNotFound, errors.Meta{Kind: errkind.Invalid, Op: op})
	}
	return devices, nil
}

// GetLatestiOSVersion returns the latest available iOS version in the form 'MAJOR.MINOR'.
func (dl DeviceList) GetLatestiOSVersion() (string, error) {
	const op = errors.Op("DeviceList.GetLatestiOSVersion")
	type version struct {
		major int
		minor int
	}
	var osVersions []version
	for osVersion := range dl.deviceMap {
		// Format is `iOS-major-minor`
		parts := strings.Split(osVersion, "-")
		if len(parts) != 3 {
			return "", errors.New(
				errkind.Internal,
				fmt.Sprintf("malformed iOS version string %q", osVersion),
				op,
			)
		}
		major, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", errors.New(
				errkind.Internal,
				fmt.Sprintf("failed to parse major version %q", parts[1]),
				op,
			)
		}
		minor, err := strconv.Atoi(parts[2])
		if err != nil {
			return "", errors.New(
				errkind.Internal,
				fmt.Sprintf("failed to parse minor version %q", parts[2]),
				op,
			)
		}
		osVersions = append(osVersions, version{major, minor})
	}
	if len(osVersions) == 0 {
		return "", errors.Wrap(ErrOSNotFound, errors.Meta{
			Kind:   errkind.Invalid,
			Reason: "no OS versions found",
			Op:     op,
		})
	}

	// Sort the os versions in decending order so that the first element is the latest
	sort.Slice(osVersions, func(i, j int) bool {
		if osVersions[i].major == osVersions[j].major {
			return osVersions[i].minor > osVersions[j].minor
		}
		return osVersions[i].major > osVersions[j].major
	})
	latest := osVersions[0]
	return fmt.Sprintf("%d.%d", latest.major, latest.minor), nil
}
