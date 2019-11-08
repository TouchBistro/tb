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

var stopCmd = &cobra.Command{
	Use:   "stop [services...]",
	Short: "Stop running containers",
	PreRun: func(cmd *cobra.Command, args []string) {
		err := config.CloneMissingRepos(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var services string

		services, err := config.ValidateServiceList(args)
		if err != nil {
			fatal.ExitErr(err, "failed stopping containers.")
		}

		cmdStr := fmt.Sprintf("%s stop %s", docker.ComposeFile(), services)
		execCmd := exec.Command("docker-compose", strings.Fields(cmdStr)...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		err = execCmd.Run()
		if err != nil {
			fatal.ExitErr(err, "Could not stop containers.")
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
