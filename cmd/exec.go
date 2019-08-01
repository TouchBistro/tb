package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
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
		err := deps.Resolve(deps.Docker)
		if err != nil {
			fatal.ExitErr(err, "Could not resolve dependencies.")
		}

		err = config.Clone(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		service := args[0]

		// Make sure it's a valid service
		if _, ok := config.Services()[service]; !ok {
			fatal.Exitf("%s is not a valid service\n. Try running `tb list` to see available services\n", service)
		}

		cmds := strings.Join(args[1:], " ")
		cmdStr := fmt.Sprintf("%s exec %s %s", docker.ComposeFile(), service, cmds)

		execCmd := exec.Command("docker-compose", strings.Fields(cmdStr)...)
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
