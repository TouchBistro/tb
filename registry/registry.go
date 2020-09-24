package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/app"
	"github.com/TouchBistro/tb/playlist"
	"github.com/TouchBistro/tb/service"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// File names in registry
const (
	appsFileName      = "apps.yml"
	playlistsFileName = "playlists.yml"
	servicesFileName  = "services.yml"
	staticDirName     = "static"
)

// TODO(@cszatmary): Figure out a better way to differentiate between app types
type appType int

const (
	appTypeiOS appType = iota
	appTypeDesktop
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

type serviceGlobalConfig struct {
	BaseImages      []string
	LoginStrategies []string
}

type registryAppConfig struct {
	IOSApps     map[string]app.App `yaml:"iosApps"`
	DesktopApps map[string]app.App `yaml:"desktopApps"`
}

type ReadOptions struct {
	ShouldReadServices bool
	ShouldReadApps     bool
	RootPath           string
	ReposPath          string
	Overrides          map[string]service.ServiceOverride
	CustomPlaylists    map[string]playlist.Playlist
}

type RegistryResult struct {
	Services        *service.ServiceCollection
	Playlists       *playlist.PlaylistCollection
	IOSApps         *app.AppCollection
	DesktopApps     *app.AppCollection
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

func validateService(s service.Service) error {
	// Make sure mode is a valid value
	if s.Mode != service.ModeRemote && s.Mode != service.ModeBuild {
		return errors.Errorf("'%s.mode' value is invalid must be 'remote' or 'build'", s.Name)
	}

	// Make sure image is specified if using remote
	if s.UseRemote() && s.Remote.Image == "" {
		return errors.Errorf("'%s.mode' is set to 'remote' but 'remote.image' was not provided", s.Name)
	}

	// Make sure repo is specified if not using remote
	if !s.UseRemote() && !s.CanBuild() {
		return errors.Errorf("'%s.mode' is set to 'build' but 'build.dockerfilePath' was not provided", s.Name)
	}

	return nil
}

func validateApp(a app.App, t appType) error {
	// No validations needed for desktop currently
	if t != appTypeiOS {
		return nil
	}

	// Make sure RunsOn is valid
	if a.DeviceType() == app.DeviceTypeUnknown {
		return errors.Errorf("'%s.runsOn' value is invalid, must be 'all', 'ipad', or 'iphone'", a.Name)
	}

	return nil
}

type readServicesOptions struct {
	rootPath  string
	reposPath string
	overrides map[string]service.ServiceOverride
}

func readServices(r Registry, opts readServicesOptions) ([]service.Service, serviceGlobalConfig, error) {
	serviceConf := registryServiceConfig{}
	err := readRegistryFile(servicesFileName, r, &serviceConf)
	if err != nil {
		return nil, serviceGlobalConfig{}, errors.Wrapf(err, "failed to read services file from registry %s", r.Name)
	}

	services := make([]service.Service, 0, len(serviceConf.Services))

	// Set special vars
	vars := serviceConf.Global.Variables

	// If no variables are defined in services.yml the map will be nil
	if vars == nil {
		vars = make(map[string]string)
	}

	vars["@ROOTPATH"] = opts.rootPath
	vars["@STATICPATH"] = filepath.Join(r.Path, staticDirName)

	// Add vars for each service name
	for name := range serviceConf.Services {
		fullName := fmt.Sprintf("%s/%s", r.Name, name)
		vars["@"+name] = util.DockerName(fullName)
	}

	for n, s := range serviceConf.Services {
		s.Name = n
		s.RegistryName = r.Name

		err := validateService(s)
		if err != nil {
			return nil, serviceGlobalConfig{}, errors.Wrapf(err, "service %s failed validation", n)
		}

		override, ok := opts.overrides[s.FullName()]

		// Set special service specific vars
		repoPath := ""
		if ok && override.GitRepo.Path != "" {
			p := override.GitRepo.Path
			if strings.HasPrefix(p, "~") {
				repoPath = filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(p, "~"))
			} else {
				repoPath = p
			}
		} else if s.HasGitRepo() {
			repoPath = filepath.Join(opts.reposPath, s.GitRepo.Name)
		}

		vars["@REPOPATH"] = repoPath

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

	globalConf := serviceGlobalConfig{
		BaseImages:      serviceConf.Global.BaseImages,
		LoginStrategies: serviceConf.Global.LoginStrategies,
	}

	return services, globalConf, nil
}

func readPlaylists(r Registry) ([]playlist.Playlist, error) {
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

func readApps(r Registry) ([]app.App, []app.App, error) {
	appConf := registryAppConfig{}
	err := readRegistryFile(appsFileName, r, &appConf)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read apps file from registry %s", r.Name)
	}

	iosApps := make([]app.App, 0, len(appConf.IOSApps))
	desktopApps := make([]app.App, 0, len(appConf.DesktopApps))

	// Deal with iOS apps
	for n, a := range appConf.IOSApps {
		a.Name = n
		a.RegistryName = r.Name

		err := validateApp(a, appTypeiOS)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "app %s failed validation", n)
		}

		iosApps = append(iosApps, a)
	}

	// Deal with desktop apps
	for n, a := range appConf.DesktopApps {
		a.Name = n
		a.RegistryName = r.Name

		desktopApps = append(desktopApps, a)
	}

	return iosApps, desktopApps, nil
}

func ReadRegistries(registries []Registry, opts ReadOptions) (RegistryResult, error) {
	serviceList := make([]service.Service, 0)
	playlistList := make([]playlist.Playlist, 0)
	baseImages := make([]string, 0)
	loginStrategies := make([]string, 0)
	iosAppList := make([]app.App, 0)
	desktopAppList := make([]app.App, 0)

	for _, r := range registries {
		if opts.ShouldReadServices {
			log.Debugf("Reading services from registry %s", r.Name)

			services, globalConf, err := readServices(r, readServicesOptions{
				rootPath:  opts.RootPath,
				reposPath: opts.ReposPath,
				overrides: opts.Overrides,
			})
			if err != nil {
				return RegistryResult{}, errors.Wrapf(err, "failed to read services from registry %s", r.Name)
			}

			log.Debugf("Reading playlists from registry %s", r.Name)

			playlists, err := readPlaylists(r)
			if err != nil {
				return RegistryResult{}, errors.Wrapf(err, "failed to read playlists from registry %s", r.Name)
			}

			serviceList = append(serviceList, services...)
			playlistList = append(playlistList, playlists...)
			baseImages = append(baseImages, globalConf.BaseImages...)
			loginStrategies = append(loginStrategies, globalConf.LoginStrategies...)
		}

		if opts.ShouldReadApps {
			log.Debugf("Reading apps from registry %s", r.Name)

			iosApps, desktopApps, err := readApps(r)
			if err != nil {
				return RegistryResult{}, errors.Wrapf(err, "failed to read apps from registry %s", r.Name)
			}

			iosAppList = append(iosAppList, iosApps...)
			desktopAppList = append(desktopAppList, desktopApps...)
		}
	}

	var sc *service.ServiceCollection
	var pc *playlist.PlaylistCollection
	var err error
	if opts.ShouldReadServices {
		sc, err = service.NewServiceCollection(serviceList, opts.Overrides)
		if err != nil {
			return RegistryResult{}, errors.Wrap(err, "failed to create ServiceCollection")
		}

		pc, err = playlist.NewPlaylistCollection(playlistList, opts.CustomPlaylists)
		if err != nil {
			return RegistryResult{}, errors.Wrap(err, "failed to create PLaylistCollection")
		}
	}

	var iosAC, desktopAC *app.AppCollection
	if opts.ShouldReadApps {
		iosAC, err = app.NewAppCollection(iosAppList)
		if err != nil {
			return RegistryResult{}, errors.Wrap(err, "failed to create AppCollection for iOS apps")
		}

		desktopAC, err = app.NewAppCollection(desktopAppList)
		if err != nil {
			return RegistryResult{}, errors.Wrap(err, "failed to create AppCollection for desktop apps")
		}
	}

	return RegistryResult{
		Services:        sc,
		Playlists:       pc,
		IOSApps:         iosAC,
		DesktopApps:     desktopAC,
		BaseImages:      util.UniqueStrings(baseImages),
		LoginStrategies: util.UniqueStrings(loginStrategies),
	}, nil
}

func Validate(path string) error {
	// Validate apps.yml
	log.Infof(color.Cyan("Validating %s..."), appsFileName)

	appsPath := filepath.Join(path, appsFileName)
	if file.FileOrDirExists(appsPath) {
		af, err := os.Open(appsPath)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", appsPath)
		}
		defer af.Close()

		appConf := registryAppConfig{}
		err = yaml.NewDecoder(af).Decode(&appConf)
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", appsPath)
		}

		// Make sure all iOS apps are valid
		for n, a := range appConf.IOSApps {
			a.Name = n
			err := validateApp(a, appTypeiOS)
			if err != nil {
				log.Infof(color.Red("❌ app %s is invalid"), n)
				return errors.Wrapf(err, "app %s failed validation", n)
			}
		}

		log.Infof(color.Green("✅ %s is valid"), appsFileName)
	} else {
		log.Infof(color.Yellow("No %s file"), appsFileName)
	}

	// Validate playlists.yml
	log.Infof(color.Cyan("Validating %s..."), playlistsFileName)

	playlistsPath := filepath.Join(path, playlistsFileName)
	if file.FileOrDirExists(playlistsPath) {
		pf, err := os.Open(playlistsPath)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", playlistsPath)
		}
		defer pf.Close()

		playlistMap := make(map[string]playlist.Playlist)
		err = yaml.NewDecoder(pf).Decode(&playlistMap)
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", playlistsPath)
		}

		log.Infof(color.Green("✅ %s is valid"), playlistsFileName)
	} else {
		log.Infof(color.Yellow("No %s file"), playlistsFileName)
	}

	// Validate services.yml
	log.Infof(color.Cyan("Validating %s..."), servicesFileName)

	servicesPath := filepath.Join(path, servicesFileName)
	if file.FileOrDirExists(servicesPath) {
		sf, err := os.Open(servicesPath)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", servicesPath)
		}
		defer sf.Close()

		serviceConf := registryServiceConfig{}
		err = yaml.NewDecoder(sf).Decode(&serviceConf)
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", servicesPath)
		}

		// Keep track of ports to check for conflicting ports
		usedPorts := make(map[string]string)

		// Make sure all services are valid
		for n, s := range serviceConf.Services {
			s.Name = n
			err := validateService(s)
			if err != nil {
				log.Infof(color.Red("❌ service %s is invalid"), n)
				return errors.Wrapf(err, "service %s failed validation", n)
			}

			// Check for port conflict
			for _, p := range s.Ports {
				// ports are of the form EXTERNAL:INTERNAL
				// get external part
				exposedPort := strings.Split(p, ":")[0]
				if conflict, ok := usedPorts[exposedPort]; ok {
					log.Infof(color.Red("❌ service %s has conflicting port %s with service %s"), n, exposedPort, conflict)
					return errors.Errorf("service %s failed validation", n)
				}

				usedPorts[exposedPort] = n
			}
		}

		log.Infof(color.Green("✅ %s is valid"), servicesFileName)
	} else {
		log.Infof(color.Yellow("No %s file"), servicesFileName)
	}

	return nil
}
