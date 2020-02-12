package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/goutils/fatal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var host, user string
var port int

var dbCmd = &cobra.Command{
	Use:   "db <db-name>",
	Short: "Connects to the database in a service",
	Long: `Connects to the database in a service using pgcli.

Examples:
- Connect to the partners-config-service database.
	tb db core_db_dev`,
	Args: cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := deps.Resolve(deps.Pgcli)
		if err != nil {
			fatal.ExitErr(err, "could not resolve dependencies.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("starting pgcli client.")

		dbName := args[0]
		cmdStr := fmt.Sprintf("-h %s -p %d -U %s %s", host, port, user, dbName)

		execCmd := exec.Command("pgcli", strings.Fields(cmdStr)...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		err := execCmd.Run()

		if err != nil {
			fatal.ExitErr(err, "could not start database client.")
		}

	},
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.Flags().StringVarP(&host, "host", "H", "localhost", "host address of the database")
	dbCmd.Flags().IntVarP(&port, "port", "p", 5432, "port that the database is listening on")
	dbCmd.Flags().StringVarP(&user, "user", "u", "core", "user name to connect to the database")
}
