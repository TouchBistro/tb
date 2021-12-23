package ios

import (
	"os"
	"os/exec"

	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type logsOptions struct {
	iosVersion    string
	deviceName    string
	numberOfLines string
}

func newLogsCommand(c *cli.Container) *cobra.Command {
	var opts logsOptions
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Displays logs from the given simulator",
		Long: `Displays logs from the given simulator.

Examples:
- displays the last 10 logs in the default iOS simulator
	tb app logs

- displays the last 20 logs in an iOS 12.4 iPad Air 2 simulator
	tb app logs --number 20 --ios-version 12.4 --device iPad Air 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			logsPath, err := c.Engine.AppiOSLogsPath(ctx, engine.AppiOSLogsPathOptions{
				IOSVersion: opts.iosVersion,
				DeviceName: opts.deviceName,
			})
			if err != nil {
				return err
			}
			c.Tracker.Info("Attaching to simulator logs")
			tail := exec.CommandContext(ctx, "tail", "-f", "-n", opts.numberOfLines, logsPath)
			tail.Stdout = os.Stdout
			tail.Stderr = os.Stderr
			if err := tail.Run(); err != nil {
				code := tail.ProcessState.ExitCode()
				if code != -1 {
					os.Exit(code)
				}
				os.Exit(1)
			}
			return nil
		},
	}

	flags := logsCmd.Flags()
	flags.StringVarP(&opts.iosVersion, "ios-version", "i", "", "The iOS version to use")
	flags.StringVarP(&opts.deviceName, "device", "d", "", "The name of the device to use")
	flags.StringVarP(&opts.numberOfLines, "number", "n", "10", "The number of lines to display")
	return logsCmd
}