package main

import (
	"github.com/TouchBistro/tb/cmd"
	"github.com/TouchBistro/tb/config"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	log.SetLevel(log.DebugLevel) // TODO: Make this configurable in viper.

	// TODO: This is ugly and supposedly slow - we should only enable it if the user wants verbose logging on.
	// log.SetReportCaller(true)

	err := config.Init("./config.json", "./playlists.yml")
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to initialise config files")
	}

	err = cmd.RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
