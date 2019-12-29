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
      migrations: boolean # Does it have migrations that need to be run?
      remote:
        enabled: boolean  # Whether or not to use the remote version
        image: string     # The image name or a valid URI pointing to a remote docker registry.
        tag: string       # The the image tag to use (ex: master)
      repo: string        # The repo name on GitHub
    ```
    Note that `remote` is required only if the service is available on a remote docker registry such as DockerHub or AWS ECR. If it is only built locally the `remote` field and all subfields can be omitted.
2. Add the service to `static/docker-compose.yml`:  
    If the service is available from a remote registry:
    * Add a `x-<sevice-name>-boilerplate` dictionary in the boilerplates section.
    * Add `<service-name>-remote` and `<service-name>` dictionaries to the services section.  

    Otherwise:  
    * Add `<service-name>` directly to the services section.
3. Add the service to any necessary playlists in `static/playlists.yml` (optional):  
    Simply add the service as an entry to the `services` array of any playlist.
