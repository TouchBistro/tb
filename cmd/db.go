package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type dbConfig struct {
	dbType   string
	name     string
	port     string
	user     string
	password string
}

func getDbConf(serviceName string) (dbConfig, error) {
	required := []string{"DB_TYPE", "DB_NAME", "DB_PORT", "DB_USER", "DB_PASSWORD"}
	missing := "missing"

	// This is ugly, but less ugly than using printenv and much faster than doing individual execs for every var
	// generates a command in the following format: sh -c echo ${var1:-missing} ${var2:-missing} ...${varN:-missing}
	// mssing is used as a blank value instead of an empty string to make producing nicer errors to the user much easier.
	var sb strings.Builder
	sb.WriteString("echo")
	for _, req := range required {
		sb.WriteString(fmt.Sprintf(" ${%s:-%s}", req, missing))
	}
	args := []string{"sh", "-c", sb.String()}

	buf := &bytes.Buffer{}
	cmd := command.New(command.WithStdin(os.Stdin), command.WithStdout(buf), command.WithStderr(os.Stderr))
	err := docker.ComposeExec(serviceName, args, cmd)
	if err != nil {
		return dbConfig{}, errors.Wrap(err, "failed execing command inside this service's container.")
	}

	result := strings.Split(strings.TrimSpace(buf.String()), " ")

	// Validate that all required env vars were found.
	notFound := make([]string, 0)
	for i, res := range result {
		if res == missing {
			notFound = append(notFound, required[i])
		}
	}
	if len(notFound) != 0 {
		return dbConfig{}, errors.Errorf("The following required env vars were not defined: [%s]", strings.Join(notFound, ", "))
	}

	conf := dbConfig{
		dbType:   result[0],
		name:     result[1],
		port:     result[2],
		user:     result[3],
		password: result[4],
	}

	return conf, nil
}

var dbCmd = &cobra.Command{
	Use:        "db <service-name>",
	Deprecated: "it will be removed soon",
	Short:      "Connects to the database in a service",
	Long: `Connects to the database in a service using a cli database client.

Examples:
- Connect to the partners-config-service's database.
	tb db partners-config-service`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		s, err := config.LoadedServices().Get(serviceName)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a valid service\n. Try running `tb list` to see available services\n", serviceName)
		}

		log.Info("checking required env vars.")

		dbConf, err := getDbConf(util.DockerName(s.FullName()))
		if err != nil {
			fatal.ExitErr(err, "Could not retrieve database config for this service.")
		}

		var cli string
		var connArg string

		switch dbConf.dbType {
		case "postgresql":
			cli = deps.Pgcli
			connArg = fmt.Sprintf("%s://%s:%s@localhost:%s/%s", dbConf.dbType, dbConf.user, dbConf.password, dbConf.port, dbConf.name)
		case "mysql":
			cli = deps.Mycli
			connArg = fmt.Sprintf("%s://%s:%s@localhost:%s/%s", dbConf.dbType, dbConf.user, dbConf.password, dbConf.port, dbConf.name)
		case "mssql":
			cli = deps.Mssqlcli
			connArg = fmt.Sprintf("-U %s -P %s -S localhost -d %s", dbConf.user, dbConf.password, dbConf.name)
			fmt.Println(connArg)
		default:
			fatal.Exitf("DB_TYPE %s is not currently supported by tb db. Please consider making a pull request or let the maintainers know about your use case.", dbConf.dbType)
		}

		if !command.IsAvailable(cli) {
			shouldInstallCli := util.Prompt(fmt.Sprintf("This command requires %s. Would you like tb to install it for you? y/n\n> ", cli))
			if !shouldInstallCli {
				fatal.Exitf("This command requires %s for %s, which uses a %s database.\n Consider installing it yourself or letting tb install it for you.", cli, serviceName, dbConf.dbType)
			}
		}

		err = deps.Resolve(cli)
		if err != nil {
			fatal.ExitErrf(err, "could not install %s", cli)
		}

		log.Infof("starting %s...", cli)

		err = command.New(command.WithStdin(os.Stdin), command.WithStdout(os.Stdout), command.WithStderr(os.Stderr)).
			Exec(cli, strings.Fields(connArg)...)
		if err != nil {
			fatal.ExitErrf(err, "could not start database client %s", cli)
		}
	},
}

func init() {
	rootCmd.AddCommand(dbCmd)
}
