package ios

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/TouchBistro/tb/fatal"
	"github.com/TouchBistro/tb/simulator"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	numberOfLines string
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Displays logs from the given simulator",
	Long: `Displays logs from the given simulator.
	
Examples:
- displays the last 10 logs in the default iOS simulator
	tb logs

- displays the last 20 logs in an iOS 12.4 iPad Air 2 simulator
	tb logs --number 20 --ios-version 12.4 --device iPad Air 2`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugln("☐ Finding device UDID")

		deviceUDID, err := simulator.GetDeviceUDID("iOS "+iosVersion, deviceName)
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to get device UUID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}

		log.Debugf("☑ Found device UDID: %s\n", deviceUDID)

		logsPath := fmt.Sprintf("%s/Library/Logs/CoreSimulator/%s/system.log", os.Getenv("HOME"), deviceUDID)
		log.Infof("Attaching to logs for simulator %s\n\n", deviceName)

		execCmd := exec.Command("tail", "-f", "-n", numberOfLines, logsPath)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		err = execCmd.Run()
		if err != nil {
			fatal.ExitErrf(err, "Failed to get logs for simulator %s with iOS version %s", deviceName, iosVersion)
		}
	},
}

func init() {
	iosCmd.AddCommand(logsCmd)
	logsCmd.Flags().StringVarP(&iosVersion, "ios-version", "i", "13.1", "The iOS version to use")
	logsCmd.Flags().StringVarP(&deviceName, "device", "d", "iPad Air (3rd generation)", "The name of the device to use")
	logsCmd.Flags().StringVarP(&numberOfLines, "number", "n", "10", "The number of lines to display")
}
