package engine

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/integrations/git"
	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/TouchBistro/tb/integrations/storage"
	"github.com/TouchBistro/tb/resource/app"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
)

// Engine provides the API for performing actions on services, playlists, and apps.
type Engine struct {
	workdir          string // Path to root dir where data is stored
	experimentalMode bool
	services         *service.Collection
	playlists        *playlist.Collection
	iosApps          *app.Collection
	desktopApps      *app.Collection
	baseImages       []string
	deviceList       simulator.DeviceList
	concurrency      uint

	gitClient        git.Git
	dockerClient     *docker.Docker
	composeClient    docker.Compose
	storageProviders map[string]storage.Provider // cached providers for reuse
}

// Options allows for configuring an Engine instance created by New.
type Options struct {
	// Workdir is the working directory on the OS filesystem where the engine can store data.
	// Defaults to ~/.tb if omitted.
	Workdir string
	// ExperimentalMode controls if experimental mode is enabled, which gives access
	// to new features that aren't generally available.
	ExperimentalMode bool
	// Services is the collection of services that the Engine can manage.
	// If no value is provided, then there will be no services available to use.
	Services *service.Collection
	// Services is the collection of playlists that the Engine can manage.
	// If no value is provided, then there will be no playlists available to use.
	Playlists *playlist.Collection
	// IOSApps is the collection of iOS applications that the Engine can manage.
	// If no value is provided, then there will be no apps available to use.
	IOSApps *app.Collection
	// IOSApps is the collection of desktop applications that the Engine can manage.
	// If no value is provided, then there will be no apps available to use.
	DesktopApps *app.Collection
	// BaseImages is a list of docker base images that will be pulled before building images.
	// If no value is provided, no base images will be pulled.
	BaseImages []string
	// DeviceList is the list of iOS devices to use for running iOS apps.
	// If no value is provided, no devices will be available to use.
	DeviceList simulator.DeviceList
	// Concurrency controls how many goroutines can run concurrently.
	// Defaults to runtime.NumCPU if omitted.
	Concurrency uint
	// GitClient is the client to use for git operations.
	// This allows for overriding the default git client if provided.
	GitClient git.Git
	// DockerAPIClient is the client to use for docker operations.
	// This allows for overriding the default docker client if provided.
	DockerAPIClient docker.APIClient
	// ComposeClient is the client to use for docker-compose operations.
	// This allows for overriding the default docker-compose client if provided.
	ComposeClient docker.Compose
}

// New creates a new Engine instance.
func New(opts Options) (*Engine, error) {
	const op = errors.Op("engine.New")
	const projectName = "tb"

	// Set defaults
	if opts.Workdir == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{
				Kind:   errkind.Internal,
				Reason: "unable to find user home directory",
				Op:     op,
			})
		}
		opts.Workdir = filepath.Join(homedir, ".tb")
	}
	if opts.Services == nil {
		opts.Services = &service.Collection{}
	}
	if opts.Playlists == nil {
		opts.Playlists = &playlist.Collection{}
	}
	if opts.IOSApps == nil {
		opts.IOSApps = &app.Collection{}
	}
	if opts.DesktopApps == nil {
		opts.DesktopApps = &app.Collection{}
	}

	// Initialize clients
	if opts.GitClient == nil {
		opts.GitClient = git.New()
	}
	if opts.DockerAPIClient == nil {
		apiClient, err := docker.NewAPIClient()
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{Op: op})
		}
		opts.DockerAPIClient = apiClient
	}
	if opts.ComposeClient == nil {
		opts.ComposeClient = docker.NewCompose(opts.Workdir, projectName)
	}

	return &Engine{
		workdir:          opts.Workdir,
		services:         opts.Services,
		playlists:        opts.Playlists,
		iosApps:          opts.IOSApps,
		desktopApps:      opts.DesktopApps,
		baseImages:       opts.BaseImages,
		deviceList:       opts.DeviceList,
		concurrency:      opts.Concurrency,
		gitClient:        opts.GitClient,
		dockerClient:     docker.New(opts.DockerAPIClient, projectName),
		composeClient:    opts.ComposeClient,
		storageProviders: make(map[string]storage.Provider),
	}, nil
}

// ExperimentalMode returns whether or not experimental mode is enabled.
func (e *Engine) ExperimentalMode() bool {
	return e.experimentalMode
}

// Paths used to store data under workdir.
const (
	reposDir      = "repos"
	iosDir        = "ios"
	desktopDir    = "desktop"
	registriesDir = "registries"
)

// getStorageProvider returns a storage.Provider for the given provider name.
func (e *Engine) getStorageProvider(providerName string) (storage.Provider, error) {
	if p, ok := e.storageProviders[providerName]; ok {
		return p, nil
	}
	p, err := storage.NewProvider(providerName)
	if err != nil {
		return nil, err
	}
	e.storageProviders[providerName] = p
	return p, nil
}
