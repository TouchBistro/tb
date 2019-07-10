package main

import (
	"os"

	"github.com/TouchBistro/tb/cmd"
	"github.com/TouchBistro/tb/config"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := config.InitRC()
	if err != nil {
		log.Fatal(err.Error())
	}

	logLevel, err := log.ParseLevel(config.TBRC().LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(logLevel)
	log.SetReportCaller(logLevel == log.DebugLevel)

	err = config.Init("./services.yml", "./playlists.yml")
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to initialise config files")
	}

	err = cmd.RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
