package commands

import (
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
		Use:   "exec <service-name> <command> [additional-commands...]",
		Short: "executes a command in a service container",
		Long: `Executes a command in a service container.

	Examples:
	- run yarn db:prepare:test in the core-database container.
		tb exec core-database yarn db:prepare:test

	- start an interactive shell in the core-database container.
		tb exec core-database bash`,
		Args: cobra.MinimumNArgs(2),
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
	flags.BoolVar(&opts.skipGitPull, "no-git-pull", false, "dont update git repositories")
	return execCmd
}
