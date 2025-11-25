package docker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/distribution/reference"
	configtypes "github.com/docker/cli/cli/config/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/registry"
)

// notFoundError implements the docker errdefs.ErrNotFound interface.
type notFoundError string

func (notFoundError) NotFound() {}

func (e notFoundError) Error() string {
	return string(e)
}

type mockConfig struct {
	// map of registry name to auth config
	authConfigs map[string]configtypes.AuthConfig
}

// NewMockConfig returns a mock Config that is suitable for tests.
func NewMockConfig(authConfigs []configtypes.AuthConfig) Config {
	m := &mockConfig{authConfigs: make(map[string]configtypes.AuthConfig)}
	for _, ac := range authConfigs {
		if ac.ServerAddress == "" {
			panic("auth config missing server address")
		}
		m.authConfigs[ac.ServerAddress] = ac
	}
	return m
}

func (m *mockConfig) GetAuthConfig(registryHostname string) (configtypes.AuthConfig, error) {
	// Docker does not actually return an error if the auth config isn't found but instead lets
	// the user know if it receives an auth error from the registry. This allows pulling public
	// images without being logged in.
	return m.authConfigs[registryHostname], nil
}

type mockAPIClient struct {
	// We don't need all functionality provided by the docker SDK so embed an APIClient
	// so satisfy the interface even though we have unimplemented methods.
	// If a test tries to use an unimplemented method it will panic.
	APIClient
	indexServerAddress string

	// State for mock functionality
	// Each map's keys are the IDs of the given resource for easy lookup.

	containers map[string]containertypes.Summary
	images     map[string]imagetypes.Summary
	networks   map[string]networktypes.Inspect
	volumes    map[string]volumetypes.Volume

	// map of server address to registry
	registries map[string]MockRegistry
}

type MockRegistry struct {
	ServerAddress string
	AuthConfig    configtypes.AuthConfig
	Repositories  map[string]MockRegistryRepository
}

type MockRegistryRepository struct {
	Images []imagetypes.Summary
	Public bool
}

type MockAPIClientOptions struct {
	// Containers is the initial containers the mock client should have.
	Containers []containertypes.Summary
	// Images is the initial images the mock client should have.
	Images []imagetypes.Summary
	// Networks is the initial networks the mock client should have.
	Networks []networktypes.Inspect
	// Volumes is the initial volumes the mock client should have.
	Volumes []volumetypes.Volume
	// Registries is a list of mock registries to pull images from.
	Registries []MockRegistry
}

// NewMock returns a mock APIClient that is suitable for tests.
func NewMockAPIClient(opts MockAPIClientOptions) APIClient {
	m := &mockAPIClient{
		indexServerAddress: registry.IndexServer,
		containers:         make(map[string]containertypes.Summary),
		images:             make(map[string]imagetypes.Summary),
		networks:           make(map[string]networktypes.Inspect),
		volumes:            make(map[string]volumetypes.Volume),
		registries:         make(map[string]MockRegistry),
	}
	for _, c := range opts.Containers {
		if c.ID == "" {
			panic("container is missing id")
		}
		m.containers[c.ID] = c
	}
	for _, im := range opts.Images {
		if im.ID == "" {
			panic("image is missing id")
		}
		m.images[im.ID] = im
	}
	for _, n := range opts.Networks {
		if n.ID == "" {
			panic("network is missing id")
		}
		m.networks[n.ID] = n
	}
	for _, v := range opts.Volumes {
		if v.Name == "" {
			panic("volume is missing name")
		}
		m.volumes[v.Name] = v
	}
	for _, r := range opts.Registries {
		if r.ServerAddress == "" {
			panic("registry is missing server address")
		}
		m.registries[r.ServerAddress] = r
	}
	return m
}

func (m *mockAPIClient) Info(ctx context.Context) (system.Info, error) {
	return system.Info{IndexServerAddress: m.indexServerAddress}, nil
}

func (m *mockAPIClient) ContainerList(ctx context.Context, options containertypes.ListOptions) ([]containertypes.Summary, error) {
	// Get all filters we will need to check
	labelFilters := options.Filters.Get("label")
	// Turn string slice into a map for easy lookup
	nameFilters := make(map[string]bool)
	for _, n := range options.Filters.Get("name") {
		nameFilters[n] = true
	}

	var found []containertypes.Summary
	for _, c := range m.containers {
		if !options.All && c.State != ContainerStateRunning {
			continue
		}
		// Handle filters
		if !checkLabelFilters(c.Labels, labelFilters) {
			continue
		}
		if len(nameFilters) > 0 {
			match := false
			for _, n := range c.Names {
				if ok := nameFilters[n]; ok {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		found = append(found, c)
	}
	return found, nil
}

func (m *mockAPIClient) ContainerRemove(ctx context.Context, container string, options containertypes.RemoveOptions) error {
	found, err := m.findContainerByID(container)
	if err != nil {
		return err
	}
	if found.State == ContainerStateRunning {
		return fmt.Errorf("cannot remove a running container: %s", container)
	}

	delete(m.containers, container)
	return nil
}

func (m *mockAPIClient) ContainerStop(ctx context.Context, container string, options containertypes.StopOptions) error {
	if container == "" {
		return fmt.Errorf("container cannot be empty")
	}
	found, err := m.findContainerByID(container)
	if err != nil {
		return err
	}
	if found.State == ContainerStateExited {
		// The docker API returns a 304 if the container is already stopped, so this isn't an error
		return nil
	}

	found.State = ContainerStateExited
	m.containers[container] = found
	return nil
}

func (m *mockAPIClient) findContainerByID(id string) (containertypes.Summary, error) {
	if id == "" {
		return containertypes.Summary{}, fmt.Errorf("container cannot be empty")
	}
	c, ok := m.containers[id]
	if !ok {
		return containertypes.Summary{}, notFoundError(fmt.Sprintf("no such container: %s", id))
	}
	return c, nil
}

func (m *mockAPIClient) ImagePull(ctx context.Context, ref string, options imagetypes.PullOptions) (io.ReadCloser, error) {
	// Resolve registry from image name
	parsedRef, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return nil, err
	}
	serverAddress := reference.Domain(parsedRef)

	// Decode auth
	var authConfig configtypes.AuthConfig
	if options.RegistryAuth != "" {
		data, err := base64.URLEncoding.DecodeString(options.RegistryAuth)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &authConfig); err != nil {
			return nil, err
		}

		// Make sure the server matches
		authServerAddress := serverAddress
		if serverAddress == registry.IndexName {
			authServerAddress = m.indexServerAddress
		}
		if authConfig.ServerAddress != authServerAddress {
			return nil, fmt.Errorf("auth is not for the correct registry")
		}
	}

	// Find registry
	r, ok := m.registries[serverAddress]
	if !ok {
		return nil, fmt.Errorf("registry does not exist: %s", serverAddress)
	}
	imageName := reference.FamiliarName(parsedRef)
	repo, ok := r.Repositories[imageName]
	if !ok {
		return nil, fmt.Errorf("no such repository: %s", imageName)
	}

	// Fake auth check
	if !repo.Public && options.RegistryAuth == "" {
		return nil, fmt.Errorf("authentication required")
	} else if authConfig.Username != r.AuthConfig.Username || authConfig.Password != r.AuthConfig.Password {
		return nil, fmt.Errorf("authentication error")
	}
	// Add latest tag if no tag
	imageName = reference.TagNameOnly(parsedRef).String()

	var image imagetypes.Summary
	found := false
Loop:
	for _, im := range repo.Images {
		for _, rt := range im.RepoTags {
			if rt == imageName {
				image = im
				found = true
				break Loop
			}
		}
	}
	if !found {
		return nil, notFoundError(fmt.Sprintf("no such image: %s", imageName))
	}

	// Add the image to "local images" so it's pulled
	m.images[image.ID] = image
	// TODO(@cszatmary): Figure out how to send a proper message.
	// For now just try sending an empty buffer which will return EOF on read and mark the end.
	return io.NopCloser(&bytes.Reader{}), nil
}

func (m *mockAPIClient) ImageList(ctx context.Context, options imagetypes.ListOptions) ([]imagetypes.Summary, error) {
	// All only applies if no filters are provided
	if options.All && options.Filters.Len() == 0 {
		var images []imagetypes.Summary
		for _, im := range m.images {
			images = append(images, im)
		}
		return images, nil
	}

	// Turn string slice into a map for easy lookup
	referenceFilters := make(map[string]bool)
	for _, n := range options.Filters.Get("reference") {
		referenceFilters[n] = true
	}

	var found []imagetypes.Summary
	for _, im := range m.images {
		// Handle filters
		if len(referenceFilters) > 0 {
			match := false
			for _, rt := range im.RepoTags {
				// First check if the full tag was provided
				if ok := referenceFilters[rt]; ok {
					match = true
					// If full match filter out tags that don't match
					im.RepoTags = []string{rt}
					break
				}
				// Also check if just the name without the tag matches
				ref, err := reference.ParseNormalizedNamed(rt)
				if err != nil {
					panic(err)
				}
				if ok := referenceFilters[reference.FamiliarName(ref)]; ok {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		found = append(found, im)
	}
	return found, nil
}

func (m *mockAPIClient) ImageRemove(ctx context.Context, image string, options imagetypes.RemoveOptions) ([]imagetypes.DeleteResponse, error) {
	if image == "" {
		return nil, fmt.Errorf("image cannot be empty")
	}
	im, ok := m.images[image]
	if !ok {
		return nil, notFoundError(fmt.Sprintf("no such image: %s", image))
	}
	delete(m.images, image)
	resp := []imagetypes.DeleteResponse{{Deleted: im.ID}}
	for _, rt := range im.RepoTags {
		resp = append(resp, imagetypes.DeleteResponse{Untagged: rt})
	}
	return resp, nil
}

func (m *mockAPIClient) ImagesPrune(ctx context.Context, pruneFilter filters.Args) (imagetypes.PruneReport, error) {
	danglingFilters := pruneFilter.Get("dangling")
	removeDangling := false
	for _, df := range danglingFilters {
		if b, err := strconv.ParseBool(df); err == nil {
			removeDangling = b
		}
	}

	var report imagetypes.PruneReport
	for id, im := range m.images {
		if im.Containers > 0 {
			// Image is used, skip
			continue
		}
		if len(im.RepoTags) > 0 || !removeDangling {
			continue
		}
		delete(m.images, id)
		report.ImagesDeleted = append(report.ImagesDeleted, imagetypes.DeleteResponse{Deleted: im.ID})
		report.SpaceReclaimed += uint64(im.Size)
	}
	return report, nil
}

func (m *mockAPIClient) NetworkList(ctx context.Context, options networktypes.ListOptions) ([]networktypes.Inspect, error) {
	// Get all filters we will need to check
	labelFilters := options.Filters.Get("label")

	var found []networktypes.Inspect
	for _, n := range m.networks {
		// Handle filters
		if !checkLabelFilters(n.Labels, labelFilters) {
			continue
		}
		found = append(found, n)
	}
	return found, nil
}

func (m *mockAPIClient) NetworkRemove(ctx context.Context, network string) error {
	if network == "" {
		return fmt.Errorf("network cannot be empty")
	}
	_, ok := m.networks[network]
	if !ok {
		return notFoundError(fmt.Sprintf("no such network: %s", network))
	}
	delete(m.networks, network)
	return nil
}

func (m *mockAPIClient) VolumeList(ctx context.Context, options volumetypes.ListOptions) (volumetypes.ListResponse, error) {
	// Get all filters we will need to check
	labelFilters := options.Filters.Get("label")

	var found []*volumetypes.Volume
	for _, v := range m.volumes {
		// Handle filters
		if !checkLabelFilters(v.Labels, labelFilters) {
			continue
		}
		// The return type requires a slice of volume pointers for some reason which is annoying.
		// Need to make a copy so we can take a pointer
		vv := v
		found = append(found, &vv)
	}
	return volumetypes.ListResponse{Volumes: found}, nil
}

func (m *mockAPIClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	if volumeID == "" {
		return fmt.Errorf("volumeID cannot be empty")
	}
	_, ok := m.volumes[volumeID]
	if !ok {
		return notFoundError(fmt.Sprintf("no such volume: %s", volumeID))
	}
	delete(m.volumes, volumeID)
	return nil
}

func checkLabelFilters(labels map[string]string, labelFilters []string) bool {
	for _, f := range labelFilters {
		parts := strings.Split(f, "=")
		v, ok := labels[parts[0]]
		if !ok {
			// Label does not exist
			return false
		}
		if len(parts) == 1 {
			// If only one part it means no value was provided in the filter
			// Therefore we have a match since the label key exists
			continue
		}
		if v != parts[1] {
			return false
		}
	}
	return true
}
