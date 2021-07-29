package engine

import (
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/integrations/git"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
)

// Engine provides the API for performing actions on services, playlists, and apps.
type Engine struct {
	workdir     string // Path to root dir where data is stored
	services    *service.Collection
	playlists   *playlist.Collection
	baseImages  []string
	concurrency uint

	gitClient     git.Git
	dockerClient  docker.Docker
	composeClient docker.Compose
}

// Options allows for configuring an Engine instance created by New.
type Options struct {
	// Workdir is the working directory on the OS filesystem where the engine can store data.
	// Defaults to ~/.tb if omitted.
	Workdir string
	// Services is the collection of services that the Engine can manage.
	// If no value is provided, then there will be no services available to use.
	Services *service.Collection
	// Services is the collection of playlists that the Engine can manage.
	// If no value is provided, then there will be no playlists available to use.
	Playlists *playlist.Collection
	// BaseImages is a list of docker base images that will be pulled before building images.
	// If no value is provided, no base images will be pulled.
	BaseImages []string
	// Concurrency controls how many goroutines can run concurrently.
	// Defaults to runtime.NumCPU if omitted.
	Concurrency uint
}

// New creates a new Engine instance.
func New(opts Options) (*Engine, error) {
	const op = errors.Op("engine.New")

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

	// Initialize clients
	dockerClient, err := docker.New()
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{Op: op})
	}

	return &Engine{
		workdir:       opts.Workdir,
		services:      opts.Services,
		playlists:     opts.Playlists,
		baseImages:    opts.BaseImages,
		concurrency:   opts.Concurrency,
		gitClient:     git.New(),
		dockerClient:  dockerClient,
		composeClient: docker.NewCompose(opts.Workdir),
	}, nil
}

// resolveRepoPath resolve the absolute path to where the repo is stored on the OS filesystem.
func (e *Engine) resolveRepoPath(repo string) string {
	return filepath.Join(e.workdir, "repos", repo)
}
