# `tb`

tb is a CLI for running TouchBistro services on a development machine.

It is aimed at making local development easy in a complicated microservices architecture by provisioning your machine with the dependencies you need and making it easy for you to run them on your machine in an environment that is close to how they run in production.

## Requirements

### Installed Software

Right now, the only requirement is that you have the xcode cli tools and `nvm`.

This project will install and manage all other dependencies that you need.

### AWS ECR

You will need access to AWS ECR (Amazon's docker registry) to pull artifacts instead of having `tb` build them on the host.

Once you have been provided access to our AWS account by DevOps Support, create a personal access key and make note of your secret key.

Configure your AWS CLI credentials by running `aws configure` (use `us-east-1` for region).

## Quickstart

TBD: Add instructions for getting binary from homebrew when we set that app

Run `tb up -s postgres` to setup your system and start a `postgresql` service running in a docker container. Try running `tb --help` or `tb up --help` to see what else you can do.

## Configuration

`tb` can be configured to either build images/containers locally, or to pull existing images from ECR. This is all set in `config.yml` with the `ecr` and `imageURI` flags.

## Contributing

If you want to work on `tb` rather than just use it, you will need to install `go`.

The easiest way to do so is with homebrew. `brew install go`.

After that, you just need to clone the repo.

TODO: Tell people what to do if they just want to add serivces.

## Commands

`tb` comes with a lot of convenient commands. See the documentation [here](https://github.com/TouchBistro/tb/blob/master/docs/tb.md) for the command documentation.

## Having trouble?

Check the [FAQ](https://github.com/TouchBistro/core-devtools/blob/master/FAQ.md) for common problems and solutions. (Pull requests welcome!)

## Configuration

## FAQ

## Gotchas / Tips

- Do not run npm run or npm run commands from the host unless you absolutely need to.

- **Previous Setup**: If you already have `postgres.app` or are running postgres with homebrew or any other way, Datagrip-like tools will be confused about which pg to connect to. You won't need these anymore, so you can just delete them. Use `pgrep postgres` and make sure you don't have any other instances running.

- **SQL EDITORS**: To use external db tools like datagrip or `psql`, keep `CORE_DB_HOST` in the .env file as it is, but use `localhost` as the hostname in datagrip (or tool of choice). see `bin/db` for an example that uses `pgcli` on the host. Inside the docker network, containers uses the service names in `docker-compose.yml` as their hostname. Externally, their hostname is just `localhost`.

- **Slowness**: If running things in Docker on a mac is slow, allocate more CPUs, Memory and Swap space using the Docker For Mac advanced preferences. Keep in mind that some tools (like `jest`) have threading issues on linux and are not going to be faster with more cores. Use `docker stats` to see resource usage by image.
