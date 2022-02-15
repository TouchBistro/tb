// Package cli provides general functionality for all CLI commands.
package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/TouchBistro/goutils/log"
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
	// This is only here for cleanup purposes, don't use it directly, use Tracker instead.
	Logger *Logger
}

// Logger is a logger that writes to both stderr and a temp file.
type Logger struct {
	// Embed a logger to automatically implement all the log methods.
	*log.Logger

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

	logger := log.New(
		log.WithOutput(f),
		log.WithFormatter(&log.TextFormatter{}),
		log.WithLevel(log.LevelDebug),
	)
	h := &loggerHook{
		w:       os.Stderr,
		verbose: verbose,
		formatter: &log.TextFormatter{
			Pretty:           true,
			DisableTimestamp: true,
		},
	}
	logger.AddHook(h)
	return &Logger{logger, f, h}, nil
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

// loggerHook is a logger hook to writes to an io.Writer.
type loggerHook struct {
	w         io.Writer
	verbose   bool
	mu        sync.Mutex
	formatter log.Formatter
	buf       bytes.Buffer
}

func (h *loggerHook) Run(e *log.Entry) error {
	if e.Level == log.LevelDebug && !h.verbose {
		// Ignore debug level if we aren't verbose
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf.Reset()
	b, err := h.formatter.Format(e, &h.buf)
	if err != nil {
		return err
	}
	_, err = h.w.Write(b)
	return err
}
