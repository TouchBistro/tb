package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/fatal"
	"github.com/spf13/cobra"
)

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
		err := config.CloneMissingRepos(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]

		// Make sure it's a valid service
		service, ok := config.Services()[serviceName]
		if !ok {
			fatal.Exitf("%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
		}

		composeCmd := fmt.Sprintf("%s exec %s", docker.ComposeFile(), config.ComposeName(serviceName, service))
		composeCmdArgs := strings.Split(composeCmd, " ")
		composeCmdArgs = append(composeCmdArgs, args[1:]...)

		execCmd := exec.Command("docker-compose", composeCmdArgs...)

		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		err := execCmd.Run()
		if err != nil {
			fatal.ExitErr(err, "Could not execute command against this service.")
		}
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
