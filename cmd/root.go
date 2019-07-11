package cmd

import (
	"github.com/TouchBistro/tb/config"
	"github.com/gobuffalo/packr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tb",
	Short: "tb is a CLI for running TouchBistro services on a development machine",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	err := config.InitRC()
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to initialise .tbrc file")
	}

	logLevel, err := log.ParseLevel(config.TBRC().LogLevel)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to initialise logger level")
	}

	log.SetLevel(logLevel)
	log.SetReportCaller(logLevel == log.DebugLevel)

	err = config.Init("./services.yml", "./playlists.yml")
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to initialise config files")
	}
}
