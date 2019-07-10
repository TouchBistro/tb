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

	err = config.Init("./config.json", "./playlists.yml")
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	err = cmd.RootCmd.Execute()
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
}
