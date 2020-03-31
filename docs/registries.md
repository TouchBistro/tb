# Registries

A registry is a GitHub repo that provides configuration of apps, playlists, and services that can be run with `tb`.

A registry has the following directory structure:
```
apps.yml
playlists.yml
services.yml
static/
```

All files are optional. A `static` directory can be present with files that can be referenced by services in `services.yml`.

## Using Registries

To use a registry add it to the `registries` section of your `~/.tbrc.yml`. Registries are always of the form `org/repo`.

Ex:
```yaml
registries:
  - name: TouchBistro/tb-registry
```

All services, playlists, and apps in a registry are scoped by the name of that registry to ensure they are globally unique. If a service, playlist or app name is unique, however you can use this name directly in commands and `tb` will figure out which service you are referring to.

For example if there is a service named `postgres` in the registry `TouchBistro/tb-registry`, you can run it with the following command:
```
tb up -s TouchBistro/tb-registry/postgres
```

If this is the only service named `postgres` that `tb` knows about, the following also works:
```
tb up -s postgres
```

`tb` will report an error if a service named `postgres` is found in multiple registries.

### Testing changes to a registry

To have `tb` validate your changes to make sure it is able load the configs run the following:
```
tb registry validate <path>
```

For example if you are currently in the root directory of your registry you would run:
```
tb registry validate .
```

For more robust testing you can temporarily tell `tb` to use your local version of the registry instead of the version on GitHub.
To do that add a `localPath` field to the registry and set it to the path of the registry on your machine in your `~/.tbrc.yml`.

Ex:
```yaml
registries:
  - name: TouchBistro/tb-registry
    localPath: ~/Development/tb-registry
```

## Configuring Apps

Apps are configured in `apps.yml`.

The schema of the file is:
```yaml
iosApps: map
desktopApps: map
```

The schema of an iOS app is:
```yaml
<name>:
  bundleID: string             # The bundle ID of the iOS app
  branch: string               # The base branch of the repo, ex: master
  repo: string                 # The repo name on GitHub, format: org/repo
  envVars: map<string, string> # Env vars to set for the app
  storage:
    provider: s3   # The storage provider to use
    bucket: string # The name of the bucket the builds are stored in
```

The schema of a desktop app is:
```yaml
<name>:
  branch: string               # The base branch of the repo, ex: master
  repo: string                 # The repo name on GitHub, format: org/repo
  envVars: map<string, string> # Env vars to set for the app
  storage:
    provider: s3   # The storage provider to use
    bucket: string # The name of the bucket the builds are stored in
```

## Configuring Services

Services are configured in `services.yml`.

The schema of the file is:
```yaml
global:
  baseImages: string[]           # A list of docker images to pull before building containers.
  loginStrategies: string[]      # A list of login strategies to run, valid values: ecr, npm
  variables: map<string, string> # Variables that can be used in service definitions
services: map<string, Service> # The services that can be run
```

### Login Strategies

`tb` can ensure you are logged in to certain 3rd party services before running your services.

The following login strategies are avaible:
* `ecr`: Performs `docker login` to AWS ECR images can be pulled from there. This strategy assumes you have your AWS account configured locally using the `aws cli`.
* `npm`: Enures you are logged in to the NPM registry and have the environment variable `NPM_TOKEN` set.

We welcome PRs for additional login strategies if these don't meet your needs.

### Adding a new service

To add a new service add an entry to the `services` field in `services.yml`.

The schema is as follows:
```yaml
<service-name>:
  dependencies: string[]       # Any services that this service requires to run (eg postgres)
  entrypoint: string           # Custom Docker entrypoint
  envFile: string              # Path to env file
  envVars: map<string, string> # Env vars to set for the services
  mode: remote | build         # What mode to use: remote or build
  ports: string[]              # List of ports to expose
  preRun: string               # Script to run before starting the service, e.g. 'yarn db:prepare' to run db migrations
  repo:
    name: string # The repo name on GitHub, format: org/repo
  build:
    args: map<string, string> # List of args to pass to docker build
    command: string           # Command to run when container starts
    dockerfilePath: string    # Path to the Dockerfile
    target: string            # Target to build in a multi-stage build
    volumes:                  # List of docker volumes to create
      - value: string  # The volume to create
        named: boolean # Whether or not to create a named volume
  remote:
    command: string  # Command to run when the container starts
    image: string    # The image name or a valid URI pointing to a remote docker registry.
    tag: string      # The the image tag to use (ex: master)
    volumes:         # List of docker volumes to create
      - value: string  # The volume to create
        named: boolean # Whether or not to create a named volume
```
At least one of `build` or `remote` are required. `build` is only required if the service can be built locally with `docker build`, `remote` is only required if the service can be pulled from a remote registry with `docker pull`.

Any unneeded fields can be omitted.

#### Variable Expansion

Variable expansion is supported by the following fields in a service:
* `dependencies`
* `envFile`
* `envVars`
* `build.dockerfilePath`
* `build.volumes.value`
* `remote.image`
* `remote.volumes.value`

Variable expansion is done by placing the variable name inside `${}`.
Ex:
```yml
envFile: ${path}/.env
```

There are two types are variables that can be used:
1. User defined variables. These are variables set in the `global.variables` field of `services.yml`.
2. Builtin variables. These are variables provided by `tb` and are automatically set when `services.yml` is read. Builtin variables are prefixed with `@` to distinguish them from user defined variables.

`tb` provides the following builtin variables:
* `@ROOTPATH`: The path to `tb`'s root directory, i.e. `~/.tb`.
* `@STATICPATH`: The path to your registry's `static` directory. Use this variable to reference static files in your registry.
* `@REPOPATH`: The path to the current service's cloned git repo. This is set for each service.

Additionally `tb` will also create a builtin variable that is the name of each service. This variable can be used to reference the containers of other services for fields like `dependencies`. This is useful because the resulting container name will not be the name of your service directly.

Ex:
If you have a service named `postgres` and need it as a dependency for another service you could do:
```yaml
dependencies:
  - ${@postgres}
```

## Configuring Playlists

A playlist is a collection of services that can be run with `tb up`.

Playlists are configured by adding entries to `playlists.yml`.

The schema is as follows:
```yaml
<playlist-name>:
  extends: string # A playlist to extend, i.e. add the services from that playlist to this playlist
  services: string[] # A list of services in the playlist
```
The `extends` field is optional.

All services listed in a playlist are assumed to exist in the same registry. It is not possible to use services from a different registry in a playlist.
