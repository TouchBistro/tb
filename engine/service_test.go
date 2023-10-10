package engine_test

import (
	"context"
	"sort"
	"testing"

	"github.com/TouchBistro/tb/engine"
	"github.com/TouchBistro/tb/integrations/docker"
	"github.com/TouchBistro/tb/integrations/git"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
	dockertypes "github.com/docker/docker/api/types"
	volumetypes "github.com/docker/docker/api/types/volume"
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
					ID:    "32ce4d8d9c648dd5fce39cf48319da8d55b195513b6fe0cef4a425de9380590c",
					Names: []string{"touchbistro-tb-registry-postgres"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				{
					ID:    "f4d2913f1010244b61940cf52845e6dbe5d687791ea185237efe9121adf15edd",
					Names: []string{"touchbistro-tb-registry-touchbistro-node-boilerplate"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				// Additional container not part of tb to make sure it is untouched.
				{
					ID:    "e8dc7c16f7dd4be23b96951a34b7ecc69cd727ed13a626a309a96b472646c5e9",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
			remainingContainers: []dockertypes.Container{
				{
					ID:    "e8dc7c16f7dd4be23b96951a34b7ecc69cd727ed13a626a309a96b472646c5e9",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
		},
		{
			name: "remove only specified containers",
			existingContainers: []dockertypes.Container{
				{
					ID:    "32ce4d8d9c648dd5fce39cf48319da8d55b195513b6fe0cef4a425de9380590c",
					Names: []string{"touchbistro-tb-registry-postgres"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				{
					ID:    "f4d2913f1010244b61940cf52845e6dbe5d687791ea185237efe9121adf15edd",
					Names: []string{"touchbistro-tb-registry-touchbistro-node-boilerplate"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				// Additional container not part of tb to make sure it is untouched.
				{
					ID:    "e8dc7c16f7dd4be23b96951a34b7ecc69cd727ed13a626a309a96b472646c5e9",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
			serviceNames: []string{"TouchBistro/tb-registry/touchbistro-node-boilerplate"},
			remainingContainers: []dockertypes.Container{
				{
					ID:    "32ce4d8d9c648dd5fce39cf48319da8d55b195513b6fe0cef4a425de9380590c",
					Names: []string{"touchbistro-tb-registry-postgres"},
					Labels: map[string]string{
						docker.ProjectLabel: "tb",
					},
					State: docker.ContainerStateRunning,
				},
				{
					ID:    "e8dc7c16f7dd4be23b96951a34b7ecc69cd727ed13a626a309a96b472646c5e9",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := newServiceCollection(t, nil)
			dockerAPIClient := docker.NewMockAPIClient(docker.MockAPIClientOptions{
				Containers: tt.existingContainers,
			})
			e := newEngine(t, engine.Options{
				Services: sc,
				DockerOptions: docker.Options{
					APIClient: dockerAPIClient,
				},
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
			sort.Slice(remaining, func(i, j int) bool {
				return remaining[i].ID < remaining[j].ID
			})
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
			sc := newServiceCollection(t, nil)
			pc := newPlaylistCollection(t, nil, nil)
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

func TestNuke(t *testing.T) {
	tests := []struct {
		name              string
		services          []service.Service
		mockAPIClientOpts docker.MockAPIClientOptions
		nukeOpts          engine.NukeOptions
		wantContainers    []dockertypes.Container
		wantImages        []dockertypes.ImageSummary
		wantNetworks      []dockertypes.NetworkResource
		wantVolumes       []volumetypes.Volume
	}{
		{
			name: "only remove tb resources",
			services: []service.Service{
				{
					Mode: service.ModeRemote,
					Remote: service.Remote{
						Image: "postgres",
						Tag:   "12",
					},
					Name:         "postgres",
					RegistryName: "TouchBistro/tb-registry",
				},
				// Both build and remote
				{
					Mode: service.ModeBuild,
					Build: service.Build{
						DockerfilePath: ".tb/repos/TouchBistro/touchbistro-node-boilerplate",
						Target:         "release",
					},
					Remote: service.Remote{
						Image: "123456789.dkr.ecr.us-east-1.amazonaws.com/touchbistro-node-boilerplate",
						Tag:   "a9993e364706816aba3e25717850c26c9cd0d89d",
					},
					Name:         "touchbistro-node-boilerplate",
					RegistryName: "TouchBistro/tb-registry",
				},
			},
			mockAPIClientOpts: docker.MockAPIClientOptions{
				Containers: []dockertypes.Container{
					{
						ID:    "32ce4d8d9c648dd5fce39cf48319da8d55b195513b6fe0cef4a425de9380590c",
						Names: []string{"touchbistro-tb-registry-postgres"},
						Labels: map[string]string{
							docker.ProjectLabel: "tb",
						},
						State: docker.ContainerStateRunning,
					},
					{
						ID:    "f4d2913f1010244b61940cf52845e6dbe5d687791ea185237efe9121adf15edd",
						Names: []string{"touchbistro-tb-registry-touchbistro-node-boilerplate"},
						Labels: map[string]string{
							docker.ProjectLabel: "tb",
						},
						State: docker.ContainerStateRunning,
					},
					// Additional container not part of tb to make sure it is untouched.
					{
						ID:    "e8dc7c16f7dd4be23b96951a34b7ecc69cd727ed13a626a309a96b472646c5e9",
						Names: []string{"test-ubuntu"},
						State: docker.ContainerStateRunning,
					},
				},
				Images: []dockertypes.ImageSummary{
					{
						ID: "sha256:807372352591d91230c1e7a7f4dbaf17a7edaa8283be598e0af73ccbb138c1ac",
						RepoTags: []string{
							"postgres:12",
							"postgres:latest",
						},
					},
					{
						ID: "sha256:145c8bee9bfb11be06b6136000db9d6153f21aee047f23a8fe4048d73dd2628e",
						RepoTags: []string{
							"123456789.dkr.ecr.us-east-1.amazonaws.com/touchbistro-node-boilerplate:a9993e364706816aba3e25717850c26c9cd0d89d",
						},
					},
					// Build image
					{
						ID:       "sha256:08834c4c6f5dc44904536531fb38c8a4b792ba8cc2ce3cbec89bc27752c15ac7",
						RepoTags: []string{"tb_touchbistro-tb-registry-touchbistro-node-boilerplate"},
					},
					{
						ID:       "sha256:f44e5c030356bacda2112f21066ed11d62364e5900f199d5fd217504f594e0ce",
						RepoTags: []string{"my_image:latest"},
					},
				},
				Networks: []dockertypes.NetworkResource{
					{
						ID:   "872eead77c73055de86c8ca6a17d509937c8e8c747b00a40a8374cec721b70e4",
						Name: "tb_default",
						Labels: map[string]string{
							docker.ProjectLabel: "tb",
						},
					},
					{
						ID:   "c3b047926ba1ecbf32ba32c16bdb6886e10d10f7c7a3511481f79b648274616a",
						Name: "my_network",
					},
				},
				Volumes: []volumetypes.Volume{
					{
						Name: "tb_postgres",
						Labels: map[string]string{
							docker.ProjectLabel: "tb",
						},
					},
					{
						Name: "my_volume",
					},
				},
			},
			nukeOpts: engine.NukeOptions{
				RemoveContainers: true,
				RemoveImages:     true,
				RemoveNetworks:   true,
				RemoveVolumes:    true,
			},
			wantContainers: []dockertypes.Container{
				{
					ID:    "e8dc7c16f7dd4be23b96951a34b7ecc69cd727ed13a626a309a96b472646c5e9",
					Names: []string{"test-ubuntu"},
					State: docker.ContainerStateRunning,
				},
			},
			wantImages: []dockertypes.ImageSummary{
				{
					ID:       "sha256:f44e5c030356bacda2112f21066ed11d62364e5900f199d5fd217504f594e0ce",
					RepoTags: []string{"my_image:latest"},
				},
			},
			wantNetworks: []dockertypes.NetworkResource{
				{
					ID:   "c3b047926ba1ecbf32ba32c16bdb6886e10d10f7c7a3511481f79b648274616a",
					Name: "my_network",
				},
			},
			wantVolumes: []volumetypes.Volume{
				{
					Name: "my_volume",
				},
			},
		},
		{
			name: "remove containers if removing other docker resources",
			mockAPIClientOpts: docker.MockAPIClientOptions{
				Containers: []dockertypes.Container{
					{
						ID:    "32ce4d8d9c648dd5fce39cf48319da8d55b195513b6fe0cef4a425de9380590c",
						Names: []string{"touchbistro-tb-registry-postgres"},
						Labels: map[string]string{
							docker.ProjectLabel: "tb",
						},
						State: docker.ContainerStateRunning,
					},
				},
				Networks: []dockertypes.NetworkResource{
					{
						ID:   "872eead77c73055de86c8ca6a17d509937c8e8c747b00a40a8374cec721b70e4",
						Name: "tb_default",
						Labels: map[string]string{
							docker.ProjectLabel: "tb",
						},
					},
				},
				Volumes: []volumetypes.Volume{
					{
						Name: "tb_postgres",
						Labels: map[string]string{
							docker.ProjectLabel: "tb",
						},
					},
				},
			},
			nukeOpts: engine.NukeOptions{
				RemoveNetworks: true,
				RemoveVolumes:  true,
			},
			wantContainers: nil,
			wantNetworks:   nil,
			wantVolumes:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := newServiceCollection(t, tt.services)
			dockerAPIClient := docker.NewMockAPIClient(tt.mockAPIClientOpts)
			e := newEngine(t, engine.Options{
				Services: sc,
				DockerOptions: docker.Options{
					APIClient: dockerAPIClient,
				},
			})

			ctx := context.Background()
			err := e.Nuke(ctx, tt.nukeOpts)
			is := is.New(t)
			is.NoErr(err)

			remainingContainers, err := dockerAPIClient.ContainerList(ctx, dockertypes.ContainerListOptions{All: true})
			is.NoErr(err)
			is.Equal(remainingContainers, tt.wantContainers)

			remainingImages, err := dockerAPIClient.ImageList(ctx, dockertypes.ImageListOptions{All: true})
			is.NoErr(err)
			is.Equal(remainingImages, tt.wantImages)

			remainingNetworks, err := dockerAPIClient.NetworkList(ctx, dockertypes.NetworkListOptions{})
			is.NoErr(err)
			is.Equal(remainingNetworks, tt.wantNetworks)

			listVolumesResult, err := dockerAPIClient.VolumeList(ctx, volumetypes.ListOptions{})
			is.NoErr(err)

			var remainingVolumes []volumetypes.Volume
			for _, v := range listVolumesResult.Volumes {
				remainingVolumes = append(remainingVolumes, *v)
			}
			is.Equal(remainingVolumes, tt.wantVolumes)
		})
	}
}

func newServiceCollection(t *testing.T, services []service.Service) *resource.Collection[service.Service] {
	t.Helper()

	// Create default services if none provided
	if services == nil {
		// Creates two services with the same name but different registries
		// and one service that's a unique name
		services = []service.Service{
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
	}

	var sc resource.Collection[service.Service]
	for _, s := range services {
		if err := sc.Set(s); err != nil {
			t.Fatalf("failed to add service %s to collection: %v", s.FullName(), err)
		}
	}
	return &sc
}

func newPlaylistCollection(t *testing.T, playlists, customPlaylists []playlist.Playlist) *playlist.Collection {
	t.Helper()

	if playlists == nil {
		playlists = []playlist.Playlist{
			{
				Services: []string{
					"TouchBistro/tb-registry/postgres",
					"TouchBistro/tb-registry/touchbistro-node-boilerplate",
				},
				Name:         "backend",
				RegistryName: "TouchBistro/tb-registry",
			},
		}
	}
	if customPlaylists == nil {
		customPlaylists = []playlist.Playlist{
			{
				Services: []string{
					"ExampleZone/tb-registry/postgres",
					"TouchBistro/tb-registry/touchbistro-node-boilerplate",
				},
				Name: "my-backend",
			},
		}
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
	return &pc
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
	if opts.DockerOptions.APIClient == nil {
		opts.DockerOptions.APIClient = docker.NewMockAPIClient(docker.MockAPIClientOptions{})
	}
	if opts.DockerOptions.Config == nil {
		opts.DockerOptions.Config = docker.NewMockConfig(nil)
	}

	e, err := engine.New(opts)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	return e
}
