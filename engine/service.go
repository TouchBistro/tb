// Package progress provides support for displaying progress of
// running actions and providing logging.
package engine

import (
	"context"
	"fmt"
	"os"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/resource/service"
)

// ResolveServices resolves a list of services from the given names.
func (e *Engine) ResolveServices(serviceNames []string) ([]service.Service, error) {
	const op = errors.Op("engine.Engine.ResolveServices")
	services := make([]service.Service, len(serviceNames))
	for i, name := range serviceNames {
		s, err := e.services.Get(name)
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{Reason: "unable to resolve service", Op: op})
		}
		services[i] = s
	}
	return services, nil
}

// ResolvePlaylist resolves a list of services from the given playlist name.
func (e *Engine) ResolvePlaylist(playlistName string) ([]service.Service, error) {
	const op = errors.Op("engine.Engine.ResolvePlaylist")
	serviceNames, err := e.playlists.ServiceNames(playlistName)
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{Reason: "unable to resolve playlist", Op: op})
	}
	return e.ResolveServices(serviceNames)
}

// UpOptions customizes the behaviour of Up.
type UpOptions struct {
	// SkipPreRun skips running the pre-run step for services.
	SkipPreRun bool
	// SkipDockerPull skips pulling both base images and service images if they already exist.
	// If an image is missing however, it will still be pulled so it can be run.
	SkipDockerPull bool
	// SkipGitPull skips pulling existing git repos to update them.
	// Missing repos will still be cloned however.
	SkipGitPull bool
}

// Up performs all necessary actions to prepare services and then starts them.
//
// Up will:
//
// - Stop and remove any services that are already running.
//
// - Pull base images and service images.
//
// - Build any services with mode build.
//
// - Run pre-run steps for services.
func (e *Engine) Up(ctx context.Context, services []service.Service, opts UpOptions) error {
	const op = errors.Op("engine.Engine.Up")
	if len(services) == 0 {
		return errors.New(errkind.Invalid, "no services provided to run", op)
	}

	tracker := progress.TrackerFromContext(ctx)
	tracker.Debug("Preparing Git repos for services")
	if err := e.prepareGitRepos(ctx, op, opts.SkipGitPull); err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to prepare git repos", Op: op})
	}

	serviceNames := make([]string, len(services))
	for i, s := range services {
		serviceNames[i] = s.FullName()
	}

	// Cleanup previous docker state
	err := progress.Run(ctx, progress.RunOptions{
		Message: "Cleaning up previous docker state",
	}, func(ctx context.Context) error {
		return e.stopServices(ctx, op, serviceNames)
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to clean up previous docker state", Op: op})
	}
	tracker.Info("✔ Cleaned up previous docker state")

	// MISSING(@cszatmary): Implement checking disk usage & cleanup

	// Pull base images
	if !opts.SkipDockerPull && len(e.baseImages) > 0 {
		err := progress.RunParallel(ctx, progress.RunParallelOptions{
			Message: "Pulling docker base images",
			Count:   len(e.baseImages),
		}, func(ctx context.Context, i int) error {
			img := e.baseImages[i]
			if err := e.dockerClient.PullImage(ctx, img); err != nil {
				return err
			}
			tracker.Debugf("Pulled base image %s", img)
			return nil
		})
		if err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to pull docker base images", Op: op})
		}
		tracker.Info("✔ Pulled docker base images")
	}

	// Pull service images
	if !opts.SkipDockerPull {
		var images []string
		for _, s := range services {
			if s.Mode == service.ModeRemote {
				images = append(images, s.ImageURI())
			}
		}
		if len(images) > 0 {
			err := progress.RunParallel(ctx, progress.RunParallelOptions{
				Message: "Pulling docker service images",
				Count:   len(images),
			}, func(ctx context.Context, i int) error {
				img := images[i]
				if err := e.dockerClient.PullImage(ctx, img); err != nil {
					return err
				}
				tracker.Debugf("Pulled service image %s", img)
				return nil
			})
			if err != nil {
				return errors.Wrap(err, errors.Meta{Reason: "failed to pull docker service images", Op: op})
			}
			tracker.Info("✔ Pulled docker service images")
		}
	}

	// Build necessary services
	var buildServices []string
	for _, s := range services {
		if s.Mode == service.ModeBuild {
			buildServices = append(buildServices, s.FullName())
		}
	}
	if len(buildServices) > 0 {
		err := progress.Run(ctx, progress.RunOptions{
			Message: "Building docker images for services",
		}, func(ctx context.Context) error {
			return e.composeClient.Build(ctx, serviceNames)
		})
		if err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to build docker images for services", Op: op})
		}
		tracker.Info("✔ Built docker service images")
	}

	// Perform service pre-run
	if !opts.SkipPreRun {
		err := progress.Run(ctx, progress.RunOptions{
			Message: "Performing pre-run step for services (this may take a long time)",
		}, func(ctx context.Context) error {
			// Do this serially since we had issues before when trying to do it in parallel.
			// TODO(@cszatmary): Should scope what the deal was and see if we do these in parallel.
			for _, s := range services {
				if s.PreRun == "" {
					tracker.Debugf("No pre-run for %s, skipping", s.FullName())
					continue
				}

				tracker.Debugf("Running pre-run for %s", s.FullName())
				if err := e.composeClient.Run(ctx, s.FullName(), s.PreRun); err != nil {
					return errors.Wrap(err, errors.Meta{
						Reason: fmt.Sprintf("failed to run pre-run command for %s", s.FullName()),
						Op:     op,
					})
				}
				tracker.Debugf("Ran pre-run for %s", s.FullName())
			}
			return nil
		})
		if err != nil {
			return err
		}
		tracker.Info("✔ Performed pre-run step for services")
	}

	// Start services
	err = progress.Run(ctx, progress.RunOptions{
		Message: "Starting services in the background",
	}, func(ctx context.Context) error {
		return e.composeClient.Up(ctx, serviceNames)
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to start services", Op: op})
	}
	return nil
}

// Down stops services and removes the containers.
// If no services are provided, all currently running services will be stopped.
func (e *Engine) Down(ctx context.Context, services []service.Service) error {
	const op = errors.Op("engine.Engine.Down")
	tracker := progress.TrackerFromContext(ctx)
	tracker.Debug("Preparing Git repos for services")
	// TODO(@cszatmary): Figure out if we actually need this. Would be nice to only
	// have to do this for services being stopped instead of all.
	if err := e.prepareGitRepos(ctx, op, true); err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to prepare git repos", Op: op})
	}

	serviceNames := make([]string, len(services))
	for i, s := range services {
		serviceNames[i] = s.FullName()
	}

	err := progress.Run(ctx, progress.RunOptions{
		Message: "Stopping services",
	}, func(ctx context.Context) error {
		return e.stopServices(ctx, op, serviceNames)
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to stop services", Op: op})
	}
	return nil
}

// prepareGitRepos prepares the git repos for all services. Missing repos will always be cloned
// to ensure that any files referenced in the docker-compose.yml file exist.
// Repos will be pulled if skipPull is false.
func (e *Engine) prepareGitRepos(ctx context.Context, op errors.Op, skipPull bool) error {
	// action determins the type of action to take for a repo. If clone is true, it is cloned,
	// otherwise it is pulled.
	type action struct {
		repo  string
		path  string
		clone bool
	}
	var actions []action
	// Used to remove duplicates since multiple services could use the same repo, so we only
	// want to clone/pull it once
	seenRepos := make(map[string]bool)
	for it := e.services.Iter(); it.Next(); {
		s := it.Value()
		if !s.HasGitRepo() {
			continue
		}
		repo := s.GitRepo.Name
		if seenRepos[repo] {
			continue
		}
		seenRepos[repo] = true

		repoPath := e.resolveRepoPath(repo)
		if !file.Exists(repoPath) {
			actions = append(actions, action{repo, repoPath, true})
			continue
		}

		// Hack to make sure repo was cloned properly
		// Sometimes it doesn't clone properly if the user does control-c during cloning
		// Figure out a better way to do this
		dirlen, err := file.DirLen(repoPath)
		if err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.IO,
				Reason: fmt.Sprintf("could not read directory for git repo %s (%q)", repo, repoPath),
				Op:     op,
			})
		}
		// TODO(@cszatmary): Why 2? Is `.` returned by DirLen? Otherwise should be 1 since only .git
		if dirlen <= 2 {
			// Directory exists but only contains .git subdirectory, rm and clone again below
			if err := os.RemoveAll(repoPath); err != nil {
				return errors.Wrap(err, errors.Meta{
					Kind:   errkind.IO,
					Reason: fmt.Sprintf("could not remove directory for git repo %s (%q)", repo, repoPath),
					Op:     op,
				})
			}
			actions = append(actions, action{repo, repoPath, true})
			continue
		}
		if !skipPull {
			actions = append(actions, action{repo, repoPath, false})
		}
	}
	if len(actions) == 0 {
		return nil
	}

	tracker := progress.TrackerFromContext(ctx)
	err := progress.RunParallel(ctx, progress.RunParallelOptions{
		Message:     "Cloning/pulling service git repos",
		Count:       len(actions),
		Concurrency: e.concurrency,
	}, func(ctx context.Context, i int) error {
		a := actions[i]
		if a.clone {
			tracker.Debugf("Cloning git repo %s", a.repo)
			err := e.gitClient.Clone(ctx, a.path, a.repo)
			if err != nil {
				return errors.Wrap(err, errors.Meta{
					Reason: fmt.Sprintf("failed to clone git repo %s", a.repo),
					Op:     op,
				})
			}
			tracker.Debugf("Cloned git repo %s", a.repo)
			return nil
		}
		tracker.Debugf("Pulling git repo %s", a.repo)
		err := e.gitClient.Pull(ctx, a.path)
		if err != nil {
			return errors.Wrap(err, errors.Meta{
				Reason: fmt.Sprintf("failed to pull git repo %s", a.repo),
				Op:     op,
			})
		}
		tracker.Debugf("Pulled git repo %s", a.repo)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to prepare service git repos", Op: op})
	}
	tracker.Info("✔ Finished preparing service git repos")
	return nil
}

// stopServices stops and removes any containers for the given services.
func (e *Engine) stopServices(ctx context.Context, op errors.Op, serviceNames []string) error {
	tracker := progress.TrackerFromContext(ctx)
	if err := e.composeClient.Stop(ctx, serviceNames); err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to stop running containers", Op: op})
	}
	tracker.Debug("Stopped service containers")
	if err := e.composeClient.Rm(ctx, serviceNames); err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to remove stopped containers", Op: op})
	}
	tracker.Debug("Removed service containers")
	return nil
}
