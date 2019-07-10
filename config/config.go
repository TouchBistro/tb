package config

import (
	"fmt"

	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
)

var services map[string]Service
var playlists map[string]Playlist

type Service struct {
	IsGithubRepo bool   `yaml:"repo"`
	Migrations   bool   `yaml:"migrations"`
	ECR          bool   `yaml:"ecr"`
	ImageURI     string `yaml:"imageURI"`
}

func Init(confPath, playlistPath string) error {
	err := util.ReadYaml(confPath, &services)
	if err != nil {
		return err
	}

	err = util.ReadYaml(playlistPath, &playlists)
	if err != nil {
		return err
	}

	return nil
}

func Services() map[string]Service {
	return services
}

func BaseImages() []string {
	return []string{
		"touchbistro/alpine-node:10-build",
		"touchbistro/alpine-node:10-runtime",
	}
}

func GetPlaylist(name string) []string {
	// TODO: Make this less yolo if Init() wasn't called
	if playlists == nil {
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
	} else if playlist, ok := playlists[name]; ok {
		if playlist.Extends != "" {
			parentPlaylist := GetPlaylist(playlist.Extends)
			return append(parentPlaylist, playlist.Services...)
		}

		return playlist.Services
	}

	// TODO: Fallback to user-custom choice somehow
	log.Fatal(fmt.Sprintf("Playlist %s does not exist", name))
	return []string{}
}
