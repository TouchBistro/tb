package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/TouchBistro/tb/internal/util"
	"github.com/TouchBistro/tb/resource/app"
)

// AppiOSRunOptions customizes the behaviour of AppiOSRun.
// All fields are optional.
type AppiOSRunOptions struct {
	// IOSVersion is the iOS version to use.
	IOSVersion string
	// DeviceName is the name of the device to use.
	DeviceName string
	// DataPath is the path to a data directory to inject into the simulator.
	DataPath string
	// Branch is the name of the Git branch associated to the build to run.
	Branch string
}

func (e *Engine) AppiOSRun(ctx context.Context, appName string, opts AppiOSRunOptions) error {
	const op = errors.Op("engine.Engine.AppiOSRun")
	tracker := progress.TrackerFromContext(ctx)
	a, err := e.iosApps.Get(appName)
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "unable to resolve iOS app", Op: op})
	}
	// Override branch if one was provided
	if opts.Branch != "" {
		a.Branch = opts.Branch
	}
	device, err := e.resolveDevice(ctx, opts.IOSVersion, opts.DeviceName, a.DeviceType(), op)
	if err != nil {
		return err
	}
	// Make sure provided device is valid for the given app
	supportedDeviceType := a.DeviceType()
	if supportedDeviceType != simulator.DeviceTypeUnspecified && supportedDeviceType != device.Type {
		return errors.New(
			errkind.Invalid,
			fmt.Sprintf("device %s is not supported by iOS app %s", device.Name, a.FullName()),
			op,
		)
	}
	tracker.Debugf("☑ Found device UDID: %s", device.UDID)

	// Download the app
	var appPath string
	err = progress.Run(ctx, progress.RunOptions{
		Message: fmt.Sprintf("Downloading iOS app %s", a.FullName()),
	}, func(ctx context.Context) (err error) {
		// NOTE(@cszatmary): This is not ideal because this runs in a separate goroutine
		// and it is modifying shared state by assigning to appPath.
		// However, progress.Run provides synchronization so we don't have to worry about
		// a race condition. Once go 1.18 is out, progress.Run should be changed to a generic
		// function so we could return appPath.
		appPath, err = e.downloadApp(ctx, a, app.TypeiOS, op)
		return err
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Reason: fmt.Sprintf("failed to download iOS app %s", a.FullName()),
			Op:     op,
		})
	}

	// All the remaining operations are to run the app on the simulator
	// Run the spinner for the entire process and stop it when this function returns
	tracker.Start(fmt.Sprintf("Booting simulator %s", device.Name), 0)
	defer tracker.Stop()
	// TODO(@cszatmary): We need to figure out a way to mock this for tests.
	sim := simulator.New(device)
	if err := sim.Boot(ctx); err != nil {
		return errors.Wrap(err, errors.Meta{
			Reason: fmt.Sprintf("failed to boot simulator %s", device.Name),
			Op:     op,
		})
	}
	tracker.UpdateMessage("Launching simulator")
	if err = sim.Open(ctx); err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to launch simulator", Op: op})
	}
	tracker.UpdateMessage("Installing app on simulator")
	if err := sim.InstallApp(ctx, appPath); err != nil {
		return errors.Wrap(err, errors.Meta{
			Reason: fmt.Sprintf("failed to install app at path %s on simulator %s", appPath, device.Name),
			Op:     op,
		})
	}
	appDataPath, err := sim.GetAppDataPath(ctx, a.BundleID)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Reason: fmt.Sprintf("failed to get path to data for app %s", a.BundleID),
			Op:     op,
		})
	}
	if opts.DataPath != "" {
		tracker.UpdateMessage("Injecting data files into simulator")
		if err := file.CopyDirContents(opts.DataPath, appDataPath); err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.IO,
				Reason: "failed to inject data into simulator",
				Op:     op,
			})
		}
	}
	tracker.UpdateMessage("Setting environment variables")
	for k, v := range a.EnvVars {
		tracker.Debugf("Setting %s to %s", k, v)
		// Env vars can be passed to simctl if they are set in the calling environment with a SIMCTL_CHILD_ prefix.
		os.Setenv(fmt.Sprintf("SIMCTL_CHILD_%s", k), v)
	}
	tracker.UpdateMessage("Launching app in simulator")
	if err := sim.LaunchApp(ctx, a.BundleID); err != nil {
		return errors.Wrap(err, errors.Meta{
			Reason: fmt.Sprintf("failed to launch app %s on simulator %s", a.BundleID, device.Name),
			Op:     op,
		})
	}
	tracker.Infof("App data directory is located at: %s\n", appDataPath)
	return nil
}

// AppiOSLogsPathOptions customizes the behaviour of AppiOSLogsPath.
// All fields are optional.
type AppiOSLogsPathOptions struct {
	// IOSVersion is the iOS version to use.
	IOSVersion string
	// DeviceName is the name of the device to use.
	DeviceName string
}

// AppiOSLogsPath returns the path where logs are stored for the given simulator.
func (e *Engine) AppiOSLogsPath(ctx context.Context, opts AppiOSLogsPathOptions) (string, error) {
	const op = errors.Op("engine.Engine.AppiOSRun")
	tracker := progress.TrackerFromContext(ctx)
	device, err := e.resolveDevice(ctx, opts.IOSVersion, opts.DeviceName, simulator.DeviceTypeUnspecified, op)
	if err != nil {
		return "", err
	}
	tracker.Debugf("Found device UDID: %s", device.UDID)
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Kind:   errkind.Internal,
			Reason: "unable to find user home directory",
			Op:     op,
		})
	}
	return filepath.Join(homedir, "Library/Logs/CoreSimulator", device.UDID, "system.log"), nil
}

func (e *Engine) resolveDevice(ctx context.Context, iosVersion, deviceName string, deviceType simulator.DeviceType, op errors.Op) (simulator.Device, error) {
	tracker := progress.TrackerFromContext(ctx)
	// Figure out default iOS version if it wasn't provided
	if iosVersion == "" {
		var err error
		iosVersion, err = e.deviceList.GetLatestiOSVersion()
		if err != nil {
			return simulator.Device{}, errors.Wrap(err, errors.Meta{Reason: "unable to get latest iOS version", Op: op})
		}
		tracker.Infof("No iOS version provided, defaulting to version %s", iosVersion)
	}
	if deviceName == "" {
		// Figure out default iOS device if it wasn't provided
		device, err := e.deviceList.GetDefaultDevice(iosVersion, deviceType)
		if err != nil {
			return device, errors.Wrap(err, errors.Meta{Reason: "failed to get default iOS simulator", Op: op})
		}
		tracker.Infof("No iOS simulator provided, defaulting to %s\n", device.Name)
		return device, nil
	}
	// Find specified device
	device, err := e.deviceList.GetDevice(iosVersion, deviceName)
	if err != nil {
		return device, errors.Wrap(err, errors.Meta{Reason: "failed to get simulator device", Op: op})
	}
	return device, nil
}

// AppDesktopRunOptions customizes the behaviour of AppDesktopRun.
// All fields are optional.
type AppDesktopRunOptions struct {
	// Branch is the name of the Git branch associated to the build to run.
	Branch string
}

func (e *Engine) AppDesktopRun(ctx context.Context, appName string, opts AppDesktopRunOptions) error {
	const op = errors.Op("engine.Engine.AppDesktopRun")
	tracker := progress.TrackerFromContext(ctx)
	a, err := e.desktopApps.Get(appName)
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "unable to resolve desktop app", Op: op})
	}
	// Override branch if one was provided
	if opts.Branch != "" {
		a.Branch = opts.Branch
	}

	// Download the app
	var appPath string
	err = progress.Run(ctx, progress.RunOptions{
		Message: fmt.Sprintf("Downloading iOS app %s", a.FullName()),
	}, func(ctx context.Context) (err error) {
		// NOTE(@cszatmary): This is not ideal because this runs in a separate goroutine
		// and it is modifying shared state by assigning to appPath.
		// However, progress.Run provides synchronization so we don't have to worry about
		// a race condition. Once go 1.18 is out, progress.Run should be changed to a generic
		// function so we could return appPath.
		appPath, err = e.downloadApp(ctx, a, app.TypeDesktop, op)
		return err
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Reason: fmt.Sprintf("failed to download iOS app %s", a.FullName()),
			Op:     op,
		})
	}

	tracker.Start("Setting environment variables", 0)
	defer tracker.Stop()
	// Set env vars so they are available in the app process
	for k, v := range a.EnvVars {
		tracker.Debugf("Setting %s to %s", k, v)
		os.Setenv(k, v)
	}
	tracker.UpdateMessage("Launching app")
	// TODO(@cszatmary): probably want to figure out a better way to abstract opening an app cross platform
	if util.IsMacOS {
		w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
		defer w.Close()
		args := []string{"open", appPath}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdout = w
		cmd.Stderr = w
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, errors.Meta{
				Reason: fmt.Sprintf("failed to run %q", strings.Join(args, " ")),
				Op:     op,
			})
		}
	} else {
		return errors.New(errkind.Invalid, "running desktop apps is not supported on your platform", op)
	}
	return nil
}

func (e *Engine) downloadApp(ctx context.Context, a app.App, appType app.Type, op errors.Op) (string, error) {
	tracker := progress.TrackerFromContext(ctx)
	storageProvider, err := e.getStorageProvider(a.Storage.Provider)
	if err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Reason: fmt.Sprintf("failed to get storage provider %s", a.Storage.Provider),
			Op:     op,
		})
	}

	// Look up the latest build sha for user-specified branch and app.
	remoteDir := path.Join(a.Name, a.Branch)
	tracker.Debugf("Checking objects on %s in bucket %s matching prefix %s", a.Storage.Provider, a.Storage.Bucket, remoteDir)
	remoteBuilds, err := storageProvider.ListObjectKeysByPrefix(ctx, a.Storage.Bucket, remoteDir)
	if err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Reason: fmt.Sprintf("failed to list builds in %s in dir %s", a.Storage.Provider, remoteDir),
		})
	}
	if len(remoteBuilds) == 0 {
		return "", errors.New(errkind.Invalid, fmt.Sprintf("no builds found for %s", remoteDir), op)
	} else if len(remoteBuilds) > 1 {
		// We only expect one build per branch. If we find two, its likely a bug or some kind of
		// race condition from the build-uploading side.
		// If this gets clunky we can determine a sort order for the builds.
		return "", errors.New(
			errkind.Invalid,
			fmt.Sprintf("expected a single build but found multiple: %+v", remoteBuilds),
			op,
		)
	}
	remoteTarballPath := remoteBuilds[0]
	remoteBuildFilename := path.Base(remoteTarballPath)

	// Decide whether or not to pull down a new version.
	var localBranchDir string
	if appType == app.TypeiOS {
		localBranchDir = filepath.Join(e.workdir, iosDir, a.FullName(), a.Branch)
	} else {
		localBranchDir = filepath.Join(e.workdir, desktopDir, a.FullName(), a.Branch)
	}
	tracker.Debugf("checking %s to see if we need to download a new version of the app", localBranchDir)
	globPattern := filepath.Join(localBranchDir, "*.app")
	localBuilds, err := filepath.Glob(globPattern)
	if err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Kind:   errkind.Internal,
			Reason: fmt.Sprintf("failed to glob for %s", globPattern),
			Op:     op,
		})
	}

	if len(localBuilds) > 1 {
		// If we have more than one local build we are somehow in a bad state. Recover gracefully.
		tracker.Debugf("Got the following builds: %+v. Only expecting one build", localBuilds)
		tracker.Debugf("Cleaning and downloading fresh build")
	} else if len(localBuilds) == 1 {
		// If there is a local build, get latest sha from github for desired branch to see
		// if the available remote build corresponds to the latest commit on the branch.
		localBuild := localBuilds[0]
		tracker.Debugf("Checking latest github sha for %s/%s", a.GitRepo, a.Branch)
		latestGitsha, err := e.gitClient.GetBranchHeadSha(ctx, a.GitRepo, a.Branch)
		if err != nil {
			return "", errors.Wrap(err, errors.Meta{
				Reason: fmt.Sprintf("failed getting branch head sha for %s/%s", a.GitRepo, a.Branch),
				Op:     op,
			})
		}
		tracker.Debugf("Latest github sha is %s", latestGitsha)
		if !strings.HasPrefix(remoteBuildFilename, latestGitsha) {
			tracker.Warnf("sha of remote build %s does not match latest github sha %s for branch %s", remoteBuildFilename, latestGitsha, a.Branch)
		}

		currentSha := strings.Split(filepath.Base(localBuild), ".")[0]
		remoteSha := strings.Split(remoteBuildFilename, ".")[0]
		tracker.Debugf("Current local build sha is %s", currentSha)
		tracker.Debugf("Latest s3 sha is %s", remoteSha)
		if currentSha == remoteSha {
			// We have a local build that matches the latest version, no need to download
			tracker.Debugf("Current build sha matches remote sha")
			return localBuild, nil
		}
		tracker.Debugf("Current build sha is different from s3 sha, deleting local version")
	}
	// Clean up the local build dir before downloading
	if err := os.RemoveAll(localBranchDir); err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to remove %s", localBranchDir),
			Op:     op,
		})
	}

	// Download and untar the latest build
	tracker.Debugf("Downloading %s/%s from %s to %s", a.Storage.Bucket, remoteTarballPath, a.Storage.Provider, localBranchDir)
	r, err := storageProvider.GetObject(ctx, a.Storage.Bucket, remoteTarballPath)
	if err != nil {
		return "", errors.Wrap(err, errors.Meta{Op: op})
	}
	defer r.Close()
	if err := file.Untar(localBranchDir, r); err != nil {
		return "", errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: "failed to untar app archive",
			Op:     op,
		})
	}

	// Get the path to the extracted app.
	// NOTE(@cszatmary): We are assuming the app within the tar file will have the same
	// name as the tar file minus the extension. We should either make this an explicit requirement
	// in the docs, or come up with a better way to find the app path, such as reading reading the directory.
	appPath := filepath.Join(localBranchDir, remoteBuildFilename)
	// There are multiple extensions that can be used with a tar file, try each.
	for _, ext := range []string{".tar", ".tar.gz", ".tgz"} {
		if strings.HasSuffix(appPath, ext) {
			appPath = strings.TrimSuffix(appPath, ext)
			break
		}
	}
	return appPath, nil
}

// AppDesktopRunOptions customizes the behaviour of AppList.
// All fields are optional.
type AppListOptions struct {
	ListiOSApps     bool
	ListDesktopApps bool
}

type AppListResult struct {
	IOSApps     []string
	DesktopApps []string
}

// AppList lists the names of available iOS and desktopApps.
func (e *Engine) AppList(opts AppListOptions) AppListResult {
	var res AppListResult
	if opts.ListiOSApps {
		res.IOSApps = e.iosApps.Names()
	}
	if opts.ListDesktopApps {
		res.DesktopApps = e.desktopApps.Names()
	}
	return res
}
