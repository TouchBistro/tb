package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/deps"
	"github.com/spf13/cobra"
)

var host, user, dbType string
var port int

var dbCmd = &cobra.Command{
	Use:   "db <db-name>",
	Short: "Connects to the database in a service",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// TODO only check for pgcli and mssql-cli
		err := deps.Resolve()
		if err != nil {
			log.Fatal(err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var cli, cmdStr string
		dbName := args[0]

		if dbType == "postgres" {
			cli = "pgcli"
			cmdStr = fmt.Sprintf("-h %s -p %d -U %s %s", host, port, user, dbName)
		} else if dbType == "mssql" {
			cli = "mssql-cli"
			cmdStr = fmt.Sprintf("-S '%s,%d' -U %s -d %s", host, port, user, dbName)
		} else {
			log.Fatal("Unknown database type ", dbType)
		}

		execCmd := exec.Command(cli, strings.Fields(cmdStr)...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		err := execCmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(dbCmd)
	dbCmd.Flags().StringVarP(&host, "host", "H", "localhost", "host address of the database")
	dbCmd.Flags().IntVarP(&port, "port", "p", 5432, "port that the database is listening on")
	dbCmd.Flags().StringVarP(&user, "user", "u", "core", "user name to connect to the database")
	dbCmd.Flags().StringVarP(&dbType, "type", "t", "postgres", "the type of database, postgres or mssql")
}
