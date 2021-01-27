package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs <service-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Opens link to API docs for a given service (Experimental)",
	Long: `Opens link to API docs for a given service. (Experimental)

Example:
  tb docs core`,
	Run: func(cmd *cobra.Command, args []string) {
		// Experimental only until multiple docs URLs supported for one service
		if !config.IsExperimentalEnabled() {
			fatal.Exit("You need to enable experimental mode to use this feature")
		}

		serviceName := args[0]
		service, err := config.LoadedServices().Get(serviceName)
		if err != nil {
			fatal.ExitErrf(err, "%s is not a valid service.\nTry running `tb list` to see available services\n", serviceName)
		}

		if !service.HasGitRepo() {
			fatal.Exitf("%s does not have a repo or is a third-party repo\n", serviceName)
		}

		url, err := getDocsURL(service.DockerName())
		if err != nil {
			fatal.ExitErrf(err, "could not find docs url for %s\n", serviceName)
		}

		log.Infof("Opening docs for %s...\n", serviceName)

		// `open` command is macOS only
		openCmd := "open"
		if util.IsLinux() {
			// `xdg-open` is linux equivalent
			openCmd = "xdg-open"
		}

		w := log.WithField("id", "docs-open").WriterLevel(log.DebugLevel)
		defer w.Close()
		err = command.New(command.WithStdout(w), command.WithStderr(w)).Exec(openCmd, url)
		if err != nil {
			fatal.ExitErrf(err, "failed to open docs at %s\n", url)
		}
	},
}

func getDocsURL(dockerName string) (string, error) {
	required := []string{"DOCS_URL"}
	missing := "missing"

	// This is ugly, but less ugly than using printenv and much faster than doing individual execs for every var
	// generates a command in the following format: sh -c echo ${var1:-missing} ${var2:-missing} ...${varN:-missing}
	// missing is used as a blank value instead of an empty string to make producing nicer errors to the user much easier.
	var sb strings.Builder
	sb.WriteString("echo")
	for _, req := range required {
		sb.WriteString(fmt.Sprintf(" ${%s:-%s}", req, missing))
	}
	args := []string{"sh", "-c", sb.String()}

	buf := &bytes.Buffer{}
	cmd := command.New(command.WithStdin(os.Stdin), command.WithStdout(buf), command.WithStderr(os.Stderr))
	err := docker.ComposeExec(dockerName, args, cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed execing command inside this service's container.")
	}

	url := strings.TrimSpace(buf.String())
	if url == missing {
		return "", errors.Errorf("DOCS_URL environment variable not found")
	}

	return url, nil
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
