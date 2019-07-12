package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gobuffalo/packr/v2"

	"github.com/TouchBistro/tb/util"
	log "github.com/sirupsen/logrus"
)

var services map[string]Service
var playlists map[string]Playlist
var tbRoot string

const (
	servicesPath             = "services.yml"
	playlistPath             = "playlists.yml"
	dockerComposePath        = "docker-compose.yml"
	localstackEntrypointPath = "localstack-entrypoint.sh"
)

type Service struct {
	IsGithubRepo bool   `yaml:"repo"`
	Migrations   bool   `yaml:"migrations"`
	ECR          bool   `yaml:"ecr"`
	ImageURI     string `yaml:"imageURI"`
}

func setupEnv() {
	// Set $TB_ROOT so it works in the docker-compose file
	tbRoot = fmt.Sprintf("%s/.tb", os.Getenv("HOME"))
	os.Setenv("TB_ROOT", tbRoot)

	// Create $TB_ROOT directory if it doesn't exist
	if !util.FileOrDirExists(tbRoot) {
		os.Mkdir(tbRoot, 0755)
	}
}

func dumpFile(name string, box *packr.Box) error {
	path := fmt.Sprintf("%s/%s", tbRoot, name)

	if util.FileOrDirExists(path) {
		log.Debugf("%s exists", path)
		return nil
	}

	log.Debugf("%s does not exist, creating file...", path)
	buf, err := box.Find(name)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, buf, 0644)
}

func TBRootPath() string {
	return tbRoot
}

func Init() error {
	setupEnv()

	box := packr.New("static", "../static")

	sBuf, err := box.Find(servicesPath)
	if err != nil {
		return err
	}

	err = util.DecodeYaml(bytes.NewReader(sBuf), &services)
	if err != nil {
		return err
	}

	pBuf, err := box.Find(playlistPath)
	if err != nil {
		return err
	}
	err = util.DecodeYaml(bytes.NewReader(pBuf), &playlists)
	if err != nil {
		return err
	}

	err = dumpFile(dockerComposePath, box)
	if err != nil {
		return err
	}

	err = dumpFile(localstackEntrypointPath, box)
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
