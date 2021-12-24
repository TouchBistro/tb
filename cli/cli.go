// Package cli provides general functionality for all CLI commands.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/engine"
	"github.com/sirupsen/logrus"
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
