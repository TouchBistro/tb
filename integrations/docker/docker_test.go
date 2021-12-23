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

func TestParseImageName(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantRepo string
		wantTag  string
	}{
		{
			name:     "image name with tag",
			in:       "fedora/httpd:version1.0",
			wantRepo: "fedora/httpd",
			wantTag:  "version1.0",
		},
		{
			name:     "image name without tag",
			in:       "fedora/httpd",
			wantRepo: "fedora/httpd",
			wantTag:  "",
		},
		{
			name:     "image name with host",
			in:       "myregistryhost:5000/fedora/httpd:version1.0",
			wantRepo: "myregistryhost:5000/fedora/httpd",
			wantTag:  "version1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			repo, tag := docker.ParseImageName(tt.in)
			is.Equal(repo, tt.wantRepo)
			is.Equal(tag, tt.wantTag)
		})
	}
}
