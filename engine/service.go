// Package progress provides support for displaying progress of
// running actions and providing logging.
package engine

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/integrations/login"
	"github.com/TouchBistro/tb/resource/service"
	"gopkg.in/yaml.v3"
)

// ResolveService resolves a single service from the given name.
func (e *Engine) ResolveService(serviceName string) (service.Service, error) {
	s, err := e.services.Get(serviceName)
	if err != nil {
		return s, errors.Wrap(err, errors.Meta{
			Reason: "unable to resolve service",
			Op:     "engine.Engine.ResolveService",
		})
	}
	return s, nil
}

// UpOptions customizes the behaviour of Up.
type UpOptions struct {
	// ServiceNames is a list of services names to start.
	ServiceNames []string
	// ServiceTags is a map of service:image-tag to start.
	ServiceTags map[string]string
	// PlaylistName is the name of a playlist to start.
	PlaylistName string
	// SkipPreRun skips running the pre-run step for services.
	SkipPreRun bool
	// SkipDockerPull skips pulling both base images and service images if they already exist.
	// If an image is missing however, it will still be pulled so it can be run.
	SkipDockerPull bool
	// SkipGitPull skips pulling existing git repos to update them.
	// Missing repos will still be cloned however.
	SkipGitPull bool
	// OfflineMode skips login strategies and pulling remote images
	OfflineMode bool
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
//
// Exactly one of opts.ServiceNames or opts.PlaylistName must be provided to determine
// which services to start.
func (e *Engine) Up(ctx context.Context, opts UpOptions) error {
	const op = errors.Op("engine.Engine.Up")
	services, err := e.resolveServices(op, opts.ServiceNames, opts.PlaylistName, opts.ServiceTags, true)
	if err != nil {
		return err
	}
	if err := e.prepareGitRepos(ctx, op, opts.SkipGitPull || opts.OfflineMode); err != nil {
		return err
	}
	if err := e.writeComposeFile(ctx, op); err != nil {
		return err
	}

	tracker := progress.TrackerFromContext(ctx)
	if len(e.loginStrategies) > 0 && !opts.OfflineMode {
		loginStrategies := make([]login.Strategy, len(e.loginStrategies))
		for i, name := range e.loginStrategies {
			s, err := login.ParseStrategy(name)
			if err != nil {
				return errors.Wrap(err, errors.Meta{Op: op})
			}
			loginStrategies[i] = s
		}

		err := progress.RunParallel(ctx, progress.RunParallelOptions{
			Message:     "Logging into services",
			Count:       len(loginStrategies),
			Concurrency: e.concurrency,
			Timeout:     e.timeout,
			// Bail if one fails since there's no point on waiting on the others
			// since we can't proceed anyway.
			CancelOnError: true,
		}, func(ctx context.Context, i int) error {
			ls := loginStrategies[i]
			tracker := progress.TrackerFromContext(ctx)
			tracker.Debugf("Logging into %s", ls.Name())
			if err := ls.Login(ctx); err != nil {
				return err
			}
			tracker.Debugf("Logged into %s", ls.Name())
			return nil
		})
		if err != nil {
			return err
		}
		tracker.Debug("Finished logging into services")
	}

	// Cleanup previous docker state
	err = progress.Run(ctx, progress.RunOptions{
		Message: "Cleaning up previous docker state",
		Timeout: e.timeout,
	}, func(ctx context.Context) error {
		return e.stopServices(ctx, op, services)
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to clean up previous docker state", Op: op})
	}
	tracker.Info("✔ Cleaned up previous docker state")

	// Pull base images
	if !opts.SkipDockerPull && !opts.OfflineMode && len(e.baseImages) > 0 {
		err := progress.RunParallel(ctx, progress.RunParallelOptions{
			Message:     "Pulling docker base images",
			Count:       len(e.baseImages),
			Concurrency: e.concurrency,
			Timeout:     e.timeout,
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
	if !opts.SkipDockerPull && !opts.OfflineMode {
		var images []string
		for _, s := range services {
			if s.Mode == service.ModeRemote {
				images = append(images, s.ImageURI())
			}
		}
		if len(images) > 0 {
			err := progress.RunParallel(ctx, progress.RunParallelOptions{
				Message:     "Pulling docker service images",
				Count:       len(images),
				Concurrency: e.concurrency,
				Timeout:     e.timeout,
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
			Timeout: e.timeout,
		}, func(ctx context.Context) error {
			return e.dockerClient.BuildServices(ctx, buildServices)
		})
		if err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to build docker images for services", Op: op})
		}
		tracker.Info("✔ Built docker service images")
	}

	// Perform service pre-run
	if !opts.SkipPreRun {
		// Do this serially since we had issues before when trying to do it in parallel.
		// TODO(@cszatmary): Should scope what the deal was and see if we do these in parallel.
		// We might need to rethink the whole way pre-run works.
		err := progress.Run(ctx, progress.RunOptions{
			Message: "Performing pre-run step for services (this may take a long time)",
			Count:   len(services),
			Timeout: e.timeout,
		}, func(ctx context.Context) error {
			for _, s := range services {
				if s.PreRun == "" {
					tracker.Debugf("No pre-run for %s, skipping", s.FullName())
					tracker.Inc()
					continue
				}

				tracker.Debugf("Running pre-run for %s", s.FullName())
				if err := e.dockerClient.RunService(ctx, s.FullName(), s.PreRun); err != nil {
					return errors.Wrap(err, errors.Meta{
						Reason: fmt.Sprintf("failed to run pre-run command for %s", s.FullName()),
						Op:     op,
					})
				}
				tracker.Debugf("Ran pre-run for %s", s.FullName())
				tracker.Inc()
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
		Timeout: e.timeout,
	}, func(ctx context.Context) error {
		return e.dockerClient.UpServices(ctx, getServiceNames(services))
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to start services", Op: op})
	}
	return nil
}

// DownOptions customizes the behaviour of Down.
type DownOptions struct {
	// ServiceNames is a list of services names to stop.
	// If empty, all currently running services will be stopped.
	ServiceNames []string
}

// Down stops services and removes the containers.
func (e *Engine) Down(ctx context.Context, opts DownOptions) error {
	const op = errors.Op("engine.Engine.Down")
	services, err := e.resolveServices(op, opts.ServiceNames, "", make(map[string]string), false)
	if err != nil {
		return err
	}
	err = progress.Run(ctx, progress.RunOptions{
		Message: "Stopping services",
		Timeout: e.timeout,
	}, func(ctx context.Context) error {
		return e.stopServices(ctx, op, services)
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to stop services", Op: op})
	}
	return nil
}

// LogsOptions customizes the behaviour of Logs.
type LogsOptions struct {
	// ServiceNames is a list of services names for which to retrieve logs.
	// If empty, logs will be listed for all services.
	ServiceNames []string
	// Follow follows the log output. It shows new logs in real time.
	Follow bool
	// Tail is the number of lines to show from the end of the logs.
	// A value of -1 means show all logs.
	Tail int
}

// Logs retrieves the logs from one or more service containers and writes it to w.
func (e *Engine) Logs(ctx context.Context, w io.Writer, opts LogsOptions) error {
	const op = errors.Op("engine.Engine.Logs")
	services, err := e.resolveServices(op, opts.ServiceNames, "", make(map[string]string), false)
	if err != nil {
		return err
	}
	err = e.dockerClient.LogsFromServices(ctx, docker.LogsFromServicesOptions{
		ServiceNames: getServiceNames(services),
		Out:          w,
		Follow:       opts.Follow,
		Tail:         opts.Tail,
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to view logs", Op: op})
	}
	return nil
}

// ExecOptions customizes the behaviour of Exec.
type ExecOptions struct {
	// Cmd is the command to execute. It must have at
	// least one element which is the name of the command.
	// Any additional elements are args for the command.
	Cmd    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Exec executes a command in a service container and returns the exit code.
// If the exit code cannot be determined, -1 will be returned.
//
// The returned error will be non-nil if an error occurred while trying to perform execution
// of the command. If the command itself exits with a non-zero code, err will be nil.
func (e *Engine) Exec(ctx context.Context, serviceName string, opts ExecOptions) (int, error) {
	const op = errors.Op("engine.Engine.Exec")
	if len(opts.Cmd) == 0 {
		panic("ExecOptions.Cmd must have at least one element")
	}
	s, err := e.services.Get(serviceName)
	if err != nil {
		return -1, errors.Wrap(err, errors.Meta{Reason: "unable to resolve service", Op: op})
	}
	exitCode, err := e.dockerClient.ExecInService(ctx, s.FullName(), docker.ExecInServiceOptions{
		Cmd:    opts.Cmd,
		Stdin:  opts.Stdin,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	})
	if err != nil {
		return -1, errors.Wrap(err, errors.Meta{Reason: "failed to execute command in service", Op: op})
	}
	return exitCode, nil
}

// ListOptions customizes the behaviour of list.
type ListOptions struct {
	ListServices        bool
	ListPlaylists       bool
	ListCustomPlaylists bool
	// TreeMode causes playlists to be listed along with all their services.
	TreeMode bool
}

type ListResult struct {
	Services        []string
	Playlists       []PlaylistSummary
	CustomPlaylists []PlaylistSummary
}

// PlaylistSummary provides a summary of a playlist produced by List.
type PlaylistSummary struct {
	Name     string
	Services []string
}

func (e *Engine) List(opts ListOptions) ListResult {
	var lr ListResult
	if opts.ListServices {
		for it := e.services.Iter(); it.Next(); {
			lr.Services = append(lr.Services, it.Value().FullName())
		}
	}
	if opts.ListPlaylists {
		lr.Playlists = e.listPlaylists(e.playlists.Names(), opts.TreeMode)
	}
	if opts.ListCustomPlaylists {
		lr.CustomPlaylists = e.listPlaylists(e.playlists.CustomNames(), opts.TreeMode)
	}
	return lr
}

func (e *Engine) listPlaylists(names []string, tree bool) []PlaylistSummary {
	var summaries []PlaylistSummary
	for _, n := range names {
		summary := PlaylistSummary{Name: n}
		if tree {
			list, err := e.playlists.ServiceNames(n)
			if err != nil {
				// If we get an error here we have a bug since n has to be a valid service name.
				panic(err)
			}
			summary.Services = list
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

// NukeOptions customizes the behaviour of Nuke.
type NukeOptions struct {
	// RemoveContainers specifies to remove all service containers.
	RemoveContainers bool
	// RemoveImages specifies to remove all service images.
	// This option also implies RemoveContainers.
	RemoveImages bool
	// RemoveNetworks specifies to remove all tb networks.
	// This option also implies RemoveContainers.
	RemoveNetworks bool
	// RemoveVolumes specifies to remove all service volumes.
	// This option also implies RemoveContainers.
	RemoveVolumes bool
	// RemoveRepos specifies to remove all service repos.
	// This option also implies RemoveContainers.
	RemoveRepos bool
	// RemoveDesktopApps specifies to remove all downloaded desktop apps.
	RemoveDesktopApps bool
	// RemoveiOSApps specifies to remove all downloaded iOS apps.
	RemoveiOSApps bool
	// RemoveRegistries specifies to remove all cloned registries.
	RemoveRegistries bool
}

// Nuke cleans up resources based on the given options. Nuke only touches resources
// created by tb with the exception of images as dangling images will also be removed.
func (e *Engine) Nuke(ctx context.Context, opts NukeOptions) error {
	const op = errors.Op("engine.Engine.Nuke")
	return progress.Run(ctx, progress.RunOptions{
		Message: "Cleaning up tb data",
		Timeout: e.timeout,
	}, func(ctx context.Context) error {
		return e.nuke(ctx, opts, op)
	})
}

func (e *Engine) nuke(ctx context.Context, opts NukeOptions, op errors.Op) error {
	tracker := progress.TrackerFromContext(ctx)

	// Remove containers if any docker resources were specified to be removed since
	// they can't be removed if they are being used by containers.
	if opts.RemoveContainers || opts.RemoveImages || opts.RemoveNetworks || opts.RemoveVolumes {
		tracker.UpdateMessage("Stopping running containers")
		if err := e.dockerClient.StopContainers(ctx); err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to stop docker containers", Op: op})
		}
		tracker.Info("✔ Stopped running containers")
		tracker.UpdateMessage("Removing docker containers")
		if err := e.dockerClient.RemoveContainers(ctx); err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to remove docker containers", Op: op})
		}
		tracker.Info("✔ Removed docker containers")
	}

	if opts.RemoveImages {
		var imageSearches []docker.ImageSearch
		for it := e.services.Iter(); it.Next(); {
			s := it.Value()
			// Search for both remote and locally built images since the user might have switched
			// between build and remote mode in their tbrc.
			if s.Remote.Image != "" {
				imageSearches = append(imageSearches, docker.ImageSearch{Name: s.Remote.Image})
			}
			if s.CanBuild() {
				imageSearches = append(imageSearches, docker.ImageSearch{Name: s.FullName(), LocalBuild: true})
			}
		}
		for _, bi := range e.baseImages {
			imageSearches = append(imageSearches, docker.ImageSearch{Name: bi})
		}
		tracker.UpdateMessage("Removing docker images")
		if err := e.dockerClient.RemoveImages(ctx, imageSearches); err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to remove docker images", Op: op})
		}
		tracker.Info("✔ Removed docker images")

		// Also prune images to clean up space for users
		tracker.UpdateMessage("Pruning docker images")
		if err := e.dockerClient.PruneImages(ctx); err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to prune docker images", Op: op})
		}
		tracker.Info("✔ Pruned unused docker images")
	}

	if opts.RemoveNetworks {
		tracker.UpdateMessage("Removing docker networks")
		if err := e.dockerClient.RemoveNetworks(ctx); err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to remove docker networks", Op: op})
		}
		tracker.Info("✔ Removed docker networks")
	}

	if opts.RemoveVolumes {
		tracker.UpdateMessage("Removing docker volumes")
		if err := e.dockerClient.RemoveVolumes(ctx); err != nil {
			return errors.Wrap(err, errors.Meta{Reason: "failed to remove docker volumes", Op: op})
		}
		tracker.Info("✔ Removed docker volumes")
	}

	type directory struct {
		name string
		path string
	}
	var removeDirs []directory
	if opts.RemoveRepos {
		removeDirs = append(removeDirs, directory{
			name: "cloned repos",
			path: filepath.Join(e.workdir, reposDir),
		})
	}
	if opts.RemoveDesktopApps {
		removeDirs = append(removeDirs, directory{
			name: "desktop apps",
			path: filepath.Join(e.workdir, desktopDir),
		})
	}
	if opts.RemoveiOSApps {
		removeDirs = append(removeDirs, directory{
			name: "iOS apps",
			path: filepath.Join(e.workdir, iosDir),
		})
	}
	if opts.RemoveRegistries {
		removeDirs = append(removeDirs, directory{
			name: "cloned registries",
			path: filepath.Join(e.workdir, registriesDir),
		})
	}
	for _, dir := range removeDirs {
		tracker.UpdateMessage(fmt.Sprintf("Removing %s", dir.name))
		if err := os.RemoveAll(dir.path); err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.IO,
				Reason: fmt.Sprintf("failed to remove %s", dir.path),
				Op:     op,
			})
		}
		tracker.Infof("✔ Removed %s", dir.name)
	}

	// Check workdir and remove any files/dirs that shouldn't be there.
	tracker.UpdateMessage("Removing any remaining files")
	items, err := os.ReadDir(e.workdir)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to read directory %s", e.workdir),
			Op:     op,
		})
	}
	for _, item := range items {
		// Filter out ones tb manages so they don't get removed in case those
		// options weren't specified. If they were specified to be removed
		// they would have already been removed above.
		switch item.Name() {
		case reposDir, iosDir, desktopDir, registriesDir:
			continue
		}
		p := filepath.Join(e.workdir, item.Name())
		if err := os.RemoveAll(p); err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.IO,
				Reason: fmt.Sprintf("failed to remove %s", p),
				Op:     op,
			})
		}
	}
	return nil
}

// resolveServices resolves a list of services from either a list of service names or a playlist name.
//
// If both serviceNames and playlistName are provided, an error will be returned. Mixing service names
// is not supported.
//
// If neither serviceNames nor playlistName are provided, then behaviour depends on the value of requireOne.
// In this case, if requireOne is true, an error will be returned since at least one of serviceNames or playlistName
// was required. Otherwise, both the returned slice and error will be nil, which can be treated as an empty slice
// of services.
func (e *Engine) resolveServices(op errors.Op, serviceNames []string, playlistName string, serviceTags map[string]string, requireOne bool) ([]service.Service, error) {
	if len(serviceNames) > 0 && playlistName != "" {
		return nil, errors.New(errkind.Invalid, "both service names and playlist name provided", op)
	}
	if len(serviceNames) > 0 {
		for service, tag := range serviceTags {
			_, err := e.services.Get(service)
			if err != nil {
				return nil, errors.Wrap(err, errors.Meta{Reason: fmt.Sprintf("unable to resolve service %s with tag %s", service, tag), Op: op})
			}
		}

		services := make([]service.Service, len(serviceNames))
		for i, name := range serviceNames {
			s, err := e.services.Get(name)
			if err != nil {
				return nil, errors.Wrap(err, errors.Meta{Reason: "unable to resolve service", Op: op})
			}

			services[i] = s
			tag := serviceTags[s.Name]
			if len(tag) > 0 {
				override := service.ServiceOverride{
					Mode: "remote",
					Remote: service.RemoteOverride{
						Tag: tag,
					},
				}
				overridenService, err := service.Override(s, override)
				if err != nil {
					return nil, errors.Wrap(err, errors.Meta{Reason: fmt.Sprintf("failed to override service %s with tag %s", s.Name, tag), Op: op})
				}

				/* e.services is the global list of services parsed from the registry
				 * we need to update its state with the given remote tag
				 * since it's used downstream to generate the docker-compose config to tag the service
				 */
				if err := e.services.Set(overridenService); err != nil {
					return nil, errors.Wrap(err, errors.Meta{Reason: fmt.Sprintf("failed to override service %s with tag %s", s.Name, tag), Op: op})
				}

				services[i] = overridenService
			}
		}
		return services, nil
	}
	if playlistName != "" {
		serviceNames, err := e.playlists.ServiceNames(playlistName)
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{Reason: "unable to resolve playlist", Op: op})
		}
		// Can just run resolveServices again with the service names to get the actual services.
		return e.resolveServices(op, serviceNames, "", serviceTags, true)
	}
	if requireOne {
		return nil, errors.New(errkind.Invalid, "neither service names nor playlist name was provided", op)
	}
	// nil will be treated as an empty slice, which is fine since the caller said that no services is ok.
	return nil, nil
}

// prepareGitRepos prepares the git repos for all services. Missing repos will always be cloned
// to ensure that any files referenced in the docker-compose.yml file exist.
// Repos will be pulled if skipPull is false.
func (e *Engine) prepareGitRepos(ctx context.Context, op errors.Op, skipPull bool) error {
	tracker := progress.TrackerFromContext(ctx)
	tracker.Debug("Preparing Git repos for services")

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

		repoPath := filepath.Join(e.workdir, reposDir, repo)
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

	err := progress.RunParallel(ctx, progress.RunParallelOptions{
		Message:     "Cloning/pulling service git repos",
		Count:       len(actions),
		Concurrency: e.concurrency,
		Timeout:     e.timeout,
	}, func(ctx context.Context, i int) error {
		a := actions[i]
		if a.clone {
			tracker.Debugf("Cloning git repo %s", a.repo)
			err := e.gitClient.Clone(ctx, a.repo, a.path)
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

func (e *Engine) writeComposeFile(ctx context.Context, op errors.Op) error {
	tracker := progress.TrackerFromContext(ctx)
	tracker.Debug("Generating docker-compose.yml file")
	composePath := filepath.Join(e.workdir, docker.ComposeFilename)
	f, err := os.OpenFile(composePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to open file %s", composePath),
			Op:     op,
		})
	}
	defer f.Close()

	const header = "# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY\n\n"
	if _, err := io.WriteString(f, header); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: "failed to write header comment to docker-compose yaml file",
			Op:     op,
		})
	}

	composeConfig := service.ComposeConfig(e.services)
	if err := yaml.NewEncoder(f).Encode(composeConfig); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: "failed to encode docker-compose struct to yaml",
			Op:     op,
		})
	}
	tracker.Debug("Successfully generated docker-compose.yml")
	return nil
}

// stopServices stops and removes any containers for the given services.
func (e *Engine) stopServices(ctx context.Context, op errors.Op, services []service.Service) error {
	serviceNames := getServiceNames(services)
	if err := e.dockerClient.StopContainers(ctx, serviceNames...); err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to stop running containers", Op: op})
	}
	tracker := progress.TrackerFromContext(ctx)
	tracker.Debug("Stopped service containers")
	if err := e.dockerClient.RemoveContainers(ctx, serviceNames...); err != nil {
		return errors.Wrap(err, errors.Meta{Reason: "failed to remove stopped containers", Op: op})
	}
	tracker.Debug("Removed service containers")
	return nil
}

func getServiceNames(services []service.Service) []string {
	sn := make([]string, len(services))
	for i, s := range services {
		sn[i] = s.FullName()
	}
	return sn
}
