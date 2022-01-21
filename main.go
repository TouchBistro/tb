package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

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

	// Close log file and remove it if there was no error.
	// Ignore the error since there is nothing we can do about it.
	// Also, temp files are automatically cleaned up on reboots
	// so it's not a big deal.
	_ = c.Logger.Cleanup(err == nil)
	if err == nil {
		return
	}

	// Handle error
	var fatalErr *fatal.Error
	switch {
	case errors.As(err, &fatalErr):
		// Nothing to do, since fatalErr is now populated
	case errors.Is(err, context.Canceled):
		fatalErr = &fatal.Error{
			Code: 130,
			Msg:  "\nOperation cancelled",
		}
	case
		errors.Is(err, resource.ErrNotFound):
		fatalErr = &fatal.Error{
			Msg: "Try running `tb list` to see available services",
			Err: err,
		}
	default:
		// TODO(@cszatmary): We can check if errors.Error and use the Kind
		// to add custom messages to try and help the user.
		// We should also add sepecific error codes based on Kind.
		fatalErr = &fatal.Error{Err: err}
	}

	// Print the error ourselves instead of letting fatal do it since we want
	// to do some additional stuff after before exiting.
	if fatalErr.Msg != "" || fatalErr.Err != nil {
		if c.Verbose {
			fmt.Fprintf(os.Stderr, "%+v\n", fatalErr)
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", fatalErr)
		}
	}

	// Tell user where log file is for further troubleshooting.
	if name := c.Logger.Filename(); name != "" {
		fmt.Fprintf(os.Stderr, "\n👉 Logs are available at: %s 👈\n", name)
	}
	fatal.Exit(fatalErr)
}
