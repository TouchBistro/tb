package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	surveyterm "github.com/AlecAivazis/survey/v2/terminal"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/cli/commands"
	"github.com/TouchBistro/tb/resource"
)

// Set by goreleaser when release build is created.
var version = "dev"

func main() {
	// Listen for SIGINT to do a graceful abort
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt)
	go func() {
		<-abort
		cancel()
	}()

	var c cli.Container
	rootCmd := commands.NewRootCommand(&c, version)
	err := rootCmd.ExecuteContext(ctx)
	fatalErr, removeLogfile := mapError(ctx, err)

	// Close log file and remove it if there was no error.
	// Ignore the error since there is nothing we can do about it.
	// Also, temp files are automatically cleaned up on reboots
	// so it's not a big deal.
	_ = c.Logger.Cleanup(removeLogfile)
	if err == nil {
		return
	}

	exiter := &fatal.Exiter{
		PrintDetailed: c.Verbose,
		ExitFunc: func(code int) {
			// Tell user where log file is for further troubleshooting.
			// Skip if code is 130 since that means the user aborted the
			// program with control-C.
			if name := c.Logger.Filename(); name != "" && code != 130 {
				fmt.Fprintf(os.Stderr, "\n👉 Logs are available at: %s 👈\n", name)
			}
			os.Exit(code)
		},
	}
	exiter.PrintAndExit(fatalErr)
}

func mapError(ctx context.Context, err error) (fatalErr *fatal.Error, removeLogfile bool) {
	switch {
	case err == nil:
		return nil, true
	// Handle special error cases first
	case errors.Is(err, context.Canceled) || errors.Is(err, surveyterm.InterruptErr):
		return &fatal.Error{Code: 130, Msg: "\nOperation cancelled"}, true
	// See if the context was cancelled. This is important for subprocesses created by os/exec
	// since exit codes don't propagate if the subprocess was terminated with a signal.
	case errors.Is(ctx.Err(), context.Canceled):
		return &fatal.Error{Code: 130}, true
	// See if already a fatal.Error
	case errors.As(err, &fatalErr):
	default:
		return &fatal.Error{Err: err}, false
	}
	// If no underlying error, just return as is.
	if fatalErr.Err == nil {
		return
	}

	// Check for specific errors and improve the message for a better user experience.
	// TODO(@cszatmary): We can check if errors.Error and use the Kind
	// to add custom messages to try and help the user.
	// We should also add specific error codes based on Kind.
	switch {
	case errors.Is(fatalErr.Err, resource.ErrNotFound):
		// TODO(@cszatmary): Should we have a custom exit code?
		fatalErr.Msg = "Try running `tb list` to see available services"
	}
	return
}
