package simulator

import (
	"strings"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

const (
	execID = "simctl"
	xcrun  = "xcrun"
	simctl = "simctl"
)

func Boot(deviceUDID string) error {
	err := util.Exec(execID, xcrun, simctl, "bootstatus", deviceUDID, "-b")
	if err != nil {
		return errors.Wrapf(err, "Failed to boot simulator %s", deviceUDID)
	}

	return nil
}

func Open() error {
	err := util.Exec("open-sim", "open", "-a", "simulator")
	if err != nil {
		return errors.Wrap(err, "Failed to open simulator application")
	}

	return nil
}

func InstallApp(deviceUDID, appPath string) error {
	err := util.Exec(execID, xcrun, simctl, "install", deviceUDID, appPath)
	if err != nil {
		return errors.Wrapf(err, "Failed to install app on simulator %s", deviceUDID)
	}

	return nil
}

func LaunchApp(deviceUDID, appBundleID string) error {
	err := util.Exec(execID, xcrun, simctl, "launch", deviceUDID, appBundleID)
	if err != nil {
		return errors.Wrapf(err, "Failed to launch app %s on simulator %s", appBundleID, deviceUDID)
	}

	return nil
}

func GetAppDataPath(deviceUDID, appBundleID string) (string, error) {
	buf, err := util.ExecResult(execID, xcrun, simctl, "get_app_container", deviceUDID, appBundleID, "data")
	if err != nil {
		return "", errors.Wrap(err, "Failed to get path to app data directory")
	}

	return strings.TrimSpace(buf.String()), nil
}

func ListDevices() ([]byte, error) {
	// List available simulators as json
	buf, err := util.ExecResult(execID, xcrun, simctl, "list", "devices", "-j", "available")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get list of devices")
	}

	return buf.Bytes(), nil
}
