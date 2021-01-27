package simulator

import (
	"bytes"
	"io"
	"strings"

	"github.com/TouchBistro/goutils/command"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	xcrun  = "xcrun"
	simctl = "simctl"
)

func Boot(deviceUDID string) error {
	err := execSimctl("simctl-boot", nil, "bootstatus", deviceUDID, "-b")
	if err != nil {
		return errors.Wrapf(err, "Failed to boot simulator %s", deviceUDID)
	}
	return nil
}

func Open() error {
	w := log.WithField("id", "open-sim").WriterLevel(log.DebugLevel)
	defer w.Close()
	cmd := command.New(command.WithStdout(w), command.WithStderr(w))
	err := cmd.Exec("open", "-a", "simulator")
	if err != nil {
		return errors.Wrap(err, "Failed to open simulator application")
	}
	return nil
}

func InstallApp(deviceUDID, appPath string) error {
	err := execSimctl("simctl-install", nil, "install", deviceUDID, appPath)
	if err != nil {
		return errors.Wrapf(err, "Failed to install app on simulator %s", deviceUDID)
	}
	return nil
}

func LaunchApp(deviceUDID, appBundleID string) error {
	err := execSimctl("simctl-launch", nil, "launch", deviceUDID, appBundleID)
	if err != nil {
		return errors.Wrapf(err, "Failed to launch app %s on simulator %s", appBundleID, deviceUDID)
	}
	return nil
}

func GetAppDataPath(deviceUDID, appBundleID string) (string, error) {
	buf := &bytes.Buffer{}
	err := execSimctl("simctl-data-path", buf, "get_app_container", deviceUDID, appBundleID, "data")
	if err != nil {
		return "", errors.Wrap(err, "Failed to get path to app data directory")
	}
	return strings.TrimSpace(buf.String()), nil
}

func ListDevices() ([]byte, error) {
	// List available simulators as json
	buf := &bytes.Buffer{}
	err := execSimctl("simctl-list", buf, "list", "devices", "-j", "available")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get list of devices")
	}
	return buf.Bytes(), nil
}

func execSimctl(id string, stdout io.Writer, args ...string) error {
	w := log.WithField("id", id).WriterLevel(log.DebugLevel)
	defer w.Close()
	if stdout == nil {
		stdout = w
	}

	cmd := command.New(command.WithStdout(stdout), command.WithStderr(w))
	args = append([]string{simctl}, args...)
	return cmd.Exec(xcrun, args...)
}
