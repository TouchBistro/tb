package cmd

import (
	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/fatal"
	"github.com/aybabtme/logzalgo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
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

	var logLevel log.Level
	if config.TBRC().DebugEnabled {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
	}

	log.SetLevel(logLevel)
	log.SetFormatter(&log.TextFormatter{
		// TODO: Remove the log level - its quite ugly
		DisableTimestamp: true,
	})

	now := time.Now()
	if now.Month() == 10 && now.Day() == 31 {
		log.SetFormatter(logzalgo.NewZalgoFormatterrrrrr())
	}

	if logLevel != log.DebugLevel {
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
