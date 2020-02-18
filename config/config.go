package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr/v2"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/login"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var serviceConfig ServiceConfig
var playlists map[string]Playlist
var tbRoot string

const (
	servicesPath             = "services.yml"
	playlistPath             = "playlists.yml"
	dockerComposePath        = "docker-compose.yml"
	localstackEntrypointPath = "localstack-entrypoint.sh"
	lazydockerConfigPath     = "lazydocker.yml"
)

/* Getters for private & computed vars */

func TBRootPath() string {
	return tbRoot
}

func ReposPath() string {
	return filepath.Join(tbRoot, "repos")
}

func Services() ServiceMap {
	return serviceConfig.Services
}

func Playlists() map[string]Playlist {
	return playlists
}

func LoginStategies() ([]login.LoginStrategy, error) {
	s, err := login.ParseStrategies(serviceConfig.Global.LoginStategies)
	return s, errors.Wrap(err, "Failed to parse login strategies")
}

func BaseImages() []string {
	return serviceConfig.Global.BaseImages
}

/* Private functions */

func setupEnv() error {
	// Set $TB_ROOT so it works in the docker-compose file
	tbRoot = filepath.Join(os.Getenv("HOME"), ".tb")
	os.Setenv("TB_ROOT", tbRoot)

	// Create $TB_ROOT directory if it doesn't exist
	if !file.FileOrDirExists(tbRoot) {
		err := os.Mkdir(tbRoot, 0755)
		if err != nil {
			return errors.Wrapf(err, "failed to create $TB_ROOT directory at %s", tbRoot)
		}
	}
	return nil
}

func dumpFile(from, to, dir string, box *packr.Box) error {
	path := filepath.Join(dir, to)
	buf, err := box.Find(from)
	if err != nil {
		return errors.Wrapf(err, "failed to find packr box %s", from)
	}

	var reason string
	// If file exists compare the checksum to the packr version
	if file.FileOrDirExists(path) {
		log.Debugf("%s exists", path)
		log.Debugf("comparing checksums for %s", from)

		fileBuf, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read contents of %s", path)
		}

		memChecksum, err := util.MD5Checksum(buf)
		if err != nil {
			return errors.Wrapf(err, "failed to get checksum of %s in packr box", from)
		}

		fileChecksum, err := util.MD5Checksum(fileBuf)
		if err != nil {
			return errors.Wrapf(err, "failed to get checksum of %s", path)
		}

		// checksums are the same, leave as is
		if bytes.Equal(memChecksum, fileChecksum) {
			log.Debugf("checksums match, leaving %s as is", from)
			return nil
		}

		reason = "is outdated, recreating file..."
	} else {
		reason = "does not exist, creating file..."
	}

	log.Debugf("%s %s", path, reason)

	err = ioutil.WriteFile(path, buf, 0644)
	return errors.Wrapf(err, "failed to write contents of %s to %s", from, path)
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

	err = util.DecodeYaml(bytes.NewReader(sBuf), &serviceConfig)
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

	err = dumpFile(localstackEntrypointPath, localstackEntrypointPath, tbRoot, box)
	if err != nil {
		return errors.Wrapf(err, "failed to dump file to %s", localstackEntrypointPath)
	}

	ldPath := filepath.Join(os.Getenv("HOME"), "Library/Application Support/jesseduffield/lazydocker")
	err = os.MkdirAll(ldPath, 0766)
	if err != nil {
		return errors.Wrapf(err, "failed to create lazydocker config directory %s", ldPath)
	}

	err = dumpFile(lazydockerConfigPath, "config.yml", ldPath, box)
	if err != nil {
		return errors.Wrapf(err, "failed to dump file to %s", lazydockerConfigPath)
	}

	services, err := parseServices(serviceConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to load services")
	}

	services, err = applyOverrides(services, tbrc.Overrides)
	if err != nil {
		return errors.Wrap(err, "failed to apply overrides from tbrc")
	}

	// Create docker-compose.yml
	composePath := filepath.Join(tbRoot, dockerComposePath)
	file, err := os.OpenFile(composePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", composePath)
	}
	defer file.Close()

	log.Debugln("Generating docker-compose.yml file...")
	err = CreateComposeFile(services, file)
	if err != nil {
		return errors.Wrap(err, "failed to generated docker-compose file")
	}
	log.Debugln("Successfully generated docker-compose.yml")

	serviceConfig.Services = services

	return nil
}

func GetPlaylist(name string, deps map[string]bool) ([]string, error) {
	// TODO: Make this less yolo if Init() wasn't called
	if playlists == nil {
		log.Panic("this is a bug. playlists is not initialised")
	}
	customList := tbrc.Playlists

	// Check custom playlists first
	if playlist, ok := customList[name]; ok {
		// Resolve parent playlist defined in extends
		if playlist.Extends != "" {
			deps[name] = true
			if deps[playlist.Extends] {
				msg := fmt.Sprintf("Circular dependency of services, %s and %s", playlist.Extends, name)
				return []string{}, errors.New(msg)
			}
			parentPlaylist, err := GetPlaylist(playlist.Extends, deps)
			return append(parentPlaylist, playlist.Services...), err
		}

		return playlist.Services, nil
	} else if playlist, ok := playlists[name]; ok {
		if playlist.Extends != "" {
			deps[name] = true
			if deps[playlist.Extends] {
				msg := fmt.Sprintf("Circular dependency of services, %s and %s", playlist.Extends, name)
				return []string{}, errors.New(msg)
			}
			parentPlaylist, err := GetPlaylist(playlist.Extends, deps)
			return append(parentPlaylist, playlist.Services...), err
		}

		return playlist.Services, nil
	}

	return []string{}, nil
}

func RmFiles() error {
	files := [...]string{dockerComposePath, localstackEntrypointPath}

	for _, file := range files {
		log.Debugf("Removing %s...\n", file)
		path := filepath.Join(tbRoot, file)
		err := os.Remove(path)
		if err != nil {
			return errors.Wrapf(err, "could not remove file at %s", path)
		}
	}

	return nil
}
