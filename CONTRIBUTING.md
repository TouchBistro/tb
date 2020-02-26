# Contributing

The following document outlines how to contribute to the `tb` project. If all you want to do is add/modify a service you can skip to the [Adding a new Service](#adding-a-new-service) section.

### **Table of Contents**
- [Requirements](#requirements)
- [Setup](#setup)
- [Building](#building)
    + [Running locally](#running-locally)
    + [Running globally](#running-globally)
    + [Remove global build](#remove-global-build)
- [Adding a new Service](#adding-a-new-service)

## Requirements

To build and run `tb` locally you will need to install go.
This can be done through homebrew by running:
```sh
brew install go
```

## Setup
First clone the repo to your desired location:
```sh
git clone git@github.com:TouchBistro/tb.git
```

Then in the root of the repo run the following to install all dependencies and tools required:
```sh
make setup
```

## Building
### Running locally
To build the app run:
```sh
go build
```

This will create a binary named `tb` in the current directory. You can run it be doing `./tb`.

### Running globally
If you want to be able to run if from anywhere you can run:
```sh
go install
```

This will installing it in the `bin` directory in your go path.

**NOTE:** You will need to add the go bin to your `PATH` variable.
Add the following to your `.zshrc` or `.bash_profile`:
```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Then run `source ~/.zshrc` or `source ~/.bash_profile`.

### Remove global build
The global build will likely take precedence over the version installed with brew. This is fine during development but might be annoying otherwise.

To remove the globally installed build run the following from the root directory of the repo:
```sh
make go-uninstall
```

## Adding a new service

To add a new service do the following:

1. Add it to `static/services.yml`:  
    The format is as follows:
    ```yaml
    <service-name>:
      dependencies: string[] # Any services that this service requires to run (eg postgres)
      entrypoint: string     # Custom Docker entrypoint
      envFile: string        # Path to env file
      envVars: map           # Env vars to set for the services
      ports: string[]        # List of ports to expose
      preRun: string         # Script to run before starting the service, e.g. 'yarn db:prepare' to run db migrations
      repo: string           # The repo name on GitHub
      build:
        args: map              # List of args to pass to docker build
        command: string        # Command to run when container starts
        dockerfilePath: string # Path to the Dockerfile
        target: string         # Target to build in a multi-stage build
        volumes:               # List of docker volumes to create
         - value: string  # The volume to create
           named: boolean # Whether or not to create a named volume
      remote:
        command: string  # Command to run when the container starts
        enabled: boolean # Whether or not to use the remote version
        image: string    # The image name or a valid URI pointing to a remote docker registry.
        tag: string      # The the image tag to use (ex: master)
        volumes:         # List of docker volumes to create
         - value: string  # The volume to create
           named: boolean # Whether or not to create a named volume
    ```
    At least one of `build` or `remote` are required. `build` is only required if the service can be built locally with `docker build`, `remote` is only required if the service can be pulled from a remote registry with `docker pull`.

    Any unneeded fields can be omitted.
2. Add the service to any necessary playlists in `static/playlists.yml` (optional):  
    Simply add the service as an entry to the `services` array of any playlist.

3. If `go.sum` gets modified on build, please run `go mod tidy` to clean it up before committing and pushing changes.
