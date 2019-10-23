package ios

import (
	"fmt"
	"os"

	"github.com/TouchBistro/tb/fatal"
	"github.com/TouchBistro/tb/simulator"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	numberOfLines string
)

var logsCmd = &cobra.Command{
	Use:   "logs <device-name>",
	Short: "Displays logs from the given simulator",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugln("☐ Finding device UDID")

		deviceName := args[0]
		deviceUDID, err := simulator.GetDeviceUDID("iOS "+iosVersion, deviceName)
		if err != nil {
			fatal.ExitErr(err, "☒ Failed to get device UUID.\nRun \"xcrun simctl list devices\" to list available simulators.")
		}

		log.Debugf("☑ Found device UDID: %s\n", deviceUDID)

		logsPath := fmt.Sprintf("%s/Library/Logs/CoreSimulator/%s/system.log", os.Getenv("HOME"), deviceUDID)
		err = util.Exec("tail", "tail", "-f", "-n", numberOfLines, logsPath)
		if err != nil {
			fatal.ExitErrf(err, "Failed to get logs for simulator %s with iOS version %s", deviceName, iosVersion)
		}
	},
}

func init() {
	iosCmd.AddCommand(logsCmd)
	runCmd.Flags().StringVarP(&numberOfLines, "number", "n", "10", "The number of lines to display")
}
