package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// File names in registry
const (
	playlistsFileName = "playlists.yml"
	servicesFileName  = "services.yml"
	staticDirName     = "static"
)

type Registry struct {
	Name      string `yaml:"name"`
	LocalPath string `yaml:"localPath,omitempty"`
	Path      string `yaml:"-"`
}

type registryServiceConfig struct {
	Global struct {
		BaseImages      []string          `yaml:"baseImages"`
		LoginStrategies []string          `yaml:"loginStrategies"`
		Variables       map[string]string `yaml:"variables"`
	} `yaml:"global"`
	Services map[string]service.Service `yaml:"services"`
}

type GlobalConfig struct {
	BaseImages      []string
	LoginStrategies []string
}

func readRegistryFile(fileName string, r Registry, v interface{}) error {
	log.Debugf("Reading %s from registry %s", fileName, r.Name)

	filePath := filepath.Join(r.Path, fileName)
	if !file.FileOrDirExists(filePath) {
		log.Debugf("registry %s has no %s", r.Name, fileName)

		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", filePath)
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(v)
	return errors.Wrapf(err, "failed to read %s in registry %s", fileName, r.Name)
}

func ReadServices(r Registry, rootPath, reposPath string) ([]service.Service, GlobalConfig, error) {
	serviceConf := registryServiceConfig{}
	err := readRegistryFile(servicesFileName, r, &serviceConf)
	if err != nil {
		return nil, GlobalConfig{}, errors.Wrapf(err, "failed to read services file from registry %s", r.Name)
	}

	services := make([]service.Service, 0, len(serviceConf.Services))

	// Set special vars
	vars := serviceConf.Global.Variables

	// If no variables are defined in services.yml the map will be nil
	if vars == nil {
		vars = make(map[string]string)
	}

	vars["@ROOTPATH"] = rootPath
	vars["@STATICPATH"] = filepath.Join(r.Path, staticDirName)

	// Add vars for each service name
	for name := range serviceConf.Services {
		fullName := fmt.Sprintf("%s/%s", r.Name, name)
		vars["@"+name] = util.DockerName(fullName)
	}

	for n, s := range serviceConf.Services {
		s.Name = n
		s.RegistryName = r.Name

		// Make sure mode is a valid value
		if s.Mode != service.ModeRemote && s.Mode != service.ModeBuild {
			return nil, GlobalConfig{}, errors.Errorf("'%s.mode' value is invalid must be 'remote' or 'build'", n)
		}

		// Make sure image is specified if using remote
		if s.UseRemote() && s.Remote.Image == "" {
			return nil, GlobalConfig{}, errors.Errorf("'%s.mode' is set to 'remote' but 'remote.image' was not provided", n)
		}

		// Make sure repo is specified if not using remote
		if !s.UseRemote() && !s.CanBuild() {
			msg := fmt.Sprintf("'%s.mode' is set to 'build' but 'build.dockerfilePath' was not provided", n)
			return nil, GlobalConfig{}, errors.New(msg)
		}

		// Set special service specific vars
		if s.HasGitRepo() {
			vars["@REPOPATH"] = filepath.Join(reposPath, s.GitRepo)
		} else {
			vars["@REPOPATH"] = ""
		}

		// Expand any vars
		for i, dep := range s.Dependencies {
			s.Dependencies[i] = util.ExpandVars(dep, vars)
		}

		s.Build.DockerfilePath = util.ExpandVars(s.Build.DockerfilePath, vars)
		s.EnvFile = util.ExpandVars(s.EnvFile, vars)
		s.Remote.Image = util.ExpandVars(s.Remote.Image, vars)

		for key, value := range s.EnvVars {
			s.EnvVars[key] = util.ExpandVars(value, vars)
		}

		for i, volume := range s.Build.Volumes {
			s.Build.Volumes[i].Value = util.ExpandVars(volume.Value, vars)
		}

		for i, volume := range s.Remote.Volumes {
			s.Remote.Volumes[i].Value = util.ExpandVars(volume.Value, vars)
		}

		services = append(services, s)
	}

	globalConf := GlobalConfig{
		BaseImages:      serviceConf.Global.BaseImages,
		LoginStrategies: serviceConf.Global.LoginStrategies,
	}

	return services, globalConf, nil
}

func ReadPlaylists(r Registry) ([]playlist.Playlist, error) {
	playlistMap := make(map[string]playlist.Playlist)
	err := readRegistryFile(playlistsFileName, r, &playlistMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read playlist file from registry %s", r.Name)
	}

	playlists := make([]playlist.Playlist, 0, len(playlistMap))
	for n, p := range playlistMap {
		// Set necessary fields for each playlist
		p.Name = n
		p.RegistryName = r.Name

		// Make sure extends is a full name
		if p.Extends != "" {
			registryName, playlistName, err := util.SplitNameParts(p.Extends)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to resolve full name for extends field of playlist %s", n)
			}

			if registryName == "" {
				p.Extends = fmt.Sprintf("%s/%s", r.Name, playlistName)
			}
		}

		// Make sure each service name is the full name
		serviceNames := make([]string, len(p.Services))
		for i, name := range p.Services {
			registryName, serviceName, err := util.SplitNameParts(name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to resolve full name for service %s in playlist %s", n, name)
			}

			if registryName == "" {
				serviceNames[i] = fmt.Sprintf("%s/%s", r.Name, serviceName)
			} else {
				serviceNames[i] = name
			}
		}

		p.Services = serviceNames

		playlists = append(playlists, p)
	}

	return playlists, nil
}
