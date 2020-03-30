# Commands

`tb` has 3 main groups of commands for different sets of functionality.
1. `tb registry` is used for working with registries. Learn more by reading the [registry docs](registries.md).
2. `tb app` is used for running and managing iOS and desktop apps. Learn more by reading the [app docs](apps.md).
3. All remaining commands are using for working with services. Learn more about running services by reading the [services docs](services.md).

# Utility commands

`tb` has some utility commands that can generally make your life easier.

## `tb nuke`

`tb nuke` is used to clean up and remove any resources created by `tb`. This includes docker containers, docker images, files, etc. Nuke is very useful for fixing issues if `tb` ever gets into a weird state.

Nuke provides the following flags to clean up resources:
* `--containers`: Removes all docker containers
* `--images`:     Removes all docker images
* `--networks`:   Removes all docker networks
* `--volumes`:    Removes all docker volumes
* `--desktop`:    Removes all downloaded desktop app builds
* `--ios`:        Removes all downloaded iOS builds
* `--registries`: Removes all cloned registries
* `--repos`:      Removes all clone service git repos

Additionally the `--all` flag is also available which combines all the flags listed above and removes the `~/.tb` directory.

## `tb db`

`tb db` provides the ability to quickly connect to a database through the use of [dbcli tools](https://www.dbcli.com/)

The following database types are supported:
* `postgresql` through [`pgcli`](https://www.pgcli.com/)
* `mysql` through [`mycli`](https://www.mycli.net/)
* `mssql` through [`mssql-cli`](https://github.com/dbcli/mssql-cli)

`tb db` will prompt you to download the necessary CLI the first time it is used.

The have your service support `tb db` the following environment variables must be set inside your service's docker container:
* `DB_TYPE`: One of `postgresql`, `mysql`, or `mssql`
* `DB_NAME`: The name of the database to connect to
* `DB_PORT`: The port the database service is running on
* `DB_USER`: The database user name
* `DB_PASSWORD`: The database user password

To connect your service's database simply run:
```
tb db <service>
```

**NOTE:** `tb db` assumes the database service host is `localhost`.

## `tb clone`

`tb clone` provides a convenient way to clone any service that has a GitHub repo configured.

Ex:
```
tb clone venue-core-serivce
```

This would clone the git repo for `venue-core-service` into `./venue-core-service`.
