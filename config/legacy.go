package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/util"
	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

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

func legacyInit() error {
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

	appConfig.IOSApps = map[string]IOSApp{
		"TouchBistro": {
			BundleID: "com.touchbistro.TouchBistro",
			Branch:   "develop",
			Repo:     "TouchBistro/tb-pos",
			EnvVars: map[string]string{
				"debug.autoAcceptTOS": "true",
			},
		},
	}

	return nil
}
