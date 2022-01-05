// Package docker provides functionality for working with docker in the context of tb.
package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
	dockerconfig "github.com/docker/cli/cli/config"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
)

// These are the states we care about.
// There are others but they are not used by tb.
const (
	// ContainerStateCreated indicates the container has been created but was not started.
	ContainerStateCreated = "Created"
	// ContainerStateRunning indicates the container is currently running.
	ContainerStateRunning = "Running"
	// ContainerStateExited indicates the container exited and is not running.
	ContainerStateExited = "Exited"
)

// Docker labels for use in lookups
const (
	// ProjectLabel is a docker label that specifies the compose project.
	ProjectLabel = "com.docker.compose.project"
)

// NormalizeName normalizes name to make it compatible with docker.
// It replaces slashes with dashes and converts upper case letters to lower case.
func NormalizeName(name string) string {
	// docker does not allow slashes in container names
	// so we'll replace them with dashes
	s := strings.ReplaceAll(name, "/", "-")
	// docker does not allow upper case letters in image names
	// need to convert it all to lower case or docker-compose build breaks
	return strings.ToLower(s)
}

// NOTE: The docs for the Go SDK aren't great. If you are working on this and want to
// better understand how the docker APIs work check out the actual engine API docs
// which have the OpenAPI schema.
// https://docs.docker.com/engine/api/latest

// APIClient is the functionality required to be implemented by
// a client that communicates with the docker API.
type APIClient interface {
	client.APIClient
	ComposeAPIClient
}

// apiClient is an APIClient implementation.
type apiClient struct {
	// Wrap an client.APIClient to implement the APIClient interface
	client.APIClient
}

// Config provides functionality for working the docker config.
type Config interface {
	GetAuthConfig(registryHostname string) (configtypes.AuthConfig, error)
}

// Docker provides functionality for working with docker resources.
type Docker struct {
	project   ComposeProject
	apiClient APIClient

	config                 Config // docker config; for registry auth
	defaultRegistryAddress string // used to resolve creds for dockerhub
}

// Options allows for customizing a Docker instance created with New.
// All fields are optional and will be defaulted if omitted.
type Options struct {
	// APIClient is the APIClient instance to use for making docker API calls.
	// If omitted, a default one will be used that communicates with the real  docker API.
	APIClient APIClient
	// Config is the docker config to use to resolve things like registry auth.
	// If omitted, the default docker config will be loaded.
	Config Config
}

// New returns a new Docker instance that provides docker functionality for tb.
// projectName is a compose project name and is used to filter docker resources.
// Only docker resources belonging to this project will be modified.
func New(projectName, workdir string, opts Options) (*Docker, error) {
	const op = errors.Op("docker.New")
	if opts.APIClient == nil {
		// Use the actual docker SDK client for making real requests.
		dockerAPIClient, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{
				Kind:   errkind.Docker,
				Reason: "unable to create docker API client",
				Op:     op,
			})
		}
		opts.APIClient = &apiClient{dockerAPIClient}
	}
	if opts.Config == nil {
		// If no config provided, load the default docker config.
		configFile, err := dockerconfig.Load(dockerconfig.Dir())
		if err != nil {
			return nil, errors.Wrap(err, errors.Meta{
				Kind:   errkind.Docker,
				Reason: "failed to load docker config file",
				Op:     op,
			})
		}
		opts.Config = configFile
	}
	return &Docker{
		project: ComposeProject{
			Name:    projectName,
			Workdir: workdir,
		},
		apiClient: opts.APIClient,
		config:    opts.Config,
	}, nil
}

func (d *Docker) getDefaultRegistryAddress(ctx context.Context) string {
	// Check if cached and use that.
	if d.defaultRegistryAddress != "" {
		return d.defaultRegistryAddress
	}

	// Use the docker api to look up the default registry address.
	// This is how the docker cli and docker compose do it.
	tracker := progress.TrackerFromContext(ctx)
	info, err := d.apiClient.Info(ctx)
	if err != nil {
		// If there is an error just fallback to the default, but log so users know.
		tracker.WithFields(progress.Fields{
			"error": err,
		}).Warnf("Failed to get the default registry endpoint from the docker API. Using system default: %s", registry.IndexServer)
		info.IndexServerAddress = registry.IndexServer
	} else if info.IndexServerAddress == "" {
		// Apparently older versions of docker can have this missing.
		// This is unlikely to happen but handle it anyway since this is what docker itself does.
		tracker.Warnf("Docker API returned empty default registry endpoint. Using system default: %s", registry.IndexServer)
		info.IndexServerAddress = registry.IndexServer
	}

	// Cache it so we don't need to get it from the docker api every time.
	d.defaultRegistryAddress = info.IndexServerAddress
	return d.defaultRegistryAddress
}

// StopContainers stops containers matching the given service names.
// If no names are provided, all containers part of the project will be stopped.
func (d *Docker) StopContainers(ctx context.Context, serviceNames ...string) error {
	const op = errors.Op("docker.Docker.StopContainers")
	containers, err := d.listContainers(ctx, serviceNames, false, op)
	if err != nil {
		return err
	}

	tracker := progress.TrackerFromContext(ctx)
	timeout := 5 * time.Second
	for _, container := range containers {
		tracker.Debugf("Stopping container %s", container.Names[0])
		if err := d.apiClient.ContainerStop(ctx, container.ID, &timeout); err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.Docker,
				Reason: fmt.Sprintf("failed to stop container %s, %s", container.Names[0], container.ID),
				Op:     op,
			})
		}
	}
	return nil
}

// RemoveContainers removes containers matching the given service names.
// If no names, all containers part of the project will be removed.
func (d *Docker) RemoveContainers(ctx context.Context, serviceNames ...string) error {
	const op = errors.Op("docker.Docker.RemoveContainers")
	containers, err := d.listContainers(ctx, serviceNames, true, op)
	if err != nil {
		return err
	}

	tracker := progress.TrackerFromContext(ctx)
	for _, container := range containers {
		tracker.Debugf("Removing container %s", container.Names[0])
		if err := d.apiClient.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{}); err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.Docker,
				Reason: fmt.Sprintf("failed to remove container %s, %s", container.Names[0], container.ID),
				Op:     op,
			})
		}
	}
	return nil
}

// listContainers lists containers belonging to the project. If names is provided it will be used to filter
// the returned containers to only those matching the names.
func (d *Docker) listContainers(ctx context.Context, serviceNames []string, stopped bool, op errors.Op) ([]types.Container, error) {
	f := filters.NewArgs(projectFilter(d.project.Name))
	if len(serviceNames) > 0 {
		for _, n := range serviceNames {
			f.Add("name", NormalizeName(n))
		}
	}
	containers, err := d.apiClient.ContainerList(ctx, types.ContainerListOptions{
		All:     stopped,
		Filters: f,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "failed to list containers",
			Op:     op,
		})
	}
	return containers, nil
}

// PullImage pulls the specified image from a remote registry.
// imageName must be a valid image name either in normalized for or familiar form.
//
// PullImage automatically resolves authentication for the remote registry based
// on the authentication supplied via `docker login`.
func (d *Docker) PullImage(ctx context.Context, imageName string) error {
	const op = errors.Op("docker.Docker.PullImage")

	// First, we need to validate the image name and resolve the registry.
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: fmt.Sprintf("failed to parse image name %s", imageName),
			Op:     op,
		})
	}
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: fmt.Sprintf("failed to resolve repository info for image %s", imageName),
			Op:     op,
		})
	}
	// Need to get the registry name (i.e. the server address) which we will use to resolve auth.
	registryKey := repoInfo.Index.Name
	if repoInfo.Index.Official {
		// If it is an official index (i.e. dockerhub), we registry name is not actually the
		// address we want, so we get the default one.
		// No idea why it works this way, but this is how it is.
		registryKey = d.getDefaultRegistryAddress(ctx)
	}

	// Second, we need to resolve the auth for the registry.
	authConfig, err := d.config.GetAuthConfig(registryKey)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: fmt.Sprintf("failed to get auth config for registry %s", registryKey),
			Op:     op,
		})
	}

	// The registry auth needs to be passed as a base64 URL encoded JSON string.
	// This comes straight from this example:
	// https://docs.docker.com/engine/api/sdk/examples/#pull-an-image-with-authentication
	b, err := json.Marshal(authConfig)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: "failed to marshal auth config as json",
			Op:     op,
		})
	}

	// Finally we can pull the image!
	tracker := progress.TrackerFromContext(ctx)
	tracker.Debugf("Pulling image: %s", ref)
	r, err := d.apiClient.ImagePull(ctx, imageName, types.ImagePullOptions{
		RegistryAuth: base64.URLEncoding.EncodeToString(b),
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: fmt.Sprintf("failed to pull image %s", imageName),
			Op:     op,
		})
	}
	defer r.Close()

	// ImagePull returns an io.Reader which will contain details on the progress of pulling images.
	// We can display this in debug mode to get information on the pull progress equivalent to if
	// the user had run `docker pull`.
	// Only do it for debug though because it is really noisy.
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"op": op}).Debug)
	defer w.Close()

	// The docker SDK provides a handy way to write the output.
	// The magic values suck, but basically we are telling it that w is not a tty.
	// This function would try to do fancy stuff if w is a tty, but it might be wrapped by a spinner
	// or it might be non-existent so just opt out of that.
	if err = jsonmessage.DisplayJSONMessagesStream(r, w, 0, false, nil); err != nil {
		// err here can either be an error with displaying the progress or an error having
		// occurred during image pull.
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: "error while pulling image",
			Op:     op,
		})
	}
	return nil
}

// ImageSearch is used to find an image for image related operations.
type ImageSearch struct {
	// Name is the name of the image. It is expected to be a valid docker image name
	// unless LocalBuild is set in which case it is expected to be a service name which
	// will be normalized.
	Name string
	// LocalBuild specifies that the image was built locally
	// and was not pulled from a remote registry.
	LocalBuild bool
}

// RemoveImages removes all the specified images. RemoveImages will find
// all matching images with the same name regardless of tag and remove them.
// It will also remove all children of each image.
func (d *Docker) RemoveImages(ctx context.Context, imageSearches []ImageSearch) error {
	const op = errors.Op("docker.Docker.RemoveImages")
	// Create a set of filters for each image name to search for
	f := filters.NewArgs()
	const referenceKey = "reference"
	var errs errors.List
	for _, is := range imageSearches {
		if is.LocalBuild {
			// If it's a local build the image name will be the service name
			// so we need to convert it into the name generated by docker compose.
			n := buildImageName(d.project.Name, NormalizeName(is.Name))
			f.Add(referenceKey, n)
			continue
		}

		// Parse the name. This has two functions:
		// 1. Ensure the name is a valid docker name.
		// 2. Allows for extracting the name without the tag since we remove all tags.
		ref, err := reference.ParseNormalizedNamed(is.Name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		// ParseNormalizedNamed normalizes image names that are on dockerhub,
		// ex: postgres -> docker.io/library/postgres
		// For some reason ImageList does not like that so get the short name.
		f.Add(referenceKey, reference.FamiliarName(ref))
	}
	if len(errs) > 0 {
		return errors.Wrap(errs, errors.Meta{
			Kind:   errkind.Invalid,
			Reason: "unable to parse image names",
			Op:     op,
		})
	}

	images, err := d.apiClient.ImageList(ctx, types.ImageListOptions{Filters: f})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "failed to list images",
			Op:     op,
		})
	}

	tracker := progress.TrackerFromContext(ctx)
	for _, image := range images {
		// Each image can have multiple tags associated with it
		imageNames := strings.Join(image.RepoTags, ", ")
		tracker.Debugf("Removing images: %s", imageNames)
		_, err := d.apiClient.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if errdefs.IsNotFound(err) {
			tracker.Warnf("No images found to remove: %s", imageNames)
		}
		if err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.Docker,
				Reason: fmt.Sprintf("failed to remove image %s: %s", image.ID, imageNames),
				Op:     op,
			})
		}
	}
	return nil
}

// PruneImages removes all dangling images.
func (d *Docker) PruneImages(ctx context.Context) error {
	_, err := d.apiClient.ImagesPrune(ctx, filters.NewArgs(filters.Arg("dangling", "true")))
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "failed to prune images",
			Op:     "docker.Docker.PruneImages",
		})
	}
	return nil
}

// RemoveNetworks removes all networks associated with the project.
func (d *Docker) RemoveNetworks(ctx context.Context) error {
	const op = errors.Op("docker.Docker.RemoveNetworks")
	networks, err := d.apiClient.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters.NewArgs(projectFilter(d.project.Name)),
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "failed to list networks",
			Op:     op,
		})
	}

	tracker := progress.TrackerFromContext(ctx)
	for _, network := range networks {
		tracker.Debugf("Removing network %s", network.Name)
		if err := d.apiClient.NetworkRemove(ctx, network.ID); err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.Docker,
				Reason: fmt.Sprintf("failed to remove network %s, %s", network.Name, network.ID),
				Op:     op,
			})
		}
	}
	return nil
}

// RemoveVolumes removes all volumes associated with the project.
func (d *Docker) RemoveVolumes(ctx context.Context) error {
	const op = errors.Op("docker.Docker.RemoveVolumes")
	volumes, err := d.apiClient.VolumeList(ctx, filters.NewArgs(projectFilter(d.project.Name)))
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "failed to list volumes",
			Op:     op,
		})
	}

	tracker := progress.TrackerFromContext(ctx)
	for _, volume := range volumes.Volumes {
		tracker.Debugf("Removing volume %s", volume.Name)
		err := d.apiClient.VolumeRemove(ctx, volume.Name, true)
		if errdefs.IsNotFound(err) {
			tracker.Warnf("No volume found to remove: %s", volume.Name)
		}
		if err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.Docker,
				Reason: fmt.Sprintf("failed to remove volume %s", volume.Name),
				Op:     op,
			})
		}
	}
	return nil
}

// BuildServices builds images for services.
func (d *Docker) BuildServices(ctx context.Context, serviceNames []string) error {
	err := d.apiClient.ComposeBuild(ctx, d.project, normalizeNames(serviceNames))
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind: errkind.DockerCompose,
			Op:   "docker.Docker.BuildServices",
		})
	}
	return nil
}

// UpServices prepares and runs services in containers.
func (d *Docker) UpServices(ctx context.Context, serviceNames []string) error {
	err := d.apiClient.ComposeUp(ctx, d.project, normalizeNames(serviceNames))
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind: errkind.DockerCompose,
			Op:   "docker.Docker.UpServices",
		})
	}
	return nil
}

// RunServices creates a one-off service container and executes a command in it.
func (d *Docker) RunService(ctx context.Context, serviceName, cmd string) error {
	err := d.apiClient.ComposeRun(ctx, d.project, ComposeRunOptions{
		Service: NormalizeName(serviceName),
		Cmd:     strings.Fields(cmd),
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind: errkind.DockerCompose,
			Op:   "docker.Docker.RunService",
		})
	}
	return nil
}

type ExecInServiceOptions struct {
	// Cmd is the command to execute. It must have at
	// least one element which is the name of the command.
	// Any additional elements are args for the command.
	Cmd []string
	// Stdin will be attached to the container's stdin.
	Stdin io.Reader
	// Stdout will be attached to the container's stdout.
	Stdout io.Writer
	// Stderr will be attached to the container's stderr.
	Stderr io.Writer
}

// ExecInService executes a command in a running service container and returns the exit code.
func (d *Docker) ExecInService(ctx context.Context, serviceName string, opts ExecInServiceOptions) (int, error) {
	exitCode, err := d.apiClient.ComposeExec(ctx, d.project, ComposeRunOptions{
		Service: NormalizeName(serviceName),
		Cmd:     opts.Cmd,
		Stdin:   opts.Stdin,
		Stdout:  opts.Stdout,
		Stderr:  opts.Stderr,
	})
	if err != nil {
		return exitCode, errors.Wrap(err, errors.Meta{
			Kind: errkind.DockerCompose,
			Op:   "docker.Docker.ExecInService",
		})
	}
	return exitCode, nil
}

type LogsFromServicesOptions struct {
	// List of service names to retrieve logs from.
	// If omitted, logs will be retrieved from all running services.
	ServiceNames []string
	// Out is where container logs will be written.
	Out io.Writer
	// Follow follows the log output. It shows new logs in real time.
	Follow bool
	// Tail is the number of lines to show from the end of the logs.
	// A value of -1 means show all logs.
	Tail int
}

// LogsFromServices retrieves the logs from service containers.
func (d *Docker) LogsFromServices(ctx context.Context, opts LogsFromServicesOptions) error {
	var tail string
	if opts.Tail >= 0 {
		tail = strconv.Itoa(opts.Tail)
	}
	err := d.apiClient.ComposeLogs(ctx, d.project, ComposeLogsOptions{
		Services: normalizeNames(opts.ServiceNames),
		Out:      opts.Out,
		Follow:   opts.Follow,
		Tail:     tail,
	})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind: errkind.DockerCompose,
			Op:   "docker.Docker.LogsFromServices",
		})
	}
	return nil
}

// buildImageName returns the name of an image that is built locally by compose.
// It assumes serviceName has already been normalized.
func buildImageName(projectName string, serviceName string) string {
	return projectName + "_" + serviceName
}

// projectFilter returns a docker filter for the projectName.
func projectFilter(projectName string) filters.KeyValuePair {
	return filters.Arg("label", ProjectLabel+"="+projectName)
}

// normalizeNames calls normalizeName on each name.
func normalizeNames(names []string) []string {
	nn := make([]string, len(names))
	for i, n := range names {
		nn[i] = NormalizeName(n)
	}
	return nn
}
