package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"

	"github.com/TouchBistro/tb/cli"
	"github.com/TouchBistro/tb/cli/commands"
	"github.com/TouchBistro/tb/resource"
)

// Set by goreleaser when release build is created.
var version string

func main() {
	// Set version if built from source
	if version == "" {
		version = "source"
		if info, available := debug.ReadBuildInfo(); available {
			version = info.Main.Version
		}
	}

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

	// Close log file and remove it if there was no error
	c.Logger.Cleanup(err == nil)
	if err == nil {
		return
	}

	// Handle error
	var exitErr *cli.ExitError
	switch {
	case errors.As(err, &exitErr):
		// Nothing to do, since exitErr is now populated
	case errors.Is(err, context.Canceled):
		exitErr = &cli.ExitError{
			Code:    130,
			Message: "\nOperation cancelled",
		}
	case
		errors.Is(err, resource.ErrNotFound):
		exitErr = &cli.ExitError{
			Message: "Try running `tb list` to see available services",
			Err:     err,
		}
	default:
		// TODO(@cszatmary): We can check if errors.Error and use the Kind
		// to add custom messages to try and help the user.
		// We should also add sepecific error codes based on Kind.
		exitErr = &cli.ExitError{Err: err}
	}
	// Make sure a valid exit code was set, if not just default to 1
	// since that's the general catch all error code.
	if exitErr.Code <= 0 {
		exitErr.Code = 1
	}

	// Print out the error and message then exit
	if exitErr.Err != nil {
		if c.Verbose {
			fmt.Fprintf(os.Stderr, "Error: %+v\n", exitErr.Err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", exitErr.Err)
		}
	}
	// If an error was just printed and a message is going to be printed,
	// add an extra newline inbetween them
	if exitErr.Err != nil && exitErr.Message != "" {
		fmt.Fprintln(os.Stderr)
	}
	if exitErr.Message != "" {
		fmt.Fprint(os.Stderr, exitErr.Message)
		if !strings.HasSuffix(exitErr.Message, "\n") {
			fmt.Fprintln(os.Stderr)
		}
	}

	// Tell user where log file is for further troubleshooting.
	if name := c.Logger.Filename(); name != "" {
		fmt.Fprintf(os.Stderr, "\n👉 Logs are available at: %s 👈\n", name)
	}
	os.Exit(exitErr.Code)
}
