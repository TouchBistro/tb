package simulator

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
)

// ListDevices returns a JSON encoded list of available device configurations.
// It can be passed to ParseDevices.
func ListDevices(ctx context.Context) ([]byte, error) {
	// List available simulators as json
	var buf bytes.Buffer
	err := execSimctl(ctx, "simulator.ListDevices", &buf, "list", "devices", "-j", "available")
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Simulator represents the functionality provided by an iOS simulator.
type Simulator interface {
	Boot(ctx context.Context) error
	Open(ctx context.Context) error
	InstallApp(ctx context.Context, appPath string) error
	LaunchApp(ctx context.Context, appBundleID string) error
	GetAppDataPath(ctx context.Context, appBundleID string) (string, error)
}

// simulator implements Simulator using the simctl command.
type simulator struct {
	device Device
}

// NewSimulator creates a new Simulator using the given device.
func NewSimulator(device Device) Simulator {
	return &simulator{device: device}
}

func (sim *simulator) Boot(ctx context.Context) error {
	return execSimctl(ctx, "Simulator.Boot", nil, "bootstatus", sim.device.UDID, "-b")
}

func (sim *simulator) Open(ctx context.Context) error {
	const op = errors.Op("Simulator.Open")
	tracker := progress.TrackerFromContext(ctx)
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
	defer w.Close()

	args := []string{"open", "-a", "simulator"}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Simulator,
			Reason: fmt.Sprintf("failed to run %q", strings.Join(args, " ")),
			Op:     op,
		})
	}
	return nil
}

func (sim *simulator) InstallApp(ctx context.Context, appPath string) error {
	return execSimctl(ctx, "Simulator.InstallApp", nil, "install", sim.device.UDID, appPath)
}

func (sim *simulator) LaunchApp(ctx context.Context, appBundleID string) error {
	return execSimctl(ctx, "Simulator.LaunchApp", nil, "launch", sim.device.UDID, appBundleID)
}

func (sim *simulator) GetAppDataPath(ctx context.Context, appBundleID string) (string, error) {
	var buf bytes.Buffer
	err := execSimctl(ctx, "Simulator.GetAppDataPath", &buf, "get_app_container", sim.device.UDID, appBundleID, "data")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func execSimctl(ctx context.Context, op errors.Op, stdout io.Writer, args ...string) error {
	tracker := progress.TrackerFromContext(ctx)
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
	defer w.Close()
	if stdout == nil {
		stdout = w
	}

	finalArgs := append([]string{"xcrun", "simctl"}, args...)
	cmd := exec.CommandContext(ctx, finalArgs[0], finalArgs[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Simulator,
			Reason: fmt.Sprintf("failed to run %q", strings.Join(finalArgs, " ")),
			Op:     op,
		})
	}
	return nil
}
