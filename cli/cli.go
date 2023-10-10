// Package cli provides general functionality for all CLI commands.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/engine"
	"github.com/spf13/cobra"
)

// ExpectSingleArg returns a function that validates the command only receives a single arg.
// name is the name of the arg and is used in the error message.
func ExpectSingleArg(name string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("expected 1 arg for %s", name)
		} else if len(args) > 1 {
			return fmt.Errorf("expected 1 arg for %s, received %d args", name, len(args))
		}
		return nil
	}
}

// Container stores all the dependencies that can be used by commands.
//
// Fields on Container are only safe to use within the Run function of a command
// where they are guaranteed to be initialized. Outside of Run, the only usage of
// a Container instance should be to pass it to command constructors so they can capture it.
type Container struct {
	Engine  *engine.Engine
	Tracker progress.Tracker
	Verbose bool
	// Ctx is the context that should be used within a command to carry deadlines and cancellation signals.
	Ctx context.Context
	// Logfile is the log file used by the logger to record verbose logs in case of an error.
	// It is here so we can close and clean it up properly.
	Logfile *os.File
}
