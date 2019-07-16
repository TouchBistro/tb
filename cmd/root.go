package cmd

import (
	"github.com/TouchBistro/tb/config"
	_ "github.com/TouchBistro/tb/release"
	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "tb",
	Version: "0.0.7", // TODO: Fix this hardcoded bullshit
	Short:   "tb is a CLI for running TouchBistro services on a development machine",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		util.FatalErr(err, "Failed executing command.")
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	err := config.InitRC()
	if err != nil {
		util.FatalErr(err, "Failed to initialise .tbrc file.")
	}

	logLevel, err := log.ParseLevel(config.TBRC().LogLevel)
	if err != nil {
		util.FatalErr(err, "Failed to initialise logger level.")
	}

	log.SetLevel(logLevel)

	// TODO: Make this its own setting or make the format less intense.
	log.SetReportCaller(logLevel == log.DebugLevel)

	err = config.Init()
	if err != nil {
		util.FatalErr(err, "Failed to initialise config files.")
	}
}

func Root() *cobra.Command {
	return rootCmd
}
