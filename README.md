# `tb`

tb is a CLI for running TouchBistro services on a development machine.

It is aimed at making local development easy in a complicated microservices architecture by provisioning your machine with the dependencies you need and making it easy for you to run them on your machine in an environment that is close to how they run in production.

### **Table of Contents**
- [Requirements](#requirements)
    + [Installed Software](#installed-software)
    + [AWS ECR](#aws-ecr)
    + [SSH Key](#ssh-key)
- [Installation](#installation)
- [Quickstart](#quickstart)
- [Commands](#commands)
- [Configuration](#configuration)
    + [Changing log level](#changing-log-level)
    + [Adding custom playlists](#adding-custom-playlists)
    + [Overriding service properties](#overriding-service-properties)
- [Contributing](#contributing)
- [Having trouble?](#having-trouble?)
- [Gotchas / Tips](#Gotchas-/-Tips)

## Requirements

### Installed Software

Right now, the only requirement is that you have the xcode cli tools and `nvm`.

This project will install and manage all other dependencies that you need.

### AWS ECR

You will need access to AWS ECR (Amazon's docker registry) to pull artifacts instead of having `tb` build them on the host.

Once you have been provided access to our AWS account by DevOps Support, create a personal access key and make note of your secret key.

Configure your AWS CLI credentials by running `aws configure` (use `us-east-1` for region).

### SSH Key
The following instructions assume you have an ssh key connected to your GitHub account. If you do not have one, please create on by following the instructions [here](https://help.github.com/en/articles/connecting-to-github-with-ssh).

## Installation

`tb` is available through TouchBistro's `homebrew` tap. If you do not have homebrew, you can install it by going to [brew.sh](https://brew.sh)

1. Add Touchbistro's tap to get access to all the available tools:
    ```sh
    brew tap touchbistro/tap git@github.com:TouchBistro/homebrew-tap.git
    ```

2. Create a GitHub Access Token
    - Create the token with the `repo` box checked in the list of premissions. Follow the instructions [here](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) to learn more.
    - Make sure you copy the token when you create it!
    - After the token has been created, enable SSO for it.
    - Add the following to your `.bash_profile` or `.zshrc`:
    ```sh
    export HOMEBREW_GITHUB_API_TOKEN=<YOUR_TOKEN>
    ```

3. Install `tb` with brew
    ```sh
    brew install tb
    ```

## Quickstart

`tb` will configure itself and install any necessary dependencies when it run. To get started run `tb up -s postgres` to setup your system and start a `postgresql` service running in a docker container.

The `-s` or `--services` flag starts a list of services. You can also run a playlist which is a predefined set of services, by using the `-p` or `--playlist` flag.

Let's try this out now:
1. Exit lazydocker by hitting `q`. This does not stop the docker containers however, which are running in the background.
2. Run `tb down`, this will stop any running docker containers and remove them.
3. Run `tb up -p core`. This will start all services defined in the `core` playlist.

Try running `tb --help` or `tb up --help` to see what else you can do.

## Commands

`tb` comes with a lot of convenient commands. See the documentation [here](docs/tb.md) for the command documentation.

Run `tb --help` to see the commands available. Run `tb <cmd> --help` to get help on a specific command.

`tb` also provides man pages which can be viewed by running `man tb` or `man tb-<cmd>` for a specific command.

## Configuration

`tb` can be configured through the `.tbrc.yml` file located in your home directory. `tb` will automatically create a basic `.tbrc.yml` for you if one doesn't exist.

### Changing log level
The default log level is `info` which will only print minimal logs with only brief descriptions of what is happening. If you would like more detailed logs, set the `log-level` property to `debug`.

### Adding custom playlists
You can create custom playlists by adding a new object to the `playlists` property.

Example:
```yaml
playlists:
  my-playlist:
    extends: core
    services:
      - venue-admin-frontend
      - partners-config-service
```

Each playlist can extend another playlist though the use of the `extends` property. This will add all the services from the playlist being extended to this playlist.

The services in the playlist are specified in the `services` property.

### Overriding service properties
You can override certain properties for services. To do this use the `overrides` property.

Example:
```yaml
overrides:
  mokta:
    erc: false
  venue-admin-frontend:
    ecr: true
    ecrTag: <branch or commit SHA>
```

You can disable ECR by setting `ecr: false`, which will cause an image to be built from the local repo instead of pulling an image from ECR.

You can also use a specific ECR tag by setting the `ecrTag` property. This can be the name of a branch on GitHub or a commit SHA (must be the full SHA not the shortened one).

**IMPORTANT:** If you set `ecrTag` you must also set `ecr: true` for everything to work properly!

## Contributing

See [contributing](CONTRIBUTING.md) for instructions on how to contribute to `tb`.

## Having trouble?

Check the [FAQ](docs/FAQ.md) for common problems and solutions. (Pull requests welcome!)

## Gotchas / Tips

- Do not run npm run or npm run commands from the host unless you absolutely need to.

- **Previous Setup**: If you already have `postgres.app` or are running postgres with homebrew or any other way, Datagrip-like tools will be confused about which pg to connect to. You won't need these anymore, so you can just delete them. Use `pgrep postgres` and make sure you don't have any other instances running.

- **SQL EDITORS**: To use external db tools like datagrip or `psql`, keep `CORE_DB_HOST` in the .env file as it is, but use `localhost` as the hostname in datagrip (or tool of choice). see `bin/db` for an example that uses `pgcli` on the host. Inside the docker network, containers uses the service names in `docker-compose.yml` as their hostname. Externally, their hostname is just `localhost`.

- **Slowness**: If running things in Docker on a mac is slow, allocate more CPUs, Memory and Swap space using the Docker For Mac advanced preferences. Keep in mind that some tools (like `jest`) have threading issues on linux and are not going to be faster with more cores. Use `docker stats` to see resource usage by image.
