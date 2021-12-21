package cmd

import (
	"os"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/util"
	"github.com/spf13/cobra"
)

type execOptions struct {
	shouldSkipGitPull bool
}

var execOpts execOptions

var execCmd = &cobra.Command{
	Use:   "exec <service-name> <command> [additional-commands...]",
	Short: "executes a command in a service container",
	Long: `Executes a command in a service container.

Examples:
- run yarn db:prepare:test in the core-database container.
	tb exec core-database yarn db:prepare:test

- start an interactive shell in the core-database container.
	tb exec core-database bash`,
	Args: cobra.MinimumNArgs(2),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := config.CloneOrPullRepos(!execOpts.shouldSkipGitPull)
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]

		// Make sure it's a valid service
		s, err := config.LoadedServices().Get(serviceName)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
		}

		c := command.New(command.WithStdin(os.Stdin), command.WithStdout(os.Stdout), command.WithStderr(os.Stderr))
		err = docker.ComposeExec(util.DockerName(s.FullName()), args[1:], c)
		if err != nil {
			fatal.ExitErr(err, "Could not execute command against this service.")
		}
	},
}

func init() {
	execCmd.Flags().BoolVar(&execOpts.shouldSkipGitPull, "no-git-pull", false, "dont update git repositories")
}
