package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/app"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// File names in registry
const (
	AppsFileName      = "apps.yml"
	PlaylistsFileName = "playlists.yml"
	ServicesFileName  = "services.yml"
	staticDirName     = "static"
)

// ErrFileNotExist indicates a registry file does not exist.
var ErrFileNotExist = os.ErrNotExist

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
	Services        *service.Collection
	Playlists       *playlist.Collection
	IOSApps         *app.Collection
	DesktopApps     *app.Collection
	BaseImages      []string
	LoginStrategies []string
}

// ErrorList is a list of errors encountered.
type ErrorList []error

func (e ErrorList) Error() string {
	errStrs := make([]string, len(e))
	for i, err := range e {
		errStrs[i] = err.Error()
	}
	return strings.Join(errStrs, "\n")
}

func readRegistryFile(fileName string, r Registry, v interface{}) error {
	log.Debugf("Reading %s from registry %s", fileName, r.Name)

	filePath := filepath.Join(r.Path, fileName)
	if !file.Exists(filePath) {
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

type readServicesOptions struct {
	rootPath  string
	reposPath string
	overrides map[string]service.ServiceOverride
	strict    bool
}

func readServices(r Registry, opts readServicesOptions) ([]service.Service, serviceGlobalConfig, error) {
	serviceConf := registryServiceConfig{}
	err := readRegistryFile(ServicesFileName, r, &serviceConf)
	if err != nil {
		return nil, serviceGlobalConfig{}, errors.Wrapf(err, "failed to read services file from registry %s", r.Name)
	}

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

	services := make([]service.Service, 0, len(serviceConf.Services))
	var errs ErrorList
	for n, s := range serviceConf.Services {
		s.Name = n
		s.RegistryName = r.Name

		if err := service.Validate(s); err != nil {
			errs = append(errs, err)
			continue
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
		var errMsgs []string
		for i, dep := range s.Dependencies {
			d := dep
			expandVarsInField(&d, vars, &errMsgs, "dependencies")
			s.Dependencies[i] = d
		}

		expandVarsInField(&s.Build.DockerfilePath, vars, &errMsgs, "build.dockerfilePath")
		expandVarsInField(&s.EnvFile, vars, &errMsgs, "envFile")
		expandVarsInField(&s.Remote.Image, vars, &errMsgs, "remote.image")

		for key, value := range s.EnvVars {
			v := value
			expandVarsInField(&v, vars, &errMsgs, "envVars")
			s.EnvVars[key] = v
		}

		for i, volume := range s.Build.Volumes {
			v := volume.Value
			expandVarsInField(&v, vars, &errMsgs, "build.volumes")
			s.Build.Volumes[i].Value = v
		}

		for i, volume := range s.Remote.Volumes {
			v := volume.Value
			expandVarsInField(&v, vars, &errMsgs, "remote.volumes")
			s.Remote.Volumes[i].Value = v

		}

		// Report unknown vars as an error if in strict mode
		if len(errMsgs) > 0 && opts.strict {
			errs = append(errs, &resource.ValidationError{
				Resource: s,
				Messages: errMsgs,
			})
			continue
		}

		services = append(services, s)
	}

	if errs != nil {
		return nil, serviceGlobalConfig{}, errs
	}

	globalConf := serviceGlobalConfig{
		BaseImages:      serviceConf.Global.BaseImages,
		LoginStrategies: serviceConf.Global.LoginStrategies,
	}

	return services, globalConf, nil
}

// expandVarsInField is a small helper to expand vars in a service field
// and report any errors. If the expansion is successful, the value pointed to
// by field will be updated. If an error occurs, the error message will be appended
// to errMsgs.
func expandVarsInField(field *string, vars map[string]string, errMsgs *[]string, fieldName string) {
	e, err := util.ExpandVars(*field, vars)
	if err == nil {
		*field = e
		return
	}

	msg := fmt.Sprintf("%s: %s", fieldName, err)
	*errMsgs = append(*errMsgs, msg)
}

func readPlaylists(r Registry) ([]playlist.Playlist, error) {
	playlistMap := make(map[string]playlist.Playlist)
	err := readRegistryFile(PlaylistsFileName, r, &playlistMap)
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
	err := readRegistryFile(AppsFileName, r, &appConf)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read apps file from registry %s", r.Name)
	}

	var errs ErrorList

	// Deal with iOS apps
	iosApps := make([]app.App, 0, len(appConf.IOSApps))
	for n, a := range appConf.IOSApps {
		a.Name = n
		a.RegistryName = r.Name

		if err := app.Validate(a, app.TypeiOS); err != nil {
			errs = append(errs, err)
		}

		iosApps = append(iosApps, a)
	}

	// Deal with desktop apps
	desktopApps := make([]app.App, 0, len(appConf.DesktopApps))
	for n, a := range appConf.DesktopApps {
		a.Name = n
		a.RegistryName = r.Name

		desktopApps = append(desktopApps, a)
	}

	if errs != nil {
		return nil, nil, errs
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

	var sc *service.Collection
	var pc *playlist.Collection
	var err error
	if opts.ShouldReadServices {
		// TODO(@cszatmary): Refactor the registry implementation to create collections at the start
		// and add to them as each registry is read.
		sc = &service.Collection{}
		for _, s := range serviceList {
			if o, ok := opts.Overrides[s.FullName()]; ok {
				s, err = service.Override(s, o)
				if err != nil {
					return RegistryResult{}, errors.Wrap(err, "failed to apply override to service")
				}
			}
			if err := sc.Set(s); err != nil {
				return RegistryResult{}, errors.Wrap(err, "failed to add service to collection")
			}
		}

		pc = &playlist.Collection{}
		for _, p := range playlistList {
			if err := pc.Set(p); err != nil {
				return RegistryResult{}, errors.Wrap(err, "failed to add playlist to collection")
			}
		}
		for n, p := range opts.CustomPlaylists {
			p.Name = n
			pc.SetCustom(p)
		}
	}

	var iosAC, desktopAC *app.Collection
	if opts.ShouldReadApps {
		iosAC = &app.Collection{}
		for _, a := range iosAppList {
			if err := iosAC.Set(a); err != nil {
				return RegistryResult{}, errors.Wrap(err, "failed to add iOS app to collection")
			}
		}
		desktopAC = &app.Collection{}
		for _, a := range desktopAppList {
			if err := desktopAC.Set(a); err != nil {
				return RegistryResult{}, errors.Wrap(err, "failed to add desktop app to collection")
			}
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

type ValidateResult struct {
	AppsErr      error
	PlaylistsErr error
	ServicesErr  error
}

// Validate checks to see if the registry located at path is valid. It will read and validate
// each configuration file in the registry. If strict is true unknown variables will be considered errors.
//
// Validate returns a ValidateResult struct that contains errors encountered for each resource.
// If a configuration file is valid, then the corresponding error value will be nil. Otherwise,
// the error will be a non-nil value containing the details of why validation failed.
// If a configuration file does not exist, then the corresponding error will be ErrFileNotExist.
func Validate(path string, strict bool) ValidateResult {
	r := Registry{
		Name: filepath.Base(path),
		Path: path,
	}
	result := ValidateResult{}

	// Validate apps.yml

	// Do explicit check for existence because we want to print a custom message
	// If it doesn't exist
	appsPath := filepath.Join(path, AppsFileName)
	if file.Exists(appsPath) {
		_, _, err := readApps(r)
		if err != nil {
			result.AppsErr = err
		}
	} else {
		result.AppsErr = fmt.Errorf("%w: %s", ErrFileNotExist, AppsFileName)
	}

	// Validate playlists.yml

	playlistsPath := filepath.Join(path, PlaylistsFileName)
	if file.Exists(playlistsPath) {
		_, err := readPlaylists(r)
		if err != nil {
			result.PlaylistsErr = err
		}
	} else {
		result.PlaylistsErr = fmt.Errorf("%w: %s", ErrFileNotExist, PlaylistsFileName)
	}

	// Validate services.yml

	servicesPath := filepath.Join(path, ServicesFileName)
	if file.Exists(servicesPath) {
		services, _, err := readServices(r, readServicesOptions{strict: strict})
		if err == nil {
			// Keep track of ports to check for conflicting ports
			usedPorts := make(map[string]string)

			// Perform additional validations
			var errs ErrorList
			for _, s := range services {
				// Check for port conflict
				for _, p := range s.Ports {
					// ports are of the form EXTERNAL:INTERNAL
					// get external part
					exposedPort := strings.Split(p, ":")[0]
					conflict, ok := usedPorts[exposedPort]
					if !ok {
						usedPorts[exposedPort] = s.Name
						continue
					}

					// Handle port conflict
					msg := fmt.Sprintf("conflicting port %s with service %s", exposedPort, conflict)
					errs = append(errs, &resource.ValidationError{
						Resource: s,
						Messages: []string{msg},
					})
				}
			}
			if errs != nil {
				result.ServicesErr = errs
			}
		} else {
			result.ServicesErr = err
		}
	} else {
		result.ServicesErr = fmt.Errorf("%w: %s", ErrFileNotExist, ServicesFileName)
	}

	return result
}
