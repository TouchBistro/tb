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

	// Listen oforSIGINT to do a graceful abort
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
	if err == nil {
		return
	}

	// Handle error
	var exitErr *cli.ExitError
	switch {
	case errors.Is(err, context.Canceled):
		exitErr = &cli.ExitError{
			Code:    130,
			Message: "\nOperation cancelled",
		}
	case errors.As(err, &exitErr):
		// Nothing to do, since exitErr is now populated
	default:
		// TODO(@cszatmary): We can check if errors.Error and use the Kind
		// to add custom messages to try and help the user.
		exitErr = &cli.ExitError{Err: err}
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
	os.Exit(exitErr.Code)
}
