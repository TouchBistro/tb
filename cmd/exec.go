package cmd

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <service-name> <command> [additional-commands...]",
	Short: "executes a command in a service container",
	Args:  cobra.MinimumNArgs(2),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := deps.Resolve()
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

		_, err = util.Exec("docker-compose", strings.Fields(cmdStr)...)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(execCmd)
}
