package cmd

import (
	"fmt"
	"strings"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/deps"
	"github.com/TouchBistro/tb/docker"
	"github.com/TouchBistro/tb/fatal"
	"github.com/TouchBistro/tb/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type envoptions struct {
	cliServiceNames []string
	playlistName    string
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Launches an ephemeral environment from a playlist name or as a comma separated list of services. (ECR only)",
	Long: `Launches an ephemeral environment from a playlist name or as a comma separated list of services.

Examples:
- launch the services defined under the "core" key in playlists.yml
	tb env --playlist core

- launch only postgres and localstack
	tb env --services postgres,localstack`,
	Args: cobra.NoArgs,
	PreRun: func(cmd *cobra.Command, args []string) {

		if len(opts.cliServiceNames) > 0 && opts.playlistName != "" {
			fatal.Exit("you can only specify one of --playlist or --services.\nTry tb up --help for some examples.")
		}

		var err error

		selectedServices, err = config.SelectServices(opts.cliServiceNames, opts.playlistName)
		if err != nil {
			fatal.ExitErr(err, "problem with requested service configuration")
		}

		composeNames := config.ComposeNames(selectedServices)
		log.Infof("running the following services: %s", strings.Join(composeNames, ", "))

		for service := range selectedServices {
			if !selectedServices[service].ECR {
				if (service != "localstack") && (service != "postgres") {
					fatal.Exit("Non-ECR service configured, fix overrides to only use ECR services and try again")
				}
			}
		}

		err = deps.Resolve(
			deps.Brew,
			deps.Aws,
		)
		if err != nil {
			fatal.ExitErr(err, "could not resolve dependencies")
		}
		fmt.Println()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		composeFile = docker.ComposeFile()

		serviceNames := config.ComposeNames(selectedServices)

		log.Info("☐ starting docker-compose up in detached mode")

		upArgs := fmt.Sprintf("inferno.py %s %s", composeFile, strings.Join(serviceNames, " "))
		err = util.Exec("inferno-python", "python", strings.Fields(upArgs)...)
		if err != nil {
			fatal.ExitErr(err, "could not run inferno up")
		}

		log.Info("☑ finished starting environment")
		fmt.Println()
	},
}

func init() {
	envCmd.PersistentFlags().StringVarP(&opts.playlistName, "playlist", "p", "", "the name of a service playlist")
	envCmd.PersistentFlags().StringSliceVarP(&opts.cliServiceNames, "services", "s", []string{}, "comma separated list of services to start. eg --services postgres,localstack.")

	rootCmd.AddCommand(envCmd)
}
