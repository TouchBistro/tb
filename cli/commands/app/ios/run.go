package ios

import (
	"fmt"

	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newRunCommand(c *cli.Container) *cobra.Command {
	var runOpts struct {
		iosVersion string
		deviceName string
		dataPath   string
		branch     string
	}
	runCmd := &cobra.Command{
		Use: "run",
		Args: func(cmd *cobra.Command, args []string) error {
			// Verify that the app name was provided as a single arg
			if len(args) < 1 {
				return fmt.Errorf("app name is required as an argument")
			} else if len(args) > 1 {
				return fmt.Errorf("only one argument is accepted")
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
		RunE: func(cmd *cobra.Command, args []string) error {
			appName := args[0]
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			return c.Engine.AppiOSRun(ctx, appName, engine.AppiOSRunOptions{
				IOSVersion: runOpts.iosVersion,
				DeviceName: runOpts.deviceName,
				DataPath:   runOpts.dataPath,
				Branch:     runOpts.branch,
			})
		},
	}
	runCmd.Flags().StringVarP(&runOpts.iosVersion, "ios-version", "i", "", "The iOS version to use")
	runCmd.Flags().StringVarP(&runOpts.deviceName, "device", "d", "", "The name of the device to use")
	runCmd.Flags().StringVarP(&runOpts.branch, "branch", "b", "", "The name of the git branch associated build to pull down and run")
	runCmd.Flags().StringVarP(&runOpts.dataPath, "data-path", "D", "", "The path to a data directory to inject into the simulator")
	return runCmd
}
