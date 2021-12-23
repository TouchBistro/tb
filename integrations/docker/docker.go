package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/tb/errkind"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
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

// ParseImageName parses a docker image name into its repo and tag components.
// For example: fedora/httpd:version1.0 would return (fedora/httpd, version1.0)
func ParseImageName(name string) (repo, tag string) {
	// Need to find the last index because image names can have a colon at
	// the start if there is a domain and port.
	// Ex: myregistryhost:5000/fedora/httpd:version1.0
	i := strings.LastIndexByte(name, ':')
	if i == -1 {
		// No tag present
		return name, ""
	}
	return name[:i], name[i+1:]
}

// NOTE: The docs for the Go SDK aren't great. If you are working on this and want to
// better understand how the docker APIs work check out the actual engine API docs
// which have the OpenAPI schema.
// https://docs.docker.com/engine/api/latest

// APIClient is the functionality required to be implemented by
// a client that communicates with the docker API.
type APIClient interface {
	client.ContainerAPIClient
	client.ImageAPIClient
	client.NetworkAPIClient
	client.VolumeAPIClient
}

// NewAPIClient returns a new APIClient for communicating with the docker API.
func NewAPIClient() (APIClient, error) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "unable to create docker client",
			Op:     "docker.New",
		})
	}
	return apiClient, nil
}

// Docker provides functionality for working with docker resources.
type Docker struct {
	apiClient   APIClient
	projectName string
}

// New returns a new Docker instance that uses the given apiClient to communicate
// with the docker API. projectName is a compose project name.
// Only docker resources belonging to this project will be modified.
func New(apiClient APIClient, projectName string) *Docker {
	return &Docker{apiClient, projectName}
}

func (d *Docker) PullImage(ctx context.Context, imageURI string) error {
	tracker := progress.TrackerFromContext(ctx)
	w := progress.LogWriter(tracker, tracker.WithFields(progress.Fields{"id": "docker-pull"}).Debug)
	defer w.Close()
	cmd := exec.CommandContext(ctx, "docker", "pull", imageURI)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: fmt.Sprintf("failed to pull docker image %s", imageURI),
			Op:     "docker.Docker.ImagePull",
		})
	}

	// TODO(@cszatmary): The docker SDK supports pulling images but it requires explicit auth.
	// I'd like to switch to using that, but it will require some changes to how we handle logins.
	// rc, err := d.client.ImagePull(ctx, imageURI, types.ImagePullOptions{})
	// if err != nil {
	// 	return errors.New(errors.Docker, fmt.Sprintf("failed to pull docker image %s", imageURI), op, err)
	// }
	// defer rc.Close()
	return nil
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

// ImageSearch is used to find an image for image related operations.
type ImageSearch struct {
	// Name is the name of the image.
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
	images, err := d.apiClient.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "failed to list images",
			Op:     op,
		})
	}

	// Unfortunately the docker api has no way to filter images by name when retrieving
	// so we have to do it ourselves after
	nameSet := make(map[string]struct{})
	for _, is := range imageSearches {
		n, _ := ParseImageName(is.Name)
		if is.LocalBuild {
			// If local build we need to get the compose name.
			n = buildImageName(d.projectName, NormalizeName(n))
		}
		nameSet[n] = struct{}{}
	}

	type imageDetail struct {
		id  string
		tag string
	}
	var filteredImages []imageDetail
	for _, image := range images {
		for _, tag := range image.RepoTags {
			// Tag will be the full tag like foo/bar:latest
			// Just want to match the image name
			name := strings.Split(tag, ":")[0]
			if _, ok := nameSet[name]; ok {
				filteredImages = append(filteredImages, imageDetail{image.ID, tag})
			}
		}
	}

	tracker := progress.TrackerFromContext(ctx)
	for _, image := range filteredImages {
		tracker.Debugf("Removing image %s", image.tag)
		_, err := d.apiClient.ImageRemove(ctx, image.id, types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if errdefs.IsNotFound(err) {
			tracker.Warnf("No image found to remove: %s", image.tag)
		}
		if err != nil {
			return errors.Wrap(err, errors.Meta{
				Kind:   errkind.Docker,
				Reason: fmt.Sprintf("failed to remove image %s, %s", image.tag, image.id),
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
		Filters: filters.NewArgs(projectFilter(d.projectName)),
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
	volumes, err := d.apiClient.VolumeList(ctx, filters.NewArgs(projectFilter(d.projectName)))
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

func (d *Docker) listContainers(ctx context.Context, serviceNames []string, stopped bool, op errors.Op) ([]types.Container, error) {
	f := filters.NewArgs(projectFilter(d.projectName))
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

// buildImageName returns the name of an image that is built locally by compose.
// It assumes serviceName has already been normalized.
func buildImageName(projectName string, serviceName string) string {
	return projectName + "_" + serviceName
}

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

// projectFilter returns a docker filter for the projectName.
func projectFilter(projectName string) filters.KeyValuePair {
	return filters.Arg("label", ProjectLabel+"="+projectName)
}
