package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
)

type mockAPIClient struct {
	// We don't need all functionality provided by the docker SDK so embed an APIClient
	// so satisfy the interface even though we have unimplemented methods.
	// If a test tries to use an unimplemented method it will panic.
	APIClient

	// State for mock functionality

	// Existing containers, map of container ID to container.
	containers map[string]types.Container
}

// NewMock returns a mock APIClient that is suitable for tests.
func NewMockAPIClient(containers []types.Container) APIClient {
	m := &mockAPIClient{
		containers: make(map[string]types.Container),
	}
	for _, c := range containers {
		m.containers[c.ID] = c
	}
	return m
}

func (m *mockAPIClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	// Get all filters we will need to check
	labelFilters := options.Filters.Get("label")
	// Turn string slice into a map for easy lookup
	nameFilters := make(map[string]bool)
	for _, n := range options.Filters.Get("name") {
		nameFilters[n] = true
	}

	var found []types.Container
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

func (m *mockAPIClient) ContainerRemove(ctx context.Context, container string, options types.ContainerRemoveOptions) error {
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

func (m *mockAPIClient) ContainerStop(ctx context.Context, container string, timeout *time.Duration) error {
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

func (m *mockAPIClient) findContainerByID(id string) (types.Container, error) {
	if id == "" {
		return types.Container{}, fmt.Errorf("container cannot be empty")
	}
	c, ok := m.containers[id]
	if !ok {
		return types.Container{}, fmt.Errorf("no such container: %s", id)
	}
	return c, nil
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
