package cmd

import (
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version string

var rootCmd = &cobra.Command{
	Use:     "tb",
	Version: version,
	Short:   "tb is a CLI for running TouchBistro services on a development machine",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fatal.ExitErr(err, "Failed executing command.")
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	err := config.InitRC()
	if err != nil {
		fatal.ExitErr(err, "Failed to initialise .tbrc file.")
	}

	configLevel := config.TBRC().LogLevel

	logLevel, err := log.ParseLevel(configLevel)
	if err != nil {
		fatal.ExitErr(err, "Failed to initialise logger level.")
	}

	log.SetLevel(logLevel)
	log.SetFormatter(&log.TextFormatter{
		// TODO: Remove the log level - its quite ugly
		DisableTimestamp: true,
	})

	if configLevel != "debug" {
		fatal.ShowStackTraces = false
	}

	err = config.Init()
	if err != nil {
		fatal.ExitErr(err, "Failed to initialise config files.")
	}
}

func Root() *cobra.Command {
	return rootCmd
}
