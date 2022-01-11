package desktop

import (
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type runOptions struct {
	branch string
}

func newRunCommand(c *cli.Container) *cobra.Command {
	var opts runOptions
	runCmd := &cobra.Command{
		Use:   "run <app>",
		Args:  cli.ExpectSingleArg("app name"),
		Short: "Run a desktop app",
		Long: `Runs a desktop application.

Examples:

Run the current master build of TouchBistroServer:

	tb app desktop run TouchBistroServer

Run the build for a specific branch:

	tb app desktop run TouchBistroServer --branch task/bug-631/fix-thing`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appName := args[0]
			return c.Engine.AppDesktopRun(c.Ctx, appName, engine.AppDesktopRunOptions{
				Branch: opts.branch,
			})
		},
	}

	flags := runCmd.Flags()
	flags.StringVarP(&opts.branch, "branch", "b", "", "The name of the git branch associated build to pull down and run")
	return runCmd
}
