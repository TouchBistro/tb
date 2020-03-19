package ios

import (
	"fmt"
	"os"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
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

		// Override branch if one was provided
		if runOpts.branch != "" {
			a.Branch = runOpts.branch
		}

		if runOpts.iosVersion == "" {
			runOpts.iosVersion = simulator.GetLatestIOSVersion()
			log.Infof("No iOS version provided, defaulting to version %s\n", runOpts.iosVersion)
		}

		downloadDest := config.IOSBuildPath()
		// Check disk utilisation by ios directory
		usageBytes, err := file.DirSize(downloadDest)
		if err != nil {
			fatal.ExitErr(err, "Error checking ios build disk space usage")
		}
		log.Infof("Current ios build disk usage: %.2fGB", float64(usageBytes)/1024.0/1024.0/1024.0)

		appPath := appCmd.DownloadLatestApp(a, downloadDest)

		log.Debugln("☐ Finding device UDID")
		deviceUDID, err := simulator.GetDeviceUDID("iOS "+runOpts.iosVersion, runOpts.deviceName)
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to get device UDID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}

		log.Debugf("☑ Found device UDID: %s\n", deviceUDID)
		log.Infof("☐ Booting Simulator %s\n", runOpts.deviceName)

		err = simulator.Boot(deviceUDID)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to boot simulator %s", runOpts.deviceName)
		}

		log.Infof("☑ Booted simulator %s\n", runOpts.deviceName)
		log.Debugln("☐ Opening simulator app")

		err = simulator.Open()
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to launch simulator")
		}

		log.Debugln("☑ Opened simulator app")
		log.Infof("☐ Installing app on %s\n", runOpts.deviceName)

		err = simulator.InstallApp(deviceUDID, appPath)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to install app at path %s on simulator %s", appPath, runOpts.deviceName)
		}

		log.Infof("☑ Installed app %s on %s\n", a.BundleID, runOpts.deviceName)

		appDataPath, err := simulator.GetAppDataPath(deviceUDID, a.BundleID)
		if err != nil {
			fatal.ExitErrf(err, "Failed to get path to data for app %s", a.BundleID)
		}

		if runOpts.dataPath != "" {
			log.Infoln("☐ Injecting data files into simulator")

			err = file.CopyDirContents(runOpts.dataPath, appDataPath)
			if err != nil {
				fatal.ExitErrf(err, "☒ Failed to inject data into simulator")
			}

			log.Infoln("☑ Injected data into simulator")
		}

		log.Info("☐ Setting environment variables")

		for k, v := range a.EnvVars {
			log.Debugf("Setting %s to %s", k, v)
			// Env vars can be passed to simctl if they are set in the calling environment with a SIMCTL_CHILD_ prefix.
			os.Setenv(fmt.Sprintf("SIMCTL_CHILD_%s", k), v)
		}

		log.Info("☑ Done setting environment variables")
		log.Info("☐ Launching app in simulator")

		err = simulator.LaunchApp(deviceUDID, a.BundleID)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to launch app %s on simulator %s", a.BundleID, runOpts.deviceName)
		}

		log.Infof("☑ Launched app %s on %s\n", a.BundleID, runOpts.deviceName)
		log.Infof("App data directory is located at: %s\n", appDataPath)
		log.Info("🎉🎉🎉 Enjoy!")
	},
}

func init() {
	iosCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runOpts.iosVersion, "ios-version", "i", "", "The iOS version to use")
	runCmd.Flags().StringVarP(&runOpts.deviceName, "device", "d", "iPad Air (3rd generation)", "The name of the device to use")
	runCmd.Flags().StringVarP(&runOpts.branch, "branch", "b", "", "The name of the git branch associated build to pull down and run")
	runCmd.Flags().StringVarP(&runOpts.dataPath, "data-path", "D", "", "The path to a data directory to inject into the simulator")
}
