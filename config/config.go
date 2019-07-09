package config

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
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

	err = loadPlaylists(playlistPath)
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

func loadPlaylists(path string) error {
	var err error

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := yaml.NewDecoder(file)
	err = dec.Decode(&playlists)

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

	if names, ok := list[name]; ok {
		return names
	}

	// TODO: Fallback to user-custom choice somehow
	log.Fatal(fmt.Sprintf("Playlist %s does not exist", name))
	return []string{}
}
