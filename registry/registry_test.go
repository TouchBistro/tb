package registry_test

import (
	"io/fs"
	"sort"
	"testing"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/integrations/simulator"
	"github.com/TouchBistro/tb/registry"
	"github.com/TouchBistro/tb/resource"
	"github.com/TouchBistro/tb/resource/app"
	"github.com/TouchBistro/tb/resource/playlist"
	"github.com/TouchBistro/tb/resource/service"
	"github.com/matryer/is"
)

func TestReadRegistries(t *testing.T) {
	is := is.New(t)
	registries := []registry.Registry{
		{
			Name: "TouchBistro/tb-registry",
			Path: "testdata/registry-1",
		},
		{
			Name: "ExampleZone/tb-registry",
			Path: "testdata/registry-2",
		},
	}
	result, err := registry.ReadAll(registries, registry.ReadAllOptions{
		ReadApps:     true,
		ReadServices: true,
		RootPath:     "/home/test/.tb",
		ReposPath:    "/home/test/.tb/repos",
	})
	is.NoErr(err)

	// Sort slices to make sure different order doesn't cause flakiness
	sort.Strings(result.BaseImages)
	sort.Strings(result.LoginStrategies)
	is.Equal(result.BaseImages, []string{
		"alpine-node",
		"swift",
		"touchbistro/alpine-node:12-runtime",
	})
	is.Equal(result.LoginStrategies, []string{"ecr", "npm"})

	tbPostgres, err := result.Services.Get("TouchBistro/tb-registry/postgres")
	if err != nil {
		t.Fatal("Failed to get service TouchBistro/tb-registry/postgres")
	}

	is.Equal(tbPostgres, service.Service{
		EnvVars: map[string]string{
			"POSTGRES_USER":     "core",
			"POSTGRES_PASSWORD": "localdev",
		},
		Mode:  service.ModeRemote,
		Ports: []string{"5432:5432"},
		Remote: service.Remote{
			Image: "postgres",
			Tag:   "10.6-alpine",
			Volumes: []service.Volume{
				{
					Value:   "postgres:/var/lib/postgresql/data",
					IsNamed: true,
				},
			},
		},
		Name:         "postgres",
		RegistryName: "TouchBistro/tb-registry",
	})

	vcs, err := result.Services.Get("venue-core-service")
	if err != nil {
		t.Fatal("Failed to get service venue-core-service")
	}

	is.Equal(vcs, service.Service{
		Dependencies: []string{
			"touchbistro-tb-registry-postgres",
		},
		EnvFile: "/home/test/.tb/repos/TouchBistro/venue-core-service/.env.example",
		EnvVars: map[string]string{
			"HTTP_PORT": "8080",
			"DB_HOST":   "touchbistro-tb-registry-postgres",
		},
		Mode:   service.ModeRemote,
		Ports:  []string{"8081:8080"},
		PreRun: "yarn db:prepare",
		GitRepo: service.GitRepo{
			Name: "TouchBistro/venue-core-service",
		},
		Build: service.Build{
			Args: map[string]string{
				"NODE_ENV":       "development",
				"NPM_READ_TOKEN": "$NPM_READ_TOKEN",
			},
			Command:        "yarn start",
			DockerfilePath: "/home/test/.tb/repos/TouchBistro/venue-core-service",
			Target:         "dev",
			Volumes: []service.Volume{
				{
					Value: "/home/test/.tb/repos/TouchBistro/venue-core-service:/home/node/app:delegated",
				},
			},
		},
		Remote: service.Remote{
			Command: "yarn serve",
			Image:   "12345.dkr.ecr.us-east-1.amazonaws.com/venue-core-service",
			Tag:     "master",
		},
		Name:         "venue-core-service",
		RegistryName: "TouchBistro/tb-registry",
	})

	ezPostgres, err := result.Services.Get("ExampleZone/tb-registry/postgres")
	if err != nil {
		t.Fatal("Failed to get service ExampleZone/tb-registry/postgres")
	}

	is.Equal(ezPostgres, service.Service{
		EnvVars: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
		},
		Mode:  service.ModeRemote,
		Ports: []string{"5432:5432"},
		Remote: service.Remote{
			Image: "postgres",
			Tag:   "12",
			Volumes: []service.Volume{
				{
					Value:   "postgres:/var/lib/postgresql/data",
					IsNamed: true,
				},
			},
		},
		Name:         "postgres",
		RegistryName: "ExampleZone/tb-registry",
	})

	ves, err := result.Services.Get("venue-example-service")
	if err != nil {
		t.Fatal("Failed to get service venue-example-service")
	}

	is.Equal(ves, service.Service{
		Entrypoint: []string{"bash", "entrypoints/docker.sh", "/home/test/.tb"},
		EnvFile:    "/home/test/.tb/repos/ExampleZone/venue-example-service/.env.compose",
		EnvVars: map[string]string{
			"HTTP_PORT":     "8000",
			"POSTGRES_HOST": "examplezone-tb-registry-postgres",
		},
		Mode:   service.ModeRemote,
		Ports:  []string{"9000:8000"},
		PreRun: "yarn db:prepare:dev",
		GitRepo: service.GitRepo{
			Name: "ExampleZone/venue-example-service",
		},
		Build: service.Build{
			Command:        "yarn start",
			DockerfilePath: "/home/test/.tb/repos/ExampleZone/venue-example-service",
			Target:         "build",
		},
		Remote: service.Remote{
			Command: "yarn serve",
			Image:   "98765.dkr.ecr.us-east-1.amazonaws.com/venue-example-service",
			Tag:     "staging",
		},
		Name:         "venue-example-service",
		RegistryName: "ExampleZone/tb-registry",
	})

	dbPlaylist, err := result.Playlists.Get("db")
	if err != nil {
		t.Fatal("Failed to get db playlist")
	}

	is.Equal(dbPlaylist, playlist.Playlist{
		Services: []string{
			"TouchBistro/tb-registry/postgres",
		},
		Name:         "db",
		RegistryName: "TouchBistro/tb-registry",
	})

	tbCorePlayist, err := result.Playlists.Get("TouchBistro/tb-registry/core")
	if err != nil {
		t.Fatal("Failed to get TouchBistro/tb-registry/core playlist")
	}

	is.Equal(tbCorePlayist, playlist.Playlist{
		Extends: "TouchBistro/tb-registry/db",
		Services: []string{
			"TouchBistro/tb-registry/venue-core-service",
		},
		Name:         "core",
		RegistryName: "TouchBistro/tb-registry",
	})

	ezCorePlaylist, err := result.Playlists.Get("ExampleZone/tb-registry/core")
	if err != nil {
		t.Fatal("Failed to get ExampleZone/tb-registry/core playlist")
	}

	is.Equal(ezCorePlaylist, playlist.Playlist{
		Services: []string{
			"ExampleZone/tb-registry/postgres",
		},
		Name:         "core",
		RegistryName: "ExampleZone/tb-registry",
	})

	ezExampleZonePlaylist, err := result.Playlists.Get("example-zone")
	if err != nil {
		t.Fatal("Failed to get example-zone playlist")
	}

	is.Equal(ezExampleZonePlaylist, playlist.Playlist{
		Extends: "ExampleZone/tb-registry/core",
		Services: []string{
			"ExampleZone/tb-registry/venue-example-service",
		},
		Name:         "example-zone",
		RegistryName: "ExampleZone/tb-registry",
	})

	// Check apps

	gemSwapper, err := result.IOSApps.Get("GemSwapper")
	if err != nil {
		t.Fatal("Failed to get GemSwapper iOS app")
	}

	is.Equal(gemSwapper, app.App{
		BundleID: "com.example.GemSwapper",
		Branch:   "master",
		GitRepo:  "ExampleZone/gem-swapper",
		Storage: app.Storage{
			Provider: "s3",
			Bucket:   "ios-builds",
		},
		Name:         "GemSwapper",
		RegistryName: "ExampleZone/tb-registry",
	})
	is.Equal(gemSwapper.DeviceType(), simulator.DeviceTypeUnspecified)

	iCode, err := result.IOSApps.Get("iCode")
	if err != nil {
		t.Fatal("Failed to get iCode iOS app")
	}

	is.Equal(iCode, app.App{
		BundleID: "com.example.iCode",
		Branch:   "develop",
		GitRepo:  "ExampleZone/iCode",
		RunsOn:   "iPad",
		Storage: app.Storage{
			Provider: "s3",
			Bucket:   "ios-builds",
		},
		Name:         "iCode",
		RegistryName: "ExampleZone/tb-registry",
	})
	is.Equal(iCode.DeviceType(), simulator.DeviceTypeiPad)
}

func TestValidate(t *testing.T) {
	is := is.New(t)
	result := registry.Validate("testdata/registry-2", registry.ValidateOptions{
		Strict: true,
	})
	is.NoErr(result.AppsErr)
	is.NoErr(result.PlaylistsErr)
	is.NoErr(result.ServicesErr)
}

func TestValidateNormalizePath(t *testing.T) {
	is := is.New(t)
	// Make sure the path is normalized and it correctly resolves the registry name using
	// the base name, i.e. `local/registry-2`. Before there was a bug where the last part of the
	// path was taken as is and it lead to issues because `local/.` is not a valid registry name.
	result := registry.Validate("testdata/registry-2/.", registry.ValidateOptions{})
	is.NoErr(result.AppsErr)
	is.NoErr(result.PlaylistsErr)
	is.NoErr(result.ServicesErr)
}

func TestValidateErrors(t *testing.T) {
	is := is.New(t)
	result := registry.Validate("testdata/invalid-registry-1", registry.ValidateOptions{
		Strict: true,
	})

	var errs errors.List
	var validationErr *resource.ValidationError

	is.True(errors.As(result.AppsErr, &errs))
	is.Equal(len(errs), 1)
	is.True(errors.As(errs[0], &validationErr))
	is.Equal(validationErr.Resource.Type(), resource.TypeApp)
	is.Equal(validationErr.Resource.FullName(), "local/invalid-registry-1/GemSwapper")

	is.True(errors.Is(result.PlaylistsErr, fs.ErrNotExist))

	is.True(errors.As(result.ServicesErr, &errs))
	is.Equal(len(errs), 3)
	var serviceErrs []*resource.ValidationError
	for _, err := range errs {
		var ve *resource.ValidationError
		is.True(errors.As(err, &ve))
		is.Equal(ve.Resource.Type(), resource.TypeService)
		serviceErrs = append(serviceErrs, ve)
	}
	// The order the services are unmarshed in is not guaranteed so sort them
	// to make sure the test isn't flaky
	sort.Slice(serviceErrs, func(i, j int) bool {
		return serviceErrs[i].Resource.FullName() < serviceErrs[j].Resource.FullName()
	})
	is.Equal(serviceErrs[0].Resource.FullName(), "local/invalid-registry-1/postgres")
	is.Equal(serviceErrs[1].Resource.FullName(), "local/invalid-registry-1/venue-core-service")
	is.Equal(serviceErrs[2].Resource.FullName(), "local/invalid-registry-1/venue-example-service")
}
