# Services

`tb` main purpose from it's inception has been to make it easy to run lots of services. In a microservices world it can be hard to remember all the services you need to run to be able to do your work. `tb` takes care of this by easily running as many services as you need with one command.

## `tb up`

This command is where the magic happens. `tb up` takes either a list of services or a playlist and runs all the services in docker containers.

Ex:
```
tb up -s postgres,venue-core-service
```

Or
```
tb up -p service-deps
```

`tb up` will automatically take care of:
* Pulling the latest versions of any git repos for services
* Pulling the latest docker images for services
* Building docker images for services
* Running configured pre run commands for services (ex: running database migrations)

Once it is finished `tb up` will start [lazydocker](https://github.com/jesseduffield/lazydocker) which provides an easy way to manage and see all the running docker containers.
`tb up` runs containers in the background so you can safely exit lazydocker and the containers will continue running.

## `tb down`

As mentioned above, `tb up` runs containers in the background. `tb down` can be used to stop and remove these running containers.

You can pass a comma separated list of service names if you only wish to stop certain services:
```
tb down postgres,venue-core-service
```

If no services are passed to `tb down`, it will stop all running services.
```
tb down
```

## `tb exec`

`tb exec` can be used to execute a shell command in a running service's container.

Ex:
```
tb exec venue-core-service echo hello world
```

A common use case is needing to open a shell in a container. This can easily be done by running:
```
tb exec venue-core-service bash
```

## `tb logs`

`tb logs` can be used to view the logs for one or more services.

Ex:
```
tb logs postgres,venue-core-service
```

## `tb list`

`tb list` lists all available services, playlists, and custom playlists.

Ex:
```
tb list
```

Flags can be used to only show services, playlists or custom playlists.

Ex: Show only services
```
tb list -s
```

`tb list` also offers the `-t` or `--tree` flag which will show the services in each playlist.

Ex: Show all playlists and their services
```
tb list -s -t
```
