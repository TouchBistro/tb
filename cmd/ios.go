package cmd

import (
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	iosVersion  string
	deviceName  string
	appBundleID string
	appPath     string
)

var iosCmd = &cobra.Command{
	Use:   "ios",
	Short: "Runs iOS apps in the iOS Simulator",
	Run: func(cmd *cobra.Command, args []string) {
		err := config.InitIOS()
		if err != nil {
			fatal.ExitErr(err, "Failed to initialize iOS config")
		}

		log.Debugln("☐ Finding device UUID")
		deviceUUID, err := config.GetDeviceUUID(iosVersion, deviceName)
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to get device UUID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}

		log.Debugf("☑ Found device UUID: %s\n", deviceUUID)
		log.Infof("☐ Booting Simulator %s\n", deviceName)

		execID := "simctl"
		err = util.Exec(execID, "xcrun", "simctl", "bootstatus", deviceUUID, "-b")
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to boot simulator %s with UUID %s", deviceName, deviceUUID)
		}

		log.Infof("☑ Booted simulator %s\n", deviceName)
		log.Debugln("☐ Opening simulator app")

		err = util.Exec("open-sim", "open", "-a", "simulator")
		if err != nil {
			fatal.ExitErr(err, "Failed to open simulator application")
		}

		log.Debugln("☑ Opened simulator app")
		log.Infof("☐ Installing app on %s\n", deviceName)

		err = util.Exec(execID, "xcrun", "simctl", "install", deviceUUID, appPath)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to install app on simulator %s with UUID %s", deviceName, deviceUUID)
		}

		log.Infof("☑ Installed app %s on %s\n", appBundleID, deviceName)
		log.Info("☐ Launching app in simulator")

		err = util.Exec(execID, "xcrun", "simctl", "launch", deviceUUID, appBundleID)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to launch app %s on simulator %s with UUID %s", appBundleID, deviceName, deviceUUID)
		}

		log.Infof("☑ Launched app %s on %s\n", appBundleID, deviceName)
		log.Info("🎉🎉🎉 Enjoy!")
	},
}

func init() {
	rootCmd.AddCommand(iosCmd)
	iosCmd.Flags().StringVarP(&iosVersion, "version", "v", "iOS 12.2", "iOS version to use")
	iosCmd.Flags().StringVarP(&deviceName, "device", "d", "iPad Air 2", "The name of the device to use")
	iosCmd.Flags().StringVarP(&appBundleID, "bundleID", "b", "com.touchbistro.TouchBistro", "The application bundle identifier")
	iosCmd.Flags().StringVarP(&appPath, "path", "p", "", "The path to the app build")
	iosCmd.MarkFlagRequired("path")
}
