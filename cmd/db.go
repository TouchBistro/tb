package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/deps"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var host, user string
var port int

var dbCmd = &cobra.Command{
	Use:   "db <db-name>",
	Short: "Connects to the database in a service",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := deps.Resolve(deps.Pgcli)
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to resolve dependencies.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Starting pgcli client.")

		dbName := args[0]
		cmdStr := fmt.Sprintf("-h %s -p %d -U %s %s", host, port, user, dbName)

		execCmd := exec.Command("pgcli", strings.Fields(cmdStr)...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		err := execCmd.Run()

		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to start database client.")
		}

	},
}

func init() {
	RootCmd.AddCommand(dbCmd)
	dbCmd.Flags().StringVarP(&host, "host", "H", "localhost", "host address of the database")
	dbCmd.Flags().IntVarP(&port, "port", "p", 5432, "port that the database is listening on")
	dbCmd.Flags().StringVarP(&user, "user", "u", "core", "user name to connect to the database")
}
