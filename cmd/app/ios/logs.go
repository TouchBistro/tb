package ios

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
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
		if logOpts.iosVersion == "" {
			logOpts.iosVersion = simulator.GetLatestIOSVersion()
			log.Infof("No iOS version provided, defaulting to version %s\n", logOpts.iosVersion)
		}

		log.Debugln("☐ Finding device UDID")

		deviceUDID, err := simulator.GetDeviceUDID("iOS "+logOpts.iosVersion, logOpts.deviceName)
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to get device UUID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}

		log.Debugf("☑ Found device UDID: %s\n", deviceUDID)

		logsPath := filepath.Join(os.Getenv("HOME"), "Library/Logs/CoreSimulator", deviceUDID, "system.log")
		log.Infof("Attaching to logs for simulator %s\n\n", logOpts.deviceName)

		err = command.Exec("tail", []string{"-f", "-n", logOpts.numberOfLines, logsPath}, "ios-logs-tail", func(cmd *exec.Cmd) {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		})
		if err != nil {
			fatal.ExitErrf(err, "Failed to get logs for simulator %s with iOS version %s", logOpts.deviceName, logOpts.iosVersion)
		}
	},
}

func init() {
	iosCmd.AddCommand(logsCmd)
	logsCmd.Flags().StringVarP(&logOpts.iosVersion, "ios-version", "i", "", "The iOS version to use")
	logsCmd.Flags().StringVarP(&logOpts.deviceName, "device", "d", "iPad Air (3rd generation)", "The name of the device to use")
	logsCmd.Flags().StringVarP(&logOpts.numberOfLines, "number", "n", "10", "The number of lines to display")
}
