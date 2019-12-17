# `tb ios`

## Requirements

Make sure you have Xcode installed and have run it at least once. This will ensure the iOS simulators are available on your computer.

An AWS account is required to use `tb ios`. Follow the instructions in the [README](../README.md#aws-ecr)

## Usage

### Running iOS Apps
To run the POS run `tb ios run`. This will download the POS from S3 and run it in the simulator.

The default branch for the POS is `develop`. To change the branch use the `-b` or `--branch` flag.

Ex:
```sh
tb ios run -b task/my-branch
```

You can change the simulator type by using the `-d` or `--device` flag.

Ex:
```sh
tb ios run -d "iPad (5th generation)"
```

You can also change the iOS version by using the `-i` or `--ios-version` flag.

Ex:
```sh
tb ios run -i 12.4
```

**NOTE:** Run `xcrun simctl list devices available` to see the list of available simulators and their corresponding iOS version.

### Logs

To view the logs from the simulator run `tb ios logs`.

You can change the number of lines initially displayed by using the `-n` or `--number` flag. The default number of lines is 10.

Ex:
```sh
tb ios logs -n 20
```

You can also choose the simulator and iOS version the same way as with `tb ios run`.

## Gotchas / Tips

- **Branch does not exist**: Make sure the branch you are trying to run has an open PR in the `tb-pos` repo. This is because CI only runs on PRs, and CI is where the build gets uploaded to S3.
