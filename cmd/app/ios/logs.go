package ios

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/app"
	"github.com/TouchBistro/tb/simulator"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type logsOptions struct {
	iosVersion    string
	deviceName    string
	numberOfLines string
}

var logOpts logsOptions

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Displays logs from the given simulator",
	Long: `Displays logs from the given simulator.

Examples:
- displays the last 10 logs in the default iOS simulator
	tb app logs

- displays the last 20 logs in an iOS 12.4 iPad Air 2 simulator
	tb app logs --number 20 --ios-version 12.4 --device iPad Air 2`,
	Run: func(cmd *cobra.Command, args []string) {
		deviceData, err := simulator.ListDevices()
		if err != nil {
			fatal.ExitErr(err, "Failed to get list of simulators")
		}

		deviceList, err := simulator.ParseSimulators(deviceData)
		if err != nil {
			fatal.ExitErr(err, "Failed to find available iOS simulators")
		}

		// Figure out default iOS version if it wasn't provided
		if logOpts.iosVersion == "" {
			logOpts.iosVersion, err = deviceList.GetLatestIOSVersion()
			if err != nil {
				fatal.ExitErr(err, "failed to get latest iOS version")
			}

			log.Infof("No iOS version provided, defaulting to version %s\n", logOpts.iosVersion)
		}

		// Figure out default iOS device if it wasn't provided
		if runOpts.deviceName == "" {
			runOpts.deviceName, err = deviceList.GetDefaultDevice("iOS "+runOpts.iosVersion, app.DeviceTypeAll)
			if err != nil {
				fatal.ExitErr(err, "failed to get default iOS simulator")
			}

			log.Infof("No iOS simulator provided, defaulting to %s\n", runOpts.deviceName)
		}

		log.Debugln("☐ Finding device UDID")

		deviceUDID, err := deviceList.GetDeviceUDID("iOS "+logOpts.iosVersion, logOpts.deviceName)
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to get device UUID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}

		log.Debugf("☑ Found device UDID: %s\n", deviceUDID)

		logsPath := filepath.Join(os.Getenv("HOME"), "Library/Logs/CoreSimulator", deviceUDID, "system.log")
		log.Infof("Attaching to logs for simulator %s\n\n", logOpts.deviceName)

		c := command.New(command.WithStdout(os.Stdout), command.WithStderr(os.Stderr))
		err = c.Exec("tail", "-f", "-n", logOpts.numberOfLines, logsPath)
		if err != nil {
			fatal.ExitErrf(err, "Failed to get logs for simulator %s with iOS version %s", logOpts.deviceName, logOpts.iosVersion)
		}
	},
}

func init() {
	iosCmd.AddCommand(logsCmd)
	logsCmd.Flags().StringVarP(&logOpts.iosVersion, "ios-version", "i", "", "The iOS version to use")
	logsCmd.Flags().StringVarP(&logOpts.deviceName, "device", "d", "", "The name of the device to use")
	logsCmd.Flags().StringVarP(&logOpts.numberOfLines, "number", "n", "10", "The number of lines to display")
}
