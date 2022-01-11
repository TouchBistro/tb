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
