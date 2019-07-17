package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gobuffalo/packr/v2"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var services ServiceMap
var playlists map[string]Playlist
var tbRoot string

const (
	servicesPath             = "services.yml"
	playlistPath             = "playlists.yml"
	dockerComposePath        = "docker-compose.yml"
	localstackEntrypointPath = "localstack-entrypoint.sh"
	ecrURIRoot               = "651264383976.dkr.ecr.us-east-1.amazonaws.com"
)

func setupEnv() error {
	// Set $TB_ROOT so it works in the docker-compose file
	tbRoot = fmt.Sprintf("%s/.tb", os.Getenv("HOME"))
	os.Setenv("TB_ROOT", tbRoot)

	// Create $TB_ROOT directory if it doesn't exist
	if !util.FileOrDirExists(tbRoot) {
		err := os.Mkdir(tbRoot, 0755)
		if err != nil {
			return errors.Wrapf(err, "failed to create $TB_ROOT directory at %s", tbRoot)
		}
	}
	return nil
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
		return errors.Wrapf(err, "failed to find packr box %s", name)
	}

	err = ioutil.WriteFile(path, buf, 0644)
	return errors.Wrapf(err, "failed to write contents of %s to %s", name, path)
}

func TBRootPath() string {
	return tbRoot
}

func Init() error {
	err := setupEnv()
	if err != nil {
		return errors.Wrap(err, "failed to setup $TB_ROOT env")
	}

	box := packr.New("static", "../static")

	sBuf, err := box.Find(servicesPath)
	if err != nil {
		return errors.Wrapf(err, "failed to find packr box %s", servicesPath)
	}

	err = util.DecodeYaml(bytes.NewReader(sBuf), &services)
	if err != nil {
		return errors.Wrapf(err, "failed decode yaml for %s", servicesPath)
	}

	pBuf, err := box.Find(playlistPath)
	if err != nil {
		return errors.Wrapf(err, "failed to find packr box %s", playlistPath)
	}
	err = util.DecodeYaml(bytes.NewReader(pBuf), &playlists)
	if err != nil {
		return errors.Wrapf(err, "failed decode yaml for %s", playlistPath)
	}

	err = dumpFile(dockerComposePath, box)
	if err != nil {
		return errors.Wrapf(err, "failed to dump file to %s", dockerComposePath)
	}

	err = dumpFile(localstackEntrypointPath, box)
	if err != nil {
		return errors.Wrapf(err, "failed to dump file to %s", localstackEntrypointPath)
	}

	err = applyOverrides(services, tbrc.Overrides)
	if err != nil {
		return errors.Wrap(err, "failed to apply overrides from tbrc")
	}

	// Setup ECR image URIs for docker-compose
	for name, s := range services {
		if s.ECRTag == "" {
			continue
		}

		uri := ResolveEcrURI(name, s.ECRTag)
		uriVar := strings.ReplaceAll(strings.ToUpper(name), "-", "_") + "_IMAGE_URI"
		os.Setenv(uriVar, uri)
	}

	return nil
}

func Services() ServiceMap {
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

	return []string{}
}

func RmFiles() error {
	files := [...]string{dockerComposePath, localstackEntrypointPath}

	for _, file := range files {
		log.Debugf("Removing %s...\n", file)
		path := fmt.Sprintf("%s/%s", tbRoot, file)
		err := os.Remove(path)
		if err != nil {
			return errors.Wrapf(err, "could not remove file at %s", path)
		}
	}

	return nil
}
