# Contributing

The following document outlines how to contribute to the `tb` project.

### **Table of Contents**
- [Requirements](#requirements)
- [Setup](#setup)
- [Building](#building)
    + [Running locally](#running-locally)
    + [Running globally](#running-globally)
    + [Remove global build](#remove-global-build)
- [Linting](#linting)
- [Testing](#testing)
- [Cleaning Up](#cleaning-up)

## Requirements

To build and run `tb` locally you will need to install go.
This can likely be done through your package manager or by going to https://golang.org/doc/install.

For example, on macOS you can use homebrew by running:
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

## Linting
To run the linter run:
```sh
make lint
```

## Testing
To run the tests run:
```sh
make test
```

This will output code coverage information to the `coverage` directory.
You can open the `coverage/coverage.html` file in your browser to visually see the coverage in each file.

## Cleaning Up
To remove the build and another other generated files run:
```sh
make clean
```

Additionally if `go.sum` gets modified on build, please run `go mod tidy` to clean it up before committing and pushing changes.
