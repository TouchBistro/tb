package ios

import (
	"fmt"
	"os"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/spinner"
	appCmd "github.com/TouchBistro/tb/cmd/app"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/simulator"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type runOptions struct {
	iosVersion string
	deviceName string
	dataPath   string
	branch     string
}

var runOpts runOptions

var runCmd = &cobra.Command{
	Use: "run",
	Args: func(cmd *cobra.Command, args []string) error {
		// Verify that the app name was provided as a single arg
		if len(args) < 1 {
			return errors.New("app name is required as an argument")
		} else if len(args) > 1 {
			return errors.New("only one argument is accepted")
		}

		return nil
	},
	Short: "Runs an iOS app build in an iOS Simulator",
	Long: `Runs an iOS app build in an iOS Simulator.

Examples:
- run the current master build of TouchBistro in the default iOS Simulator
	tb app ios run TouchBistro

- run the build for specific branch in an iOS 12.3 iPad Air 2 simulator
	tb app ios run TouchBistro --ios-version 12.3 --device iPad Air 2 --branch task/pay-631/fix-thing`,
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		a, err := config.LoadedIOSApps().Get(appName)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a valid iOS app\n", appName)
		}

		deviceData, err := simulator.ListDevices()
		if err != nil {
			fatal.ExitErr(err, "Failed to get list of simulators")
		}

		deviceList, err := simulator.ParseSimulators(deviceData)
		if err != nil {
			fatal.ExitErr(err, "Failed to find available iOS simulators")
		}

		// Override branch if one was provided
		if runOpts.branch != "" {
			a.Branch = runOpts.branch
		}

		// Figure out default iOS version if it wasn't provided
		if runOpts.iosVersion == "" {
			runOpts.iosVersion, err = deviceList.GetLatestIOSVersion()
			if err != nil {
				fatal.ExitErr(err, "failed to get latest iOS version")
			}
			log.Debugf("No iOS version provided, defaulting to version %s", runOpts.iosVersion)
		}

		// Figure out default iOS device if it wasn't provided
		if runOpts.deviceName == "" {
			runOpts.deviceName, err = deviceList.GetDefaultDevice("iOS "+runOpts.iosVersion, a.DeviceType())
			if err != nil {
				fatal.ExitErr(err, "failed to get default iOS simulator")
			}
			log.Debugf("No iOS simulator provided, defaulting to %s", runOpts.deviceName)
		} else {
			// Make sure provided device is valid for the given app
			isValid := simulator.IsValidDevice(runOpts.deviceName, a.DeviceType())
			if !isValid {
				fatal.Exitf("Device %s is not supported by iOS app %s\n", runOpts.deviceName, appName)
			}
		}

		downloadDest := config.IOSBuildPath()
		// Check disk utilisation by ios directory
		usageBytes, err := file.DirSize(downloadDest)
		if err != nil {
			fatal.ExitErr(err, "Error checking ios build disk space usage")
		}
		log.Infof("Current ios build disk usage: %.2fGB", float64(usageBytes)/(1024*1024*1024))

		appPath := appCmd.DownloadLatestApp(a, downloadDest)

		log.Debug("Finding device UDID")
		deviceUDID, err := deviceList.GetDeviceUDID("iOS "+runOpts.iosVersion, runOpts.deviceName)
		if err != nil {
			fatal.ExitErr(err, "Failed to get device UDID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}
		log.Debugf("Found device UDID: %s", deviceUDID)

		s := spinner.New(
			spinner.WithStartMessage("Running iOS app"+a.FullName()),
			spinner.WithPersistMessages(log.IsLevelEnabled(log.DebugLevel)),
		)
		log.SetOutput(s)
		s.Start()

		s.UpdateMessage("Booting Simulator " + runOpts.deviceName)
		err = simulator.Boot(deviceUDID)
		if err != nil {
			s.Stop()
			fatal.ExitErrf(err, "Failed to boot simulator %s", runOpts.deviceName)
		}
		log.Debugf("Booted simulator %s", runOpts.deviceName)

		s.UpdateMessage("Opening simulator app")
		err = simulator.Open()
		if err != nil {
			s.Stop()
			fatal.ExitErr(err, "Failed to launch simulator")
		}
		log.Debug("Opened simulator app")

		s.UpdateMessage("Installing app on " + runOpts.deviceName)
		err = simulator.InstallApp(deviceUDID, appPath)
		if err != nil {
			s.Stop()
			fatal.ExitErrf(err, "Failed to install app at path %s on simulator %s", appPath, runOpts.deviceName)
		}
		log.Debugf("Installed app %s on %s\n", a.BundleID, runOpts.deviceName)

		appDataPath, err := simulator.GetAppDataPath(deviceUDID, a.BundleID)
		if err != nil {
			s.Stop()
			fatal.ExitErrf(err, "Failed to get path to data for app %s", a.BundleID)
		}
		if runOpts.dataPath != "" {
			s.UpdateMessage("Injecting data files into simulator")
			err = file.CopyDirContents(runOpts.dataPath, appDataPath)
			if err != nil {
				s.Stop()
				fatal.ExitErrf(err, "Failed to inject data into simulator")
			}
			log.Debugf("Injected data into simulator")
		}

		s.UpdateMessage("Setting environment variables")
		for k, v := range a.EnvVars {
			log.Debugf("Setting %s to %s", k, v)
			// Env vars can be passed to simctl if they are set in the calling environment with a SIMCTL_CHILD_ prefix.
			os.Setenv(fmt.Sprintf("SIMCTL_CHILD_%s", k), v)
		}
		log.Debugf("Done setting environment variables")

		s.UpdateMessage("Launching app in simulator")
		err = simulator.LaunchApp(deviceUDID, a.BundleID)
		s.Stop()
		if err != nil {
			fatal.ExitErrf(err, "Failed to launch app %s on simulator %s", a.BundleID, runOpts.deviceName)
		}
		log.SetOutput(os.Stderr)

		log.Infof("🎉🎉🎉 Launched app %s on %s, enjoy!", a.FullName(), runOpts.deviceName)
		log.Infof("App data directory is located at: %s", appDataPath)
	},
}

func init() {
	iosCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runOpts.iosVersion, "ios-version", "i", "", "The iOS version to use")
	runCmd.Flags().StringVarP(&runOpts.deviceName, "device", "d", "", "The name of the device to use")
	runCmd.Flags().StringVarP(&runOpts.branch, "branch", "b", "", "The name of the git branch associated build to pull down and run")
	runCmd.Flags().StringVarP(&runOpts.dataPath, "data-path", "D", "", "The path to a data directory to inject into the simulator")
}
