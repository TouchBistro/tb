package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/compose"
	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	servicesPath             = "services.yml"
	playlistPath             = "playlists.yml"
	dockerComposePath        = "docker-compose.yml"
	localstackEntrypointPath = "localstack-entrypoint.sh"
	lazydockerConfigPath     = "lazydocker.yml"
)

type legacyServiceConfig struct {
	Global struct {
		BaseImages      []string          `yaml:"baseImages"`
		LoginStrategies []string          `yaml:"loginStrategies"`
		Variables       map[string]string `yaml:"variables"`
	} `yaml:"global"`
	Services map[string]service.Service `yaml:"services"`
}

func parseServices(config legacyServiceConfig) (map[string]service.Service, error) {
	parsedServices := make(map[string]service.Service)

	vars := config.Global.Variables
	vars["@ROOTPATH"] = TBRootPath()

	// Add vars for each service name
	for name := range config.Services {
		vars["@"+name] = "touchbistro-tb-registry-" + name
	}

	// Validate each service and perform any necessary actions
	for name, service := range config.Services {
		// Make sure either local or remote usage is specified
		if !service.CanBuild() && service.Remote.Image == "" {
			msg := fmt.Sprintf("Must specify at least one of 'build.dockerfilePath' or 'remote.image' for service %s", name)
			return nil, errors.New(msg)
		}

		// Make sure repo is specified if not using remote
		if !service.UseRemote() && !service.CanBuild() {
			msg := fmt.Sprintf("'remote.enabled: false' is set but 'build.dockerfilePath' was not provided for service %s", name)
			return nil, errors.New(msg)
		}

		// Set special service specific vars
		if service.HasGitRepo() {
			vars["@REPOPATH"] = filepath.Join(ReposPath(), service.GitRepo)
		} else {
			vars["@REPOPATH"] = ""
		}

		// Expand any vars
		for i, dep := range service.Dependencies {
			service.Dependencies[i] = util.ExpandVars(dep, vars)
		}

		service.Build.DockerfilePath = util.ExpandVars(service.Build.DockerfilePath, vars)
		service.EnvFile = util.ExpandVars(service.EnvFile, vars)
		service.Remote.Image = util.ExpandVars(service.Remote.Image, vars)

		for key, value := range service.EnvVars {
			service.EnvVars[key] = util.ExpandVars(value, vars)
		}

		for i, volume := range service.Build.Volumes {
			service.Build.Volumes[i].Value = util.ExpandVars(volume.Value, vars)
		}

		for i, volume := range service.Remote.Volumes {
			service.Remote.Volumes[i].Value = util.ExpandVars(volume.Value, vars)
		}

		parsedServices[name] = service
	}

	return parsedServices, nil
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

func legacyInit() error {
	box := packr.New("static", "../static")

	sBuf, err := box.Find(servicesPath)
	if err != nil {
		return errors.Wrapf(err, "failed to find packr box %s", servicesPath)
	}

	serviceConfig := legacyServiceConfig{}
	err = yaml.NewDecoder(bytes.NewReader(sBuf)).Decode(&serviceConfig)
	if err != nil {
		return errors.Wrapf(err, "failed decode yaml for %s", servicesPath)
	}

	globalConfig.BaseImages = serviceConfig.Global.BaseImages
	globalConfig.LoginStrategies = serviceConfig.Global.LoginStrategies

	pBuf, err := box.Find(playlistPath)
	if err != nil {
		return errors.Wrapf(err, "failed to find packr box %s", playlistPath)
	}

	// Bridge old world to new world
	playlistMap := make(map[string]playlist.Playlist)
	err = yaml.NewDecoder(bytes.NewReader(pBuf)).Decode(&playlistMap)
	if err != nil {
		return errors.Wrapf(err, "failed decode yaml for %s", playlistPath)
	}

	playlistList := make([]playlist.Playlist, 0, len(playlistMap))
	for n, p := range playlistMap {
		p.Name = n
		p.RegistryName = "TouchBistro/tb-registry"
		playlistList = append(playlistList, p)
	}

	playlists, err = playlist.NewPlaylistCollection(playlistList, tbrc.Playlists)
	if err != nil {
		return errors.Wrapf(err, "failed to create PlaylistCollection")
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

	serviceMap, err := parseServices(serviceConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to load services")
	}

	// Bridge old world to new world
	serviceList := make([]service.Service, 0, len(serviceMap))
	for n, s := range serviceMap {
		s.Name = n
		s.RegistryName = "TouchBistro/tb-registry"
		serviceList = append(serviceList, s)
	}

	services, err = service.NewServiceCollection(serviceList, tbrc.Overrides)
	if err != nil {
		return errors.Wrap(err, "failed add services to ServiceCollection and apply overrides from tbrc")
	}

	// Create docker-compose.yml
	composePath := filepath.Join(tbRoot, dockerComposePath)
	file, err := os.OpenFile(composePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", composePath)
	}
	defer file.Close()

	log.Debugln("Generating docker-compose.yml file...")
	err = compose.CreateComposeFile(services, file)
	if err != nil {
		return errors.Wrap(err, "failed to generated docker-compose file")
	}
	log.Debugln("Successfully generated docker-compose.yml")

	return nil
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
