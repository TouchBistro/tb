package simulator

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/TouchBistro/goutils/command"
	"github.com/pkg/errors"
)

const (
	xcrun  = "xcrun"
	simctl = "simctl"
)

func Boot(deviceUDID string) error {
	err := command.Exec(xcrun, []string{simctl, "bootstatus", deviceUDID, "-b"}, "simctl-boot")
	if err != nil {
		return errors.Wrapf(err, "Failed to boot simulator %s", deviceUDID)
	}

	return nil
}

func Open() error {
	err := command.Exec("open", []string{"-a", "simulator"}, "open-sim")
	if err != nil {
		return errors.Wrap(err, "Failed to open simulator application")
	}

	return nil
}

func InstallApp(deviceUDID, appPath string) error {
	err := command.Exec(xcrun, []string{simctl, "install", deviceUDID, appPath}, "simctl-install")
	if err != nil {
		return errors.Wrapf(err, "Failed to install app on simulator %s", deviceUDID)
	}

	return nil
}

func LaunchApp(deviceUDID, appBundleID string) error {
	err := command.Exec(xcrun, []string{simctl, "launch", deviceUDID, appBundleID}, "simctl-launch")
	if err != nil {
		return errors.Wrapf(err, "Failed to launch app %s on simulator %s", appBundleID, deviceUDID)
	}

	return nil
}

func GetAppDataPath(deviceUDID, appBundleID string) (string, error) {
	buf := &bytes.Buffer{}
	err := command.Exec(xcrun, []string{simctl, "get_app_container", deviceUDID, appBundleID, "data"}, "simctl-data-path", func(cmd *exec.Cmd) {
		cmd.Stdout = buf
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to get path to app data directory")
	}

	return strings.TrimSpace(buf.String()), nil
}

func ListDevices() ([]byte, error) {
	// List available simulators as json
	buf := &bytes.Buffer{}
	err := command.Exec(xcrun, []string{simctl, "list", "devices", "-j", "available"}, "simctl-list", func(cmd *exec.Cmd) {
		cmd.Stdout = buf
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get list of devices")
	}

	return buf.Bytes(), nil
}
