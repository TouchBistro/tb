package simulator

import (
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

const (
	execID = "simctl"
	xcrun  = "xcrun"
	simctl = "simctl"
)

func Boot(deviceUUID string) error {
	err := util.Exec(execID, xcrun, simctl, "bootstatus", deviceUUID, "-b")
	if err != nil {
		return errors.Wrapf(err, "Failed to boot simulator %s", deviceUUID)
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

func InstallApp(deviceUUID, appPath string) error {
	err := util.Exec(execID, xcrun, simctl, "install", deviceUUID, appPath)
	if err != nil {
		return errors.Wrapf(err, "Failed to install app on simulator %s", deviceUUID)
	}

	return nil
}

func LaunchApp(deviceUUID, appBundleID string) error {
	err := util.Exec(execID, xcrun, simctl, "launch", deviceUUID, appBundleID)
	if err != nil {
		return errors.Wrapf(err, "Failed to launch app %s on simulator %s", appBundleID, deviceUUID)
	}

	return nil
}
