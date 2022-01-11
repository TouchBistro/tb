package docker_test

import (
	"testing"

	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/matryer/is"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "replaces slashes and capital letters",
			in:   "TouchBistro/tb-registry/touchbistro-node-boilerplate",
			want: "touchbistro-tb-registry-touchbistro-node-boilerplate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			got := docker.NormalizeName(tt.in)
			is.Equal(got, tt.want)
		})
	}
}
