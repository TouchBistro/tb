package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	AppsFileName      = "apps.yml"
	PlaylistsFileName = "playlists.yml"
	ServicesFileName  = "services.yml"
	staticDirName     = "static"
)

const (
	resourceTypeApp     = "app"
	resourceTypeService = "service"
)

// ErrFileNotExist indicates a registry file does not exist.
var ErrFileNotExist = os.ErrNotExist

// TODO(@cszatmary): Figure out a better way to differentiate between app types
type appType int

const (
	appTypeiOS appType = iota
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

// ValidationError represents a resource having failed validation.
// It contains the resource type, name, and a list of validation
// failure messages.
type ValidationError struct {
	ResourceType string
	ResourceName string
	Messages     []string
}

func (ve *ValidationError) Error() string {
	var sb strings.Builder
	sb.WriteString(ve.ResourceType)
	sb.WriteString(": ")
	sb.WriteString(ve.ResourceName)
	sb.WriteString(": ")

	for i, msg := range ve.Messages {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(msg)
	}
	return sb.String()
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
	var msgs []string

	// Make sure mode is a valid value
	if s.Mode != service.ModeRemote && s.Mode != service.ModeBuild {
		msg := fmt.Sprintf("invalid 'mode' value %q, must be 'remote' or 'build'", s.Mode)
		msgs = append(msgs, msg)
	}

	// Make sure image is specified if using remote
	if s.UseRemote() && s.Remote.Image == "" {
		msgs = append(msgs, "'mode' is set to 'remote' but 'remote.image' was not provided")
	}

	// Make sure repo is specified if not using remote
	if !s.UseRemote() && !s.CanBuild() {
		msgs = append(msgs, "'mode' is set to 'build' but 'build.dockerfilePath' was not provided")
	}

	if msgs == nil {
		return nil
	}

	return &ValidationError{
		ResourceType: resourceTypeService,
		ResourceName: s.Name,
		Messages:     msgs,
	}
}

func validateApp(a app.App, t appType) error {
	// No validations needed for desktop currently
	if t != appTypeiOS {
		return nil
	}

	var msgs []string

	// Make sure RunsOn is valid
	if a.DeviceType() == app.DeviceTypeUnknown {
		msgs = append(msgs, "'runsOn' value is invalid, must be 'all', 'ipad', or 'iphone'")
	}

	if msgs == nil {
		return nil
	}

	return &ValidationError{
		ResourceType: resourceTypeApp,
		ResourceName: a.Name,
		Messages:     msgs,
	}
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

		if err := validateService(s); err != nil {
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
			errs = append(errs, &ValidationError{
				ResourceType: resourceTypeService,
				ResourceName: s.Name,
				Messages:     errMsgs,
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

		if err := validateApp(a, appTypeiOS); err != nil {
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
	if file.FileOrDirExists(appsPath) {
		_, _, err := readApps(r)
		if err != nil {
			result.AppsErr = err
		}
	} else {
		result.AppsErr = fmt.Errorf("%w: %s", ErrFileNotExist, AppsFileName)
	}

	// Validate playlists.yml

	playlistsPath := filepath.Join(path, PlaylistsFileName)
	if file.FileOrDirExists(playlistsPath) {
		_, err := readPlaylists(r)
		if err != nil {
			result.PlaylistsErr = err
		}
	} else {
		result.PlaylistsErr = fmt.Errorf("%w: %s", ErrFileNotExist, PlaylistsFileName)
	}

	// Validate services.yml

	servicesPath := filepath.Join(path, ServicesFileName)
	if file.FileOrDirExists(servicesPath) {
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
					errs = append(errs, &ValidationError{
						ResourceType: resourceTypeService,
						ResourceName: s.Name,
						Messages:     []string{msg},
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
