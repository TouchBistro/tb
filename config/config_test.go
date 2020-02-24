package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const fixtures = "../_fixtures"
const recipe1Path = fixtures + "/recipe-1"
const recipe2Path = fixtures + "/recipe-2"
const composePath = fixtures + "/docker-compose.yml"

const tbrcYaml = `debug: true
experimental: true
recipes:
  - name: TouchBistro/tb-recipe
    localPath: ../_fixtures/recipe-1
  - name: ExampleZone/tb-recipe
    localPath: ../_fixtures/recipe-2
playlists:
  my-core:
    extends: TouchBistro/tb-recipe/core
    services:
      - partners-config-service
overrides:
  TouchBistro/tb-recipe/postgres:
    remote:
      enabled: true
      tag: 12-alpine
  venue-core-service:
    envVars:
      AUTH_URL: http://localhost:8002/auth
    preRun: yarn db:prepare:dev
    build:
      command: 'tail -f /dev/null'
      target: release
    remote:
      command: 'tail -f /dev/null'
      enabled: true
      tag: feat/delete-menu
`

func setup(tbrcStr string) (string, error) {
	tbrc = UserConfig{}
	dir, err := ioutil.TempDir("", "tb_config_test")

	if tbrcStr != "" {
		rcPath := filepath.Join(dir, tbrcFileName)
		err = ioutil.WriteFile(rcPath, []byte(tbrcYaml), 0644)
		if err != nil {
			return "", err
		}
	}

	os.Setenv("HOME", dir)
	return dir, err
}

func TestInitDefaultTBRC(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup("")
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	expectedTBRC := UserConfig{}

	err = Init(InitOptions{})

	assert.NoError(err)
	assert.Equal(expectedTBRC, tbrc)
	assert.Equal(log.GetLevel(), log.InfoLevel)
	assert.Equal(filepath.Join(dir, ".tb"), TBRootPath())
	assert.NotEmpty(TBRootPath())
	assert.DirExists(TBRootPath())
	assert.FileExists(filepath.Join(dir, tbrcFileName))
}

func TestInitCustomTBRC(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup(tbrcYaml)
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	cwd, err := os.Getwd()
	if err != nil {
		assert.FailNow("Failed to get cwd", err)
	}

	expectedTBRC := UserConfig{
		DebugEnabled:     true,
		ExperimentalMode: true,
		Recipes: []Recipe{
			Recipe{
				Name:      "TouchBistro/tb-recipe",
				LocalPath: recipe1Path,
				Path:      filepath.Join(cwd, recipe1Path),
			},
			Recipe{
				Name:      "ExampleZone/tb-recipe",
				LocalPath: recipe2Path,
				Path:      filepath.Join(cwd, recipe2Path),
			},
		},
		Playlists: map[string]Playlist{
			"my-core": Playlist{
				Extends: "TouchBistro/tb-recipe/core",
				Services: []string{
					"partners-config-service",
				},
			},
		},
		Overrides: map[string]serviceOverride{
			"TouchBistro/tb-recipe/postgres": {
				Remote: remoteOverride{
					Enabled: true,
					Tag:     "12-alpine",
				},
			},
			"venue-core-service": {
				EnvVars: map[string]string{
					"AUTH_URL": "http://localhost:8002/auth",
				},
				PreRun: "yarn db:prepare:dev",
				Build: buildOverride{
					Command: "tail -f /dev/null",
					Target:  "release",
				},
				Remote: remoteOverride{
					Command: "tail -f /dev/null",
					Enabled: true,
					Tag:     "feat/delete-menu",
				},
			},
		},
	}

	err = Init(InitOptions{})

	assert.NoError(err)
	assert.Equal(expectedTBRC, tbrc)
}

func TestInitCreateLazydockerConfig(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup("")
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	tbrc.ExperimentalMode = true
	ldDirPath := filepath.Join(dir, "Library/Application Support/jesseduffield/lazydocker")
	ldConfigPath := filepath.Join(ldDirPath, "lazydocker.yml")

	err = Init(InitOptions{})

	assert.NoError(err)
	assert.DirExists(ldDirPath)
	assert.FileExists(ldConfigPath)

	ldConfig, err := ioutil.ReadFile(ldConfigPath)

	assert.NoError(err)
	assert.Equal(lazydockerConfig, string(ldConfig))
}

func TestInitLoadServices(t *testing.T) {
	assert := assert.New(t)
	dir, err := setup(tbrcYaml)
	if err != nil {
		assert.FailNow("Failed to setup tmp dir", err)
	}
	defer os.RemoveAll(dir)

	rp := filepath.Join(dir, ".tb/repos")
	pcsPath := filepath.Join(rp, "TouchBistro/partners-config-service")
	vcsPath := filepath.Join(rp, "TouchBistro/venue-core-service")
	vesPath := filepath.Join(rp, "ExampleZone/venue-example-service")
	expectedServiceConfig := ServiceConfig{
		BaseImages: []string{
			"swift",
			"touchbistro/alpine-node:12-runtime",
			"alpine-node",
		},
		LoginStrategies: []string{
			"ecr",
			"npm",
		},
		Services: ServiceListMap{
			"partners-config-service": []Service{
				Service{
					Dependencies: []string{
						"postgres",
					},
					EnvFile: filepath.Join(pcsPath, ".env.example"),
					EnvVars: map[string]string{
						"DB_HOST":   "postgres",
						"HTTP_PORT": "8080",
					},
					Ports: []string{
						"8090:8080",
					},
					PreRun:  "yarn db:prepare",
					GitRepo: "TouchBistro/partners-config-service",
					Build: build{
						Args: map[string]string{
							"NODE_ENV":  "development",
							"NPM_TOKEN": "$NPM_TOKEN",
						},
						Command:        "yarn start",
						DockerfilePath: pcsPath,
						Target:         "dev",
						Volumes: []volume{
							volume{
								Value: pcsPath + ":/home/node/app:delegated",
							},
							volume{
								Value:   "partners-config-service-node_modules:/home/node/app/node_modules",
								IsNamed: true,
							},
						},
					},
					Remote: remote{
						Command: "yarn serve",
						Enabled: false,
						Image:   "12345.dkr.ecr.us-east-1.amazonaws.com/partners-config-service",
						Tag:     "master",
					},
					RecipeName: "TouchBistro/tb-recipe",
				},
			},
			"postgres": []Service{
				Service{
					EnvVars: map[string]string{
						"POSTGRES_USER":     "user",
						"POSTGRES_PASSWORD": "password",
					},
					Ports: []string{
						"5432:5432",
					},
					Remote: remote{
						Enabled: true,
						Image:   "postgres",
						Tag:     "12",
						Volumes: []volume{
							volume{
								Value:   "postgres:/var/lib/postgresql/data",
								IsNamed: true,
							},
						},
					},
					RecipeName: "ExampleZone/tb-recipe",
				},
				Service{
					EnvVars: map[string]string{
						"POSTGRES_USER":     "core",
						"POSTGRES_PASSWORD": "localdev",
					},
					Ports: []string{
						"5432:5432",
					},
					Remote: remote{
						Enabled: true,
						Image:   "postgres",
						Tag:     "12-alpine",
						Volumes: []volume{
							volume{
								Value:   "postgres:/var/lib/postgresql/data",
								IsNamed: true,
							},
						},
					},
					RecipeName: "TouchBistro/tb-recipe",
				},
			},
			"venue-core-service": []Service{
				Service{
					Dependencies: []string{
						"postgres",
					},
					EnvFile: filepath.Join(vcsPath, ".env.example"),
					EnvVars: map[string]string{
						"HTTP_PORT": "8080",
						"DB_HOST":   "postgres",
						"AUTH_URL":  "http://localhost:8002/auth",
					},
					Ports: []string{
						"8081:8080",
					},
					PreRun:  "yarn db:prepare:dev",
					GitRepo: "TouchBistro/venue-core-service",
					Build: build{
						Args: map[string]string{
							"NODE_ENV":  "development",
							"NPM_TOKEN": "$NPM_TOKEN",
						},
						Command:        "tail -f /dev/null",
						DockerfilePath: vcsPath,
						Target:         "release",
						Volumes: []volume{
							volume{
								Value: vcsPath + ":/home/node/app:delegated",
							},
						},
					},
					Remote: remote{
						Command: "tail -f /dev/null",
						Enabled: true,
						Image:   "12345.dkr.ecr.us-east-1.amazonaws.com/venue-core-service",
						Tag:     "feat/delete-menu",
					},
					RecipeName: "TouchBistro/tb-recipe",
				},
			},
			"venue-example-service": []Service{
				Service{
					Entrypoint: []string{"bash", "entrypoints/docker.sh"},
					EnvFile:    filepath.Join(vesPath, ".env.compose"),
					EnvVars: map[string]string{
						"HTTP_PORT":     "8000",
						"POSTGRES_HOST": "postgres",
					},
					Ports: []string{
						"9000:8000",
					},
					PreRun:  "yarn db:prepare:dev",
					GitRepo: "ExampleZone/venue-example-service",
					Build: build{
						Target:         "build",
						Command:        "yarn start",
						DockerfilePath: vesPath,
					},
					Remote: remote{
						Command: "yarn serve",
						Enabled: true,
						Image:   "98765.dkr.ecr.us-east-1.amazonaws.com/venue-example-service",
						Tag:     "staging",
					},
					RecipeName: "ExampleZone/tb-recipe",
				},
			},
		},
	}

	expectedPlaylists := map[string][]Playlist{
		"core": []Playlist{
			Playlist{
				Services: []string{
					"ExampleZone/tb-recipe/postgres",
				},
				RecipeName: "ExampleZone/tb-recipe",
			},
			Playlist{
				Services: []string{
					"TouchBistro/tb-recipe/postgres",
					"TouchBistro/tb-recipe/venue-core-service",
				},
				RecipeName: "TouchBistro/tb-recipe",
			},
		},
		"example-zone": []Playlist{
			Playlist{
				Extends: "ExampleZone/tb-recipe/core",
				Services: []string{
					"ExampleZone/tb-recipe/venue-example-service",
				},
				RecipeName: "ExampleZone/tb-recipe",
			},
		},
	}

	err = Init(InitOptions{
		LoadServices: true,
	})

	// Sort slices so assert.Equal works
	sort.Slice(serviceConfig.Services["postgres"], func(i, j int) bool {
		return serviceConfig.Services["postgres"][i].RecipeName < serviceConfig.Services["postgres"][j].RecipeName
	})

	sort.Slice(playlists["core"], func(i, j int) bool {
		return playlists["core"][i].RecipeName < playlists["core"][j].RecipeName
	})

	assert.NoError(err)
	assert.ElementsMatch(expectedServiceConfig.BaseImages, serviceConfig.BaseImages)
	assert.ElementsMatch(expectedServiceConfig.LoginStrategies, serviceConfig.LoginStrategies)
	assert.Equal(expectedServiceConfig.Services, serviceConfig.Services)
	assert.Equal(expectedPlaylists, playlists)

	b, err := ioutil.ReadFile(composePath)
	if err != nil {
		assert.FailNow("Failed to read expected docker-compose.yml file", err)
	}
	expectedComposeFile := strings.ReplaceAll(string(b), "$HOME", dir)

	composeFile, err := ioutil.ReadFile(filepath.Join(TBRootPath(), "docker-compose.yml"))
	if err != nil {
		assert.FailNow("Failed to read generated docker-compose.yml file", err)
	}

	assert.Equal(expectedComposeFile, string(composeFile))
}
