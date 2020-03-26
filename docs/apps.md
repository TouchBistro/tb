# `tb app`

`tb` can run iOS and desktop apps configured in registries. You can learn more about registries [here](registries.md).

Please note currently only macOS apps are supported. If you wish to add support for another platform, consider opening a PR.

## Requirements

For running iOS apps make sure you have Xcode installed and have run it at least once. This will ensure the iOS simulators are available on your computer.

`tb app` works by downloading apps from a storage services such as `S3`. Make sure you have access to the necessary storage service before using these commands.

Supported storage services are:
* s3

If your preferred storage provider is not listed, feel free to open a PR to add support for it.

### Storage service directory structure

`tb app` assumes a certain directory structure in your storage service. Each version of the app should be in a path matching a branch on GitHub. This allows users to run any version of an app by specifying a GitHub branch name.

`tb app` also expects apps to have their name prefixed with the Git SHA of the commit. This allows `tb app` to know if a cached build on the user's machine is out of date.

Ex:
If your bucket is called `app-builds` then the structure should look something like this:

```
app-builds/
  my-awesome-app/
    master/
      da39a3ee5e6b4b0d3255bfef95601890afd80709.my-awesome-app.app
    feat/
      add-cool-button/
        356a192b7913b04c54574d18c28d46e6395428ab.my-awesome-app.app
```

## Usage

### Listing available apps

You can see all available iOS and desktop apps by running `tb app list`.

### Running iOS Apps

To run an iOS app use the command `tb app ios run <app>`.

To change the branch use the `-b` or `--branch` flag.

Ex:
```sh
tb app ios run my-awesome-app -b task/my-branch
```

You can change the simulator type by using the `-d` or `--device` flag.

Ex:
```sh
tb app ios run my-awesome-app -d "iPad (5th generation)"
```

You can also change the iOS version by using the `-i` or `--ios-version` flag.

Ex:
```sh
tb app ios run my-awesome-app -i 12.4
```

**NOTE:** Run `xcrun simctl list devices available` to see the list of available simulators and their corresponding iOS version.

### Logs

To view the logs from the simulator run `tb app ios logs`.

You can change the number of lines initially displayed by using the `-n` or `--number` flag. The default number of lines is 10.

Ex:
```sh
tb ios logs -n 20
```

You can also choose the simulator and iOS version the same way as with `tb app ios run`.

### Running Desktop Apps

To run a desktop app use the command `tb app desktop run <app>`.

To change the branch use the `-b` or `--branch` flag.

Ex:
```sh
tb app desktop run my-awesome-app -b task/my-branch
```
