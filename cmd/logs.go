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

var logsCmd = &cobra.Command{
	Use:   "logs [services...]",
	Short: "View logs from containers",
	PreRun: func(cmd *cobra.Command, args []string) {
		err := config.CloneMissingRepos(config.Services())
		if err != nil {
			fatal.ExitErr(err, "failed cloning git repos.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var services string

		if len(args) > 0 {
			var b strings.Builder
			for _, serviceName := range args {
				// Make sure it's a valid service
				s, ok := config.Services()[serviceName]
				if !ok {
					fatal.Exitf("%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
				}

				b.WriteString(config.ComposeName(serviceName, s))
				b.WriteString(" ")
			}

			services = b.String()
		}

		cmdStr := fmt.Sprintf("%s logs -t %s", docker.ComposeFile(), services)
		execCmd := exec.Command("docker-compose", strings.Fields(cmdStr)...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		err := execCmd.Run()
		if err != nil {
			fatal.ExitErr(err, "Could not view logs.")
		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
