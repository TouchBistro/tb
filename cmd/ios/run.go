package ios

import (
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	"github.com/TouchBistro/tb/simulator"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	iosVersion string
	deviceName string
	appPath    string
	dataPath   string
)

var runCmd = &cobra.Command{
	Use:   "run <app-name>",
	Short: "Runs iOS apps in the iOS Simulator",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		app, ok := config.Apps()[appName]
		if !ok {
			fatal.Exitf("Error: No iOS app with name %s\n", appName)
		}

		log.Debugln("☐ Finding device UUID")
		deviceUUID, err := simulator.GetDeviceUUID("iOS "+iosVersion, deviceName)
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to get device UUID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}

		log.Debugf("☑ Found device UUID: %s\n", deviceUUID)
		log.Infof("☐ Booting Simulator %s\n", deviceName)

		err = simulator.Boot(deviceUUID)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to boot simulator %s", deviceName)
		}

		log.Infof("☑ Booted simulator %s\n", deviceName)
		log.Debugln("☐ Opening simulator app")

		err = simulator.Open()
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to launch simulator")
		}

		log.Debugln("☑ Opened simulator app")
		log.Infof("☐ Installing app on %s\n", deviceName)

		err = simulator.InstallApp(deviceUUID, appPath)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to install app at path %s on simulator %s", appPath, deviceName)
		}

		log.Infof("☑ Installed app %s on %s\n", app.BundleID, deviceName)

		if dataPath != "" {
			log.Infoln("☐ Injecting data files into simulator")

			appDataPath, err := simulator.GetAppDataPath(deviceUUID, app.BundleID)
			if err != nil {
				fatal.ExitErrf(err, "Failed to get path to data for app %s", app.BundleID)
			}

			err = util.CopyDirContents(dataPath, appDataPath)
			if err != nil {
				fatal.ExitErrf(err, "☒ Failed to inject data into simulator")
			}

			log.Infoln("☑ Injected data into simulator")
		}

		log.Info("☐ Launching app in simulator")

		err = simulator.LaunchApp(deviceUUID, app.BundleID)
		if err != nil {
			fatal.ExitErrf(err, "☒ Failed to launch app %s on simulator %s", app.BundleID, deviceName)
		}

		log.Infof("☑ Launched app %s on %s\n", app.BundleID, deviceName)
		log.Info("🎉🎉🎉 Enjoy!")
	},
}

func init() {
	iosCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&iosVersion, "ios-version", "i", "12.2", "The iOS version to use")
	runCmd.Flags().StringVarP(&deviceName, "device", "d", "iPad Air 2", "The name of the device to use")
	runCmd.Flags().StringVar(&dataPath, "data-path", "", "The path to a data directory to inject into the simulator")

	// TODO remove this shit once we pull from S3
	runCmd.Flags().StringVarP(&appPath, "path", "p", "", "The path to the app build")

	err := runCmd.MarkFlagRequired("path")
	if err != nil {
		fatal.ExitErrf(err, "No such command %s", "path")
	}
}
