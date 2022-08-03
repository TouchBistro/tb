package commands

import (
	"fmt"
	"os"

	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

func newExecCommand(c *cli.Container) *cobra.Command {
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
			exitCode, err := c.Engine.Exec(c.Ctx, args[0], engine.ExecOptions{
				Cmd:    args[1:],
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			})
			if err != nil {
				return err
			}
			if exitCode != 0 {
				// Match the exit code of the command
				return &fatal.Error{Code: exitCode}
			}
			return nil
		},
	}

	flags := execCmd.Flags()
	flags.Bool("no-git-pull", false, "Don't update git repositories")
	err := flags.MarkDeprecated("no-git-pull", "it is a no-op and will be removed")
	if err != nil {
		// MarkDeprecated only errors if the flag name is wrong or the message isn't set
		// which is a programming error, so we wanna blow up
		panic(err)
	}
	return execCmd
}
