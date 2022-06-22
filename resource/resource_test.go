package resource_test

import (
	"errors"
	"sort"
	"testing"

	"github.com/TouchBistro/tb/resource"
	"github.com/matryer/is"
)

func TestParseName(t *testing.T) {
	tests := []struct {
		name             string
		inputName        string
		wantRegistryName string
		wantResourceName string
		wantErr          error
	}{
		{
			name:             "full name",
			inputName:        "TouchBistro/tb-registry/touchbistro-node-boilerplate",
			wantRegistryName: "TouchBistro/tb-registry",
			wantResourceName: "touchbistro-node-boilerplate",
		},
		{
			name:             "short name",
			inputName:        "touchbistro-node-boilerplate",
			wantRegistryName: "",
			wantResourceName: "touchbistro-node-boilerplate",
		},
		{
			name:      "invalid name",
			inputName: "TouchBistro/touchbistro-node-boilerplate",
			wantErr:   resource.ErrInvalidName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			registryName, resourceName, err := resource.ParseName(tt.inputName)
			is.Equal(registryName, tt.wantRegistryName)
			is.Equal(resourceName, tt.wantResourceName)
			is.True(errors.Is(err, tt.wantErr))
		})
	}
}

func TestFullName(t *testing.T) {
	tests := []struct {
		name         string
		registryName string
		resourceName string
		wantName     string
	}{
		{
			name:         "registry name and resource name",
			registryName: "TouchBistro/tb-registry",
			resourceName: "touchbistro-node-boilerplate",
			wantName:     "TouchBistro/tb-registry/touchbistro-node-boilerplate",
		},
		{
			name:         "short name",
			registryName: "",
			resourceName: "touchbistro-node-boilerplate",
			wantName:     "touchbistro-node-boilerplate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			name := resource.FullName(tt.registryName, tt.resourceName)
			is.Equal(name, tt.wantName)
		})
	}
}

func TestCollectionGet(t *testing.T) {
	c := newCollection(t)
	tests := []struct {
		name        string
		lookupName  string
		wantService mockService
		wantErr     error
	}{
		{
			name:       "full name",
			lookupName: "TouchBistro/tb-registry/postgres",
			wantService: mockService{
				Name:         "postgres",
				RegistryName: "TouchBistro/tb-registry",
				Tag:          "12-alpine",
			},
		},
		{
			name:       "short name",
			lookupName: "venue-core-service",
			wantService: mockService{
				Name:         "venue-core-service",
				RegistryName: "TouchBistro/tb-registry",
				Tag:          "main",
			},
		},
		// Error cases
		{
			name:       "short name, multiple services",
			lookupName: "postgres",
			wantErr:    resource.ErrMultipleResources,
		},
		{
			name:       "not found",
			lookupName: "TouchBistro/tb-registry/not-a-service",
			wantErr:    resource.ErrNotFound,
		},
		{
			name:       "no registry",
			lookupName: "ExampleZone/tb-registry/venue-core-service",
			wantErr:    resource.ErrNotFound,
		},
		{
			name:       "invalid name",
			lookupName: "Invalid/bad-name",
			wantErr:    resource.ErrInvalidName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			s, err := c.Get(tt.lookupName)
			is.Equal(s, tt.wantService)
			is.True(errors.Is(err, tt.wantErr))
		})
	}
}

func TestCollectionGetEmpty(t *testing.T) {
	tests := []struct {
		name       string
		collection *resource.Collection[mockService]
		wantErr    error
	}{
		{
			name:       "zero value collection",
			collection: &resource.Collection[mockService]{},
			wantErr:    resource.ErrNotFound,
		},
		{
			name:       "nil collection",
			collection: nil,
			wantErr:    resource.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			_, err := tt.collection.Get("postgres")
			is.True(errors.Is(err, tt.wantErr))
		})
	}
}

func TestCollectionLen(t *testing.T) {
	tests := []struct {
		name       string
		collection *resource.Collection[mockService]
		wantLen    int
	}{
		{
			name:       "collection with elements",
			collection: newCollection(t),
			wantLen:    3,
		},
		{
			name:       "zero value collection",
			collection: &resource.Collection[mockService]{},
			wantLen:    0,
		},
		{
			name:       "collection with elements",
			collection: nil,
			wantLen:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			is.Equal(tt.collection.Len(), tt.wantLen)
		})
	}
}

func TestCollectionIter(t *testing.T) {
	tests := []struct {
		name       string
		collection *resource.Collection[mockService]
		wantNames  []string
	}{
		{
			name:       "collection with elements",
			collection: newCollection(t),
			wantNames: []string{
				"ExampleZone/tb-registry/postgres",
				"TouchBistro/tb-registry/postgres",
				"TouchBistro/tb-registry/venue-core-service",
			},
		},
		{
			name:       "zero value collection",
			collection: &resource.Collection[mockService]{},
			wantNames:  []string{},
		},
		{
			name:       "nil collection",
			collection: nil,
			wantNames:  []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			names := make([]string, 0, tt.collection.Len())
			for it := tt.collection.Iter(); it.Next(); {
				names = append(names, it.Value().FullName())
			}
			sort.Strings(names)
			is.Equal(names, tt.wantNames)
		})
	}
}

// mockService is a simple type that implements the Resource interface
// so we can test resource.Collection without needing the service package.
type mockService struct {
	Name         string
	RegistryName string
	Tag          string
}

func (mockService) Type() resource.Type {
	return resource.TypeService
}

func (s mockService) FullName() string {
	return resource.FullName(s.RegistryName, s.Name)
}

func newCollection(t *testing.T) *resource.Collection[mockService] {
	// Creates two services with the same name but different registries
	// and one service that's a unique name
	svcs := []mockService{
		{
			Name:         "postgres",
			RegistryName: "ExampleZone/tb-registry",
			Tag:          "12",
		},
		{
			Name:         "postgres",
			RegistryName: "TouchBistro/tb-registry",
			Tag:          "12-alpine",
		},
		{
			Name:         "venue-core-service",
			RegistryName: "TouchBistro/tb-registry",
			Tag:          "main",
		},
	}
	var c resource.Collection[mockService]
	for _, s := range svcs {
		if err := c.Set(s); err != nil {
			t.Fatalf("failed to add service %s to collection: %v", s.FullName(), err)
		}
	}
	return &c
}
