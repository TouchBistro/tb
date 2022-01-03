package commands

import (
	"fmt"
	"os"

	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

type execOptions struct {
	skipGitPull bool
}

func newExecCommand(c *cli.Container) *cobra.Command {
	var opts execOptions
	execCmd := &cobra.Command{
		Use: "exec <service> <command> [args...]",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("expected at least 2 args for service name and command to run")
			}
			return nil
		},
		Short: "Execute a command in a service container",
		Long: `Executes a command in a service container.

Examples:

Run yarn db:prepare:test in the core-database container:

	tb exec core-database yarn db:prepare:test

Start an interactive bash shell in the core-database container:

	tb exec core-database bash`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := progress.ContextWithTracker(cmd.Context(), c.Tracker)
			exitCode, err := c.Engine.Exec(ctx, args[0], engine.ExecOptions{
				SkipGitPull: opts.skipGitPull,
				Cmd:         args[1:],
				Stdin:       os.Stdin,
				Stdout:      os.Stdout,
				Stderr:      os.Stderr,
			})
			if err != nil {
				return err
			}
			// Match the exit code of the command
			os.Exit(exitCode)
			return nil
		},
	}

	flags := execCmd.Flags()
	flags.BoolVar(&opts.skipGitPull, "no-git-pull", false, "Don't update git repositories")
	return execCmd
}
