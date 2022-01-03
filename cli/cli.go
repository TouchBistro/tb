// Package cli provides general functionality for all CLI commands.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/engine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Container stores all the dependencies that can be used by commands.
type Container struct {
	Engine  *engine.Engine
	Tracker progress.Tracker
	Verbose bool
}

// ExitError is used to signal that the CLI should exit with a given
// code and message.
type ExitError struct {
	Code    int
	Message string
	Err     error
}

func (e *ExitError) Error() string {
	return e.Message
}

// OutputLogger implements progress.OutputLogger using logrus.
type OutputLogger struct {
	*logrus.Logger
}

func (ol OutputLogger) WithFields(fields progress.Fields) progress.Logger {
	return Logger{ol.Logger.WithFields(logrus.Fields(fields))}
}

func (ol OutputLogger) Output() io.Writer {
	return ol.Logger.Out
}

// Logger implements progress.Logger using logrus.
type Logger struct {
	logrus.FieldLogger
}

func (l Logger) WithFields(fields progress.Fields) progress.Logger {
	return Logger{l.FieldLogger.WithFields(logrus.Fields(fields))}
}

// Prompt prompts the user for the answer to a yes/no question.
func Prompt(msg string) bool {
	// check for yes and assume no on any other input to avoid annoyance
	fmt.Print(msg)
	var resp string
	_, err := fmt.Scanln(&resp)
	if err != nil {
		return false
	}
	if strings.ToLower(string(resp[0])) == "y" {
		return true
	}
	return false
}

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
