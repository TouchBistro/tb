package registry

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/goutils/text"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/app"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
	"github.com/TouchBistro/tb/util"
	"gopkg.in/yaml.v3"
)

// File names in registry
const (
	AppsFileName      = "apps.yml"
	PlaylistsFileName = "playlists.yml"
	ServicesFileName  = "services.yml"
	staticDirName     = "static"
)

// Registry configures a registry. A registry is a Git repo
// that contains configuration for services, playlists, and
// apps that tb can run.
type Registry struct {
	// Name is the name of the registry.
	// Must be of the form <org>/<repo>.
	Name string `yaml:"name"`
	// LocalPath specifies the location of the registry
	// on the local filesystem.
	LocalPath string `yaml:"localPath,omitempty"`

	// Path is the path to the local clone of the registry.
	Path string `yaml:"-"`
}

type registryServiceConfig struct {
	Global struct {
		BaseImages      []string          `yaml:"baseImages"`
		LoginStrategies []string          `yaml:"loginStrategies"`
		Variables       map[string]string `yaml:"variables"`
	} `yaml:"global"`
	Services map[string]service.Service `yaml:"services"`
}

type registryAppConfig struct {
	IOSApps     map[string]app.App `yaml:"iosApps"`
	DesktopApps map[string]app.App `yaml:"desktopApps"`
}

func readRegistryFile(op errors.Op, filename string, r Registry, v interface{}) error {
	fp := filepath.Join(r.Path, filename)
	f, err := os.Open(fp)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to open file %s in registry %s", filename, r.Name),
			Op:     op,
		})
	}
	defer f.Close()
	if err := yaml.NewDecoder(f).Decode(v); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to decode %s in registry %s", filename, r.Name),
			Op:     op,
		})
	}
	return nil
}

type readServicesOptions struct {
	collection *service.Collection
	homeDir    string
	rootPath   string
	reposPath  string
	overrides  map[string]service.ServiceOverride
	strict     bool
}

type serviceGlobalConfig struct {
	baseImages      []string
	loginStrategies []string
}

// readServices reads the service config from the registry r.
func readServices(op errors.Op, r Registry, opts readServicesOptions) (serviceGlobalConfig, error) {
	var serviceConf registryServiceConfig
	err := readRegistryFile(op, ServicesFileName, r, &serviceConf)
	if err != nil {
		return serviceGlobalConfig{}, err
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
		fullName := resource.FullName(r.Name, name)
		vars["@"+name] = util.DockerName(fullName)
	}

	var errs errors.List
	for n, s := range serviceConf.Services {
		s.Name = n
		s.RegistryName = r.Name
		if err := service.Validate(s); err != nil {
			errs = append(errs, err)
			continue
		}

		override, ok := opts.overrides[s.FullName()]

		// Set special service specific vars
		var repoPath string
		if ok && override.GitRepo.Path != "" {
			repoPath = override.GitRepo.Path
			if strings.HasPrefix(repoPath, "~") {
				repoPath = filepath.Join(opts.homeDir, strings.TrimPrefix(repoPath, "~"))
			}
		} else if s.HasGitRepo() {
			repoPath = filepath.Join(opts.reposPath, s.GitRepo.Name)
		}
		vars["@REPOPATH"] = repoPath

		// Expand any vars
		ve := variableExpander{vars: vars}
		for i, dep := range s.Dependencies {
			s.Dependencies[i] = ve.expand(dep, "dependencies")
		}
		s.Build.DockerfilePath = ve.expand(s.Build.DockerfilePath, "build.dockerfilePath")
		s.EnvFile = ve.expand(s.EnvFile, "envFile")
		s.Remote.Image = ve.expand(s.Remote.Image, "remote.image")

		for key, value := range s.EnvVars {
			s.EnvVars[key] = ve.expand(value, "envVars")
		}
		for i, volume := range s.Build.Volumes {
			s.Build.Volumes[i].Value = ve.expand(volume.Value, "build.volumes")
		}
		for i, volume := range s.Remote.Volumes {
			s.Remote.Volumes[i].Value = ve.expand(volume.Value, "remote.volumes")
		}

		// Report unknown vars as an error if in strict mode
		if len(ve.errMsgs) > 0 && opts.strict {
			errs = append(errs, &resource.ValidationError{
				Resource: s,
				Messages: ve.errMsgs,
			})
			continue
		}

		// Apply overrides
		if ok {
			s, err = service.Override(s, override)
			if err != nil {
				msg := fmt.Sprintf("failed to apply override to service %s", s.FullName())
				errs = append(errs, errors.Wrap(err, errors.Meta{Reason: msg, Op: op}))
				continue
			}
		}
		if err := opts.collection.Set(s); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	if len(errs) > 0 {
		return serviceGlobalConfig{}, errs
	}
	return serviceGlobalConfig{
		baseImages:      serviceConf.Global.BaseImages,
		loginStrategies: serviceConf.Global.LoginStrategies,
	}, nil
}

// variableExpander is a small helper type which expands variables in a service field.
// It records a list of error messages for missing variables.
type variableExpander struct {
	vars      map[string]string
	errMsgs   []string
	fieldName string // temp, used for expansion
}

// expand expands any variables in field. It is not safe for concurrent use.
func (ve *variableExpander) expand(field string, fieldName string) string {
	ve.fieldName = fieldName
	return text.ExpandVariablesString(field, ve.mapping)
}

func (ve *variableExpander) mapping(name string) string {
	// @env is used to "escape" expansion. The value after will be used literally.
	// Ex: ${@env:HOME} becomes ${HOME}
	const envPrefix = "@env:"
	if strings.HasPrefix(name, envPrefix) {
		return fmt.Sprintf("${%s}", strings.TrimPrefix(name, envPrefix))
	}
	if v, ok := ve.vars[name]; ok {
		return v
	}
	// Missing var, record error
	ve.errMsgs = append(ve.errMsgs, fmt.Sprintf("%s: unknown variable %q", ve.fieldName, name))
	return ""
}

// readPlaylists reads the playlist config from the registry r.
func readPlaylists(op errors.Op, r Registry, collection *playlist.Collection) error {
	playlistMap := make(map[string]playlist.Playlist)
	err := readRegistryFile(op, PlaylistsFileName, r, &playlistMap)
	if err != nil {
		return err
	}

	var errs errors.List
	for n, p := range playlistMap {
		// Set necessary fields for each playlist
		p.Name = n
		p.RegistryName = r.Name

		// Make sure extends is a full name
		if p.Extends != "" {
			registryName, playlistName, err := resource.ParseName(p.Extends)
			if err != nil {
				msg := fmt.Sprintf("failed to resolve full name for extends field of playlist %s", p.FullName())
				errs = append(errs, errors.Wrap(err, errors.Meta{Reason: msg, Op: op}))
				continue
			}
			if registryName == "" {
				p.Extends = resource.FullName(r.Name, playlistName)
			}
		}

		// Make sure each service name is the full name
		serviceNames := make([]string, len(p.Services))
		for i, name := range p.Services {
			registryName, serviceName, err := resource.ParseName(name)
			if err != nil {
				msg := fmt.Sprintf("failed to resolve full name for service %s in playlist %s", name, p.FullName())
				errs = append(errs, errors.Wrap(err, errors.Meta{Reason: msg, Op: op}))
				continue
			}
			if registryName == "" {
				serviceNames[i] = resource.FullName(r.Name, serviceName)
			} else {
				serviceNames[i] = name
			}
		}
		p.Services = serviceNames
		if err := collection.Set(p); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

type readAppsOptions struct {
	iosCollection     *app.Collection
	desktopCollection *app.Collection
}

// readPlaylists reads the app config from the registry r.
// If the given collection is nil, apps are only validated.
func readApps(op errors.Op, r Registry, opts readAppsOptions) error {
	var appConf registryAppConfig
	err := readRegistryFile(op, AppsFileName, r, &appConf)
	if err != nil {
		return err
	}

	var errs errors.List
	// Deal with iOS apps
	for n, a := range appConf.IOSApps {
		a.Name = n
		a.RegistryName = r.Name
		if err := app.Validate(a, app.TypeiOS); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := opts.iosCollection.Set(a); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	// Deal with desktop apps
	for n, a := range appConf.DesktopApps {
		a.Name = n
		a.RegistryName = r.Name
		if err := opts.desktopCollection.Set(a); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ReadAllOptions allows for customizing the behaviour of ReadAll.
type ReadAllOptions struct {
	// ReadServices specifies if services and playlists should be read.
	ReadServices bool
	// ReadApps specifies if apps should be read.
	ReadApps bool
	// HomeDir specifies the home directory that should be used for expanding paths.
	HomeDir string
	// RootPath is the root path where tb stores files.
	// It is required if ReadServices is true.
	RootPath string
	// ReposPath is the path where cloned repos are stored.
	// It is required if ReadServices is true.
	ReposPath string
	// Overrides are any overrides that should be applied to services.
	Overrides map[string]service.ServiceOverride
	// Logger can be provided to log debug details while reading registries.
	// If it is nil, logging is off.
	Logger progress.Logger
}

// ReadAllResult contains the result of reading a list of registries.
type ReadAllResult struct {
	// Services is a collection of all services read.
	// If ReadAllOptions.ReadServices was false, it will be nil.
	Services *service.Collection
	// Playlists is a collection of all playlists read.
	// If ReadAllOptions.ReadServices was false, it will be nil.
	Playlists *playlist.Collection
	// IOSApps is a collection of all iOS apps read.
	// If ReadAllOptions.ReadApps was false, it will be nil.
	IOSApps *app.Collection
	// DesktopApps is a collection of all desktop apps read.
	// If ReadAllOptions.ReadApps was false, it will be nil.
	DesktopApps *app.Collection
	// BaseImages is a list of all base images read.
	// If ReadAllOptions.ReadServices was false, it will be empty.
	BaseImages []string
	// BaseImages is a list of all login strategies read.
	// If ReadAllOptions.ReadServices was false, it will be empty.
	LoginStrategies []string
}

// ReadAll reads all the given registries and returns the combined result.
func ReadAll(registries []Registry, opts ReadAllOptions) (ReadAllResult, error) {
	const op = errors.Op("registry.ReadAll")
	var result ReadAllResult
	if opts.ReadServices {
		result.Services = &service.Collection{}
		result.Playlists = &playlist.Collection{}
	}
	if opts.ReadApps {
		result.IOSApps = &app.Collection{}
		result.DesktopApps = &app.Collection{}
	}
	if opts.Logger == nil {
		opts.Logger = progress.NoopTracker{}
	}

	for _, r := range registries {
		if opts.ReadServices {
			opts.Logger.Debugf("Reading services from registry %s", r.Name)
			globalConf, err := readServices(op, r, readServicesOptions{
				collection: result.Services,
				homeDir:    opts.HomeDir,
				rootPath:   opts.RootPath,
				reposPath:  opts.ReposPath,
				overrides:  opts.Overrides,
			})
			if errors.Is(err, fs.ErrNotExist) {
				// No file, do nothing
				opts.Logger.Debugf("registry %s has no %s", r.Name, ServicesFileName)
			} else if err != nil {
				return result, errors.Wrap(err, errors.Meta{
					Reason: fmt.Sprintf("failed to read services from registry %s", r.Name),
					Op:     op,
				})
			}

			opts.Logger.Debugf("Reading playlists from registry %s", r.Name)
			err = readPlaylists(op, r, result.Playlists)
			if errors.Is(err, fs.ErrNotExist) {
				// No file, do nothing
				opts.Logger.Debugf("registry %s has no %s", r.Name, PlaylistsFileName)
			} else if err != nil {
				return result, errors.Wrap(err, errors.Meta{
					Reason: fmt.Sprintf("failed to read playlists from registry %s", r.Name),
					Op:     op,
				})
			}

			result.BaseImages = append(result.BaseImages, globalConf.baseImages...)
			result.LoginStrategies = append(result.LoginStrategies, globalConf.loginStrategies...)
		}
		if opts.ReadApps {
			opts.Logger.Debugf("Reading apps from registry %s", r.Name)
			err := readApps(op, r, readAppsOptions{
				iosCollection:     result.IOSApps,
				desktopCollection: result.DesktopApps,
			})
			if errors.Is(err, fs.ErrNotExist) {
				// No file, do nothing
				opts.Logger.Debugf("registry %s has no %s", r.Name, AppsFileName)
			} else if err != nil {
				return result, errors.Wrap(err, errors.Meta{
					Reason: fmt.Sprintf("failed to read apps from registry %s", r.Name),
					Op:     op,
				})
			}
		}
	}
	result.BaseImages = util.UniqueStrings(result.BaseImages)
	result.LoginStrategies = util.UniqueStrings(result.LoginStrategies)
	return result, nil
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
// If a configuration file does not exist, then the corresponding error will be fs.ErrNotExist.
//
// logger can be provided to allow debug logging of actions while validating.
// If logger is nil, logging is off.
func Validate(path string, strict bool, logger progress.Logger) ValidateResult {
	const op = errors.Op("registry.Validate")
	r := Registry{
		// Name needs to be a proper registry name so just say the org is local
		Name: "local/" + filepath.Base(path),
		Path: path,
	}
	var result ValidateResult
	if logger == nil {
		logger = progress.NoopTracker{}
	}

	// Validate apps.yml
	logger.Debug("Validating apps")
	err := readApps(op, r, readAppsOptions{
		iosCollection:     &app.Collection{},
		desktopCollection: &app.Collection{},
	})
	if err != nil {
		result.AppsErr = err
	}
	// Validate playlists.yml
	logger.Debug("Validating playlists")
	if err := readPlaylists(op, r, &playlist.Collection{}); err != nil {
		result.PlaylistsErr = err
	}

	// Validate services.yml
	logger.Debug("Validating services")
	var services service.Collection
	_, err = readServices(op, r, readServicesOptions{
		collection: &services,
		strict:     strict,
	})
	if err != nil {
		result.ServicesErr = err
	} else {
		// Perform additional validations
		// Keep track of ports to check for conflicting ports
		usedPorts := make(map[string]string)
		var errs errors.List
		it := services.Iter()
		for it.Next() {
			s := it.Value()
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
		if len(errs) > 0 {
			result.ServicesErr = errs
		}
	}
	return result
}
