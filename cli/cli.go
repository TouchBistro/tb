// Package cli provides general functionality for all CLI commands.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/engine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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

// Container stores all the dependencies that can be used by commands.
type Container struct {
	Engine  *engine.Engine
	Tracker progress.Tracker
	Verbose bool
	// This is only here for cleanup purposes, don't use it directly,
	// use Tracker instead.
	Logger *Logger
}

// ExitError is used to signal that the CLI should exit with a given code and message.
type ExitError struct {
	Code    int
	Message string
	Err     error
}

func (e *ExitError) Error() string {
	return e.Message
}

// Logger that wraps a logrus.Logger and implements progress.OutputLogger.
// It writes to both stderr and a temp file.
type Logger struct {
	// Embed a logrus logger to automatically implement all the log methods.
	*logrus.Logger

	f *os.File    // temp file where all logs are written
	h *loggerHook // hook for also logging to stderr
}

// NewLogger creates a new Logger instance.
// The output logger will messages to both stderr and a temp file.
// If verbose is true debug logs will be excluded from stderr but will
// still be written to the temp file.
//
// Cleanup should be used to close and potentially remove the temp file
// once you are done with the logger. Filename can be used to retrieve
// the name of the tempfile.
func NewLogger(verbose bool) (*Logger, error) {
	// Create a temp file to log to.
	f, err := os.CreateTemp("", "tb_log_*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	logger := &logrus.Logger{
		Out: f,
		Formatter: &logrus.TextFormatter{
			DisableColors: true,
		},
		Hooks: make(logrus.LevelHooks),
		Level: logrus.DebugLevel,
	}
	h := &loggerHook{
		w:       os.Stderr,
		verbose: verbose,
		formatter: &logrus.TextFormatter{
			DisableTimestamp: true,
			ForceColors:      true,
		},
	}
	logger.AddHook(h)
	return &Logger{logger, f, h}, nil
}

func (l *Logger) WithFields(fields progress.Fields) progress.Logger {
	return progressLogger{l.Logger.WithFields(logrus.Fields(fields))}
}

func (l *Logger) Output() io.Writer {
	// Use the hook's output, not the actual logger's
	// since that is where stderr logs go.
	l.h.mu.Lock()
	defer l.h.mu.Unlock()
	return l.h.w
}

func (l *Logger) SetOutput(out io.Writer) {
	l.h.mu.Lock()
	defer l.h.mu.Unlock()
	l.h.w = out
}

// Filename returns the name of the file where logs are written.
func (l *Logger) Filename() string {
	if l == nil {
		return ""
	}
	return l.f.Name()
}

// Cleanup closes the log file and prevents further logging.
// If remove is true, the log file will also be removed.
func (l *Logger) Cleanup(remove bool) error {
	if l == nil {
		// no-op if no logger, so we can still call this method even
		// if logging wasn't initialized properly
		return nil
	}
	if err := l.f.Close(); err != nil {
		return err
	}
	if remove {
		if err := os.Remove(l.f.Name()); err != nil {
			return err
		}
	}
	return nil
}

// Logger is a simple wrapper for a logrus.FieldLogger that makes
// it implement progress.Logger.
type progressLogger struct {
	logrus.FieldLogger
}

func (pl progressLogger) WithFields(fields progress.Fields) progress.Logger {
	return progressLogger{pl.FieldLogger.WithFields(logrus.Fields(fields))}
}

// loggerHook is a logrus hook to writes to an io.Writer.
type loggerHook struct {
	w         io.Writer
	verbose   bool
	mu        sync.Mutex
	formatter logrus.Formatter
}

func (h *loggerHook) Levels() []logrus.Level {
	// We want the hook to fire on all levels and then we will decide what to do.
	return logrus.AllLevels
}

func (h *loggerHook) Fire(e *logrus.Entry) error {
	if e.Level == logrus.DebugLevel && !h.verbose {
		// Ignore debug level if we aren't verbose
		return nil
	}
	b, err := h.formatter.Format(e)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err = h.w.Write(b)
	return err
}
