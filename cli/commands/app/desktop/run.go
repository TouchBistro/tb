package desktop

import (
	"fmt"

	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newRunCommand(c *cli.Container) *cobra.Command {
	var runOpts struct {
		branch string
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
		Short: "Runs a desktop application",
		Long: `Runs a desktop application.

	Examples:
	- run the current master build of TouchBistroServer
	  tb app desktop run TouchBistroServer

	- run the build for a specific branch
	  tb app desktop run TouchBistroServer --branch task/bug-631/fix-thing`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appName := args[0]
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			return c.Engine.AppDesktopRun(ctx, appName, engine.AppDesktopRunOptions{
				Branch: runOpts.branch,
			})
		},
	}
	runCmd.Flags().StringVarP(&runOpts.branch, "branch", "b", "", "The name of the git branch associated build to pull down and run")
	return runCmd
}
