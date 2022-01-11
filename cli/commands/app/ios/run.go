package ios

import (
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type runOptions struct {
	iosVersion string
	deviceName string
	dataPath   string
	branch     string
}

func newRunCommand(c *cli.Container) *cobra.Command {
	var opts runOptions
	runCmd := &cobra.Command{
		Use:   "run <app>",
		Args:  cli.ExpectSingleArg("app name"),
		Short: "Run an iOS app build in an iOS Simulator",
		Long: `Runs an iOS app build in an iOS Simulator. Flags can be provided to specify the simulator device and iOS version.

Examples:

Run the current master build of TouchBistro in the default iOS Simulator:

	tb app ios run TouchBistro

Run the build for specific branch in an iOS 12.3 iPad Air 2 simulator:

	tb app ios run TouchBistro --ios-version 12.3 --device iPad Air 2 --branch task/pay-631/fix-thing`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appName := args[0]
			return c.Engine.AppiOSRun(c.Ctx, appName, engine.AppiOSRunOptions{
				IOSVersion: opts.iosVersion,
				DeviceName: opts.deviceName,
				DataPath:   opts.dataPath,
				Branch:     opts.branch,
			})
		},
	}

	flags := runCmd.Flags()
	flags.StringVarP(&opts.iosVersion, "ios-version", "i", "", "The iOS version to use")
	flags.StringVarP(&opts.deviceName, "device", "d", "", "The name of the device to use")
	flags.StringVarP(&opts.branch, "branch", "b", "", "The name of the git branch associated build to pull down and run")
	flags.StringVarP(&opts.dataPath, "data-path", "D", "", "The path to a data directory to inject into the simulator")
	return runCmd
}
