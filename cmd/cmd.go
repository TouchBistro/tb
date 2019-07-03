package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/util"
	"github.com/spf13/cobra"
)

// wack but I'm just copying core-devtools for now
// should probably rename this to something less wack later
var cmdCmd = &cobra.Command{
	Use:   "cmd <service-name> <command> [additional-commands...]",
	Short: "executes a command in a service container",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		files, err := util.ComposeFiles()

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
			log.Fatal(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(cmdCmd)
}
