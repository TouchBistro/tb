package main

import (
	"os"

	"github.com/TouchBistro/tb/cmd"
	"github.com/TouchBistro/tb/config"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := config.Init("./config.json", "./playlists.yml")
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
