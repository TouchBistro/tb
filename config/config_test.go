package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TouchBistro/tb/config"
	"github.com/TouchBistro/tb/registry"
	"github.com/matryer/is"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		data string
		want func(homedir string) config.Config
	}{
		{
			name: "basic tbrc",
			data: `experimental: true
registries:
  - name: TouchBistro/tb-registry
  - name: ExampleZone/tb-registry
    localPath: ~/tools/tb-registry`,
			want: func(homedir string) config.Config {
				return config.Config{
					ExperimentalMode: true,
					Registries: []registry.Registry{
						{
							Name:      "TouchBistro/tb-registry",
							LocalPath: "",
						},
						{
							Name:      "ExampleZone/tb-registry",
							LocalPath: "~/tools/tb-registry",
						},
					},
				}
			},
		},
		{
			name: "no tbrc",
			want: func(homedir string) config.Config {
				return config.Config{}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			configPath := filepath.Join(tmpdir, ".tbrc.yml")
			if tt.data != "" {
				err := os.WriteFile(configPath, []byte(tt.data), 0o644)
				if err != nil {
					t.Fatalf("failed to write file %s: %v", configPath, err)
				}
			}

			cfg, err := config.Read(tmpdir)
			is := is.New(t)
			is.NoErr(err)
			is.Equal(cfg, tt.want(tmpdir))
		})
	}
}

type addRegistryTest struct {
	name         string
	registryName string
	existingTBRC string
	expectedTBRC string
	err          error
}

var addRegistryTests = []addRegistryTest{
	{
		name:         "no existing registries",
		registryName: "TouchBistro/tb-registry",
		existingTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		expectedTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
registries:
  - name: TouchBistro/tb-registry
`,
		err: nil,
	},
	{
		name:         "empty registries list",
		registryName: "TouchBistro/tb-registry",
		existingTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		expectedTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
	},
	{
		name:         "adding a new registry",
		registryName: "TouchBistro/tb-registry-example",
		existingTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
    localPath: ~/registries/TouchBistro/tb-registry
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		expectedTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
    localPath: ~/registries/TouchBistro/tb-registry
  - name: TouchBistro/tb-registry-example
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		err: nil,
	},
	{
		name:         "registry already exists",
		registryName: "TouchBistro/tb-registry",
		existingTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		expectedTBRC: `# Toggle debug mode for more verbose logging
debug: false
# Toggle experimental mode to test new features
experimental: false
# Add registries to access their services and playlists
# A registry corresponds to a GitHub repo and is of the form <org>/<repo>
registries:
  - name: TouchBistro/tb-registry
# Custom playlists
# Each playlist can extend another playlist as well as define its services
playlists:
  online-ordering:
    services:
      - online-ordering-service
`,
		err: config.ErrRegistryExists,
	},
}

func TestAddRegistry(t *testing.T) {
	for _, test := range addRegistryTests {
		t.Run(test.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			tbrcPath := filepath.Join(tmpdir, ".tbrc.yml")
			err := os.WriteFile(tbrcPath, []byte(test.existingTBRC), 0o644)
			if err != nil {
				t.Fatalf("failed to write file %s: %v", tbrcPath, err)
			}
			if _, err := config.Read(tmpdir); err != nil {
				t.Fatalf("failed to load tbrc: %v", err)
			}

			err = config.AddRegistry(test.registryName, tmpdir)
			is := is.New(t)
			is.Equal(err, test.err)

			data, err := os.ReadFile(tbrcPath)
			if err != nil {
				t.Fatalf("Failed to read tbrc file: %v", err)
			}
			is.Equal(string(data), test.expectedTBRC)
		})
	}
}
