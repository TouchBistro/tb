package resource_test

import (
	"errors"
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
