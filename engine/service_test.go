package engine_test

import (
	"context"
	"testing"

	"github.com/TouchBistro/tb/engine"
	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/integrations/git"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/matryer/is"
)

func TestDown(t *testing.T) {
	tests := []struct {
		name                string
		existingContainers  []dockertypes.Container
		serviceNames        []string
		remainingContainers []dockertypes.Container
	}{
		{
			name: "remove all containers since none specified",
			existingContainers: []dockertypes.Container{
				{
					ID:    "touchbistro-tb-registry-postgres",
					Names: []string{"touchbistro-tb-registry-postgres"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				{
					ID:    "touchbistro-tb-registry-touchbistro-node-boilerplate",
					Names: []string{"touchbistro-tb-registry-touchbistro-node-boilerplate"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				// Additional container not part of tb to make sure it is untouched.
				{
					ID:    "test-ubuntu",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
			remainingContainers: []dockertypes.Container{
				{
					ID:    "test-ubuntu",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
		},
		{
			name: "remove only specified containers",
			existingContainers: []dockertypes.Container{
				{
					ID:    "touchbistro-tb-registry-postgres",
					Names: []string{"touchbistro-tb-registry-postgres"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				{
					ID:    "touchbistro-tb-registry-touchbistro-node-boilerplate",
					Names: []string{"touchbistro-tb-registry-touchbistro-node-boilerplate"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				// Additional container not part of tb to make sure it is untouched.
				{
					ID:    "test-ubuntu",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
			serviceNames: []string{"TouchBistro/tb-registry/touchbistro-node-boilerplate"},
			remainingContainers: []dockertypes.Container{
				{
					ID:    "touchbistro-tb-registry-postgres",
					Names: []string{"touchbistro-tb-registry-postgres"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				{
					ID:    "test-ubuntu",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc, pc := newServiceFixtures(t)
			dockerAPIClient := docker.NewMockAPIClient(tt.existingContainers)
			e := newEngine(t, engine.Options{
				Services:        sc,
				Playlists:       pc,
				DockerAPIClient: dockerAPIClient,
			})

			ctx := context.Background()
			err := e.Down(ctx, engine.DownOptions{
				ServiceNames: tt.serviceNames,
			})
			is := is.New(t)
			is.NoErr(err)

			// Check to make sure those containers were removed
			remaining, err := dockerAPIClient.ContainerList(ctx, dockertypes.ContainerListOptions{All: true})
			is.NoErr(err)
			is.Equal(remaining, tt.remainingContainers)
		})
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name string
		opts engine.ListOptions
		want engine.ListResult
	}{
		{
			name: "list all",
			opts: engine.ListOptions{
				ListServices:        true,
				ListPlaylists:       true,
				ListCustomPlaylists: true,
			},
			want: engine.ListResult{
				Services: []string{
					"ExampleZone/tb-registry/postgres",
					"TouchBistro/tb-registry/postgres",
					"TouchBistro/tb-registry/touchbistro-node-boilerplate",
				},
				Playlists: []engine.PlaylistSummary{
					{Name: "TouchBistro/tb-registry/backend"},
				},
				CustomPlaylists: []engine.PlaylistSummary{
					{Name: "my-backend"},
				},
			},
		},
		{
			name: "tree mode",
			opts: engine.ListOptions{
				ListPlaylists:       true,
				ListCustomPlaylists: true,
				TreeMode:            true,
			},
			want: engine.ListResult{
				Playlists: []engine.PlaylistSummary{
					{
						Name: "TouchBistro/tb-registry/backend",
						Services: []string{
							"TouchBistro/tb-registry/postgres",
							"TouchBistro/tb-registry/touchbistro-node-boilerplate",
						},
					},
				},
				CustomPlaylists: []engine.PlaylistSummary{
					{
						Name: "my-backend",
						Services: []string{
							"ExampleZone/tb-registry/postgres",
							"TouchBistro/tb-registry/touchbistro-node-boilerplate",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc, pc := newServiceFixtures(t)
			e := newEngine(t, engine.Options{
				Services:  sc,
				Playlists: pc,
			})

			result := e.List(tt.opts)
			is := is.New(t)
			is.Equal(result, tt.want)
		})
	}
}

func newServiceFixtures(t *testing.T) (*service.Collection, *playlist.Collection) {
	t.Helper()

	// Creates two services with the same name but different registries
	// and one service that's a unique name
	svcs := []service.Service{
		{
			EnvVars: map[string]string{
				"POSTGRES_USER":     "user",
				"POSTGRES_PASSWORD": "password",
			},
			Mode: service.ModeRemote,
			Remote: service.Remote{
				Image: "postgres",
				Tag:   "12",
			},
			Name:         "postgres",
			RegistryName: "ExampleZone/tb-registry",
		},
		{
			EnvVars: map[string]string{
				"POSTGRES_USER":     "core",
				"POSTGRES_PASSWORD": "localdev",
			},
			Mode: service.ModeRemote,
			Remote: service.Remote{
				Image: "postgres",
				Tag:   "12-alpine",
			},
			Name:         "postgres",
			RegistryName: "TouchBistro/tb-registry",
		},
		{
			EnvFile: ".tb/repos/TouchBistro/touchbistro-node-boilerplate/.env.example",
			EnvVars: map[string]string{
				"HTTP_PORT": "8080",
			},
			Mode: service.ModeBuild,
			Ports: []string{
				"8081:8080",
			},
			PreRun: "yarn db:prepare:dev",
			GitRepo: service.GitRepo{
				Name: "TouchBistro/touchbistro-node-boilerplate",
			},
			Build: service.Build{
				Args: map[string]string{
					"NODE_ENV": "development",
				},
				Command:        "yarn start",
				DockerfilePath: ".tb/repos/TouchBistro/touchbistro-node-boilerplate",
				Target:         "release",
			},
			Name:         "touchbistro-node-boilerplate",
			RegistryName: "TouchBistro/tb-registry",
		},
	}
	var sc service.Collection
	for _, s := range svcs {
		if err := sc.Set(s); err != nil {
			t.Fatalf("failed to add service %s to collection: %v", s.FullName(), err)
		}
	}

	// Create a playlist collection with a playlist and a custom playlist
	playlists := []playlist.Playlist{
		{
			Services: []string{
				"TouchBistro/tb-registry/postgres",
				"TouchBistro/tb-registry/touchbistro-node-boilerplate",
			},
			Name:         "backend",
			RegistryName: "TouchBistro/tb-registry",
		},
	}
	customPlaylists := []playlist.Playlist{
		{
			Services: []string{
				"ExampleZone/tb-registry/postgres",
				"TouchBistro/tb-registry/touchbistro-node-boilerplate",
			},
			Name: "my-backend",
		},
	}
	var pc playlist.Collection
	for _, p := range playlists {
		if err := pc.Set(p); err != nil {
			t.Fatalf("failed to add playlist %s to collection: %v", p.FullName(), err)
		}
	}
	for _, p := range customPlaylists {
		pc.SetCustom(p)
	}

	return &sc, &pc
}

func newEngine(t *testing.T, opts engine.Options) *engine.Engine {
	t.Helper()

	// Set defaults to ensure we don't forget to mock out stuff and accidentally
	// try to use real clients as if clients are omitted the real versions will
	// be initialized by New.
	if opts.Workdir == "" {
		opts.Workdir = t.TempDir()
	}
	if opts.GitClient == nil {
		opts.GitClient = git.NewMock()
	}
	if opts.DockerAPIClient == nil {
		opts.DockerAPIClient = docker.NewMockAPIClient(nil)
	}
	if opts.ComposeClient == nil {
		opts.ComposeClient = docker.NewMockCompose()
	}

	e, err := engine.New(opts)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	return e
}
