package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
)

var config *[]Service
var playlists *map[string][]string

type Service struct {
	Name         string `json:"name"`
	IsGithubRepo bool   `json:"repo"`
	Migrations   bool   `json:"migrations"`
	ECR          bool   `json:"ecr"`
	ImageURI     string `json:"imageURI"`
}

func Init(confPath, playlistPath string) error {
	err := loadConfig(confPath)
	if err != nil {
		return err
	}

	err = util.ReadYaml(playlistPath, &playlists)
	if err != nil {
		return err
	}

	return nil
}

func loadConfig(path string) error {
	var err error

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(&config)
	return err
}

func All() *[]Service {
	return config
}

func BaseImages() []string {
	return []string{
		"touchbistro/alpine-node:10-build",
		"touchbistro/alpine-node:10-runtime",
	}
}

func GetPlaylist(name string) []string {
	// TODO: Make this less yolo if Init() wasn't called
	list := *playlists
	if list == nil {
		log.Panic("this is a bug. playlists is not initialised")
	}
	customList := tbrc.Playlists

	// Check custom playlists first
	if playlist, ok := customList[name]; ok {
		// TODO: Circular extends can make this infinite loop
		// Make sure people don't do that
		// Resolve parent playlist defined in extends
		if playlist.Extends != "" {
			parentPlaylist := GetPlaylist(playlist.Extends)
			return append(parentPlaylist, playlist.Services...)
		}

		return playlist.Services
	} else if names, ok := list[name]; ok {
		return names
	}

	// TODO: Fallback to user-custom choice somehow
	log.Fatal(fmt.Sprintf("Playlist %s does not exist", name))
	return []string{}
}
