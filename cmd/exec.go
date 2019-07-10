package cmd

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

var execCmd = &cobra.Command{
	Use:   "exec <service-name> <command> [additional-commands...]",
	Short: "executes a command in a service container",
	Args:  cobra.MinimumNArgs(2),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := deps.Resolve(deps.Docker)
		if err != nil {
			log.Fatal(err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		files, err := docker.ComposeFiles()
		if err != nil {
			log.Fatal(err)
		}

		service := args[0]
		cmds := strings.Join(args[1:], " ")
		cmdStr := fmt.Sprintf("%s exec %s %s", files, service, cmds)

		execCmd := exec.Command("docker-compose", strings.Fields(cmdStr)...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		err = execCmd.Run()
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to run exec command.")
		}
	},
}

func init() {
	RootCmd.AddCommand(execCmd)
}
