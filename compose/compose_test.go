package compose

import (
	"bytes"
	"testing"

	"github.com/TouchBistro/tb/resource/service"
	"github.com/stretchr/testify/assert"
)

func TestCreateComposeFile(t *testing.T) {
	assert := assert.New(t)

	services := []service.Service{
		{
			EnvVars: map[string]string{
				"POSTGRES_USER":     "user",
				"POSTGRES_PASSWORD": "password",
			},
			Mode: service.ModeRemote,
			Ports: []string{
				"5432:5432",
			},
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
		},
		{
			EnvVars: map[string]string{
				"DB_USER":     "core",
				"DB_PASSWORD": "localdev",
			},
			Mode: service.ModeRemote,
			Ports: []string{
				"5432:5432",
			},
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
		},
		{
			Dependencies: []string{
				"touchbistro-tb-registry-postgres",
			},
			EnvFile: ".tb/repos/TouchBistro/venue-core-service/.env.example",
			EnvVars: map[string]string{
				"HTTP_PORT": "8080",
				"DB_HOST":   "touchbistro-tb-registry-postgres",
			},
			Mode: service.ModeBuild,
			Ports: []string{
				"8081:8080",
			},
			PreRun: "yarn db:prepare",
			GitRepo: service.GitRepo{
				Name: "TouchBistro/venue-core-service",
			},
			Build: service.Build{
				Args: map[string]string{
					"NODE_ENV":  "development",
					"NPM_TOKEN": "$NPM_TOKEN",
				},
				Command:        "yarn start",
				DockerfilePath: ".tb/repos/TouchBistro/venue-core-service",
				Target:         "dev",
				Volumes: []service.Volume{
					{
						Value: ".tb/repos/TouchBistro/venue-core-service:/home/node/app:delegated",
					},
				},
			},
			Name:         "venue-core-service",
			RegistryName: "TouchBistro/tb-registry",
		},
	}
	sc := &service.Collection{}
	for _, s := range services {
		if err := sc.Set(s); err != nil {
			assert.FailNow("Failed to create ServiceCollection", err)
		}
	}

	expectedComposeFile := `# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY

version: "3.7"
services:
    examplezone-tb-registry-postgres:
        container_name: examplezone-tb-registry-postgres
        environment:
            POSTGRES_PASSWORD: password
            POSTGRES_USER: user
        image: postgres:12
        ports:
            - 5432:5432
        volumes:
            - postgres:/var/lib/postgresql/data
    touchbistro-tb-registry-postgres:
        container_name: touchbistro-tb-registry-postgres
        environment:
            DB_PASSWORD: localdev
            DB_USER: core
        image: postgres:10.6-alpine
        ports:
            - 5432:5432
        volumes:
            - postgres:/var/lib/postgresql/data
    touchbistro-tb-registry-venue-core-service:
        build:
            args:
                NODE_ENV: development
                NPM_TOKEN: $NPM_TOKEN
            context: .tb/repos/TouchBistro/venue-core-service
            target: dev
        command: yarn start
        container_name: touchbistro-tb-registry-venue-core-service
        depends_on:
            - touchbistro-tb-registry-postgres
        env_file:
            - .tb/repos/TouchBistro/venue-core-service/.env.example
        environment:
            DB_HOST: touchbistro-tb-registry-postgres
            HTTP_PORT: "8080"
        ports:
            - 8081:8080
        volumes:
            - .tb/repos/TouchBistro/venue-core-service:/home/node/app:delegated
volumes:
    postgres: null
`

	buf := &bytes.Buffer{}
	err := CreateComposeFile(sc, buf)

	assert.NoError(err)
	assert.Equal(expectedComposeFile, buf.String())
}
