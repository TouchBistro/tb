.DEFAULT_GOAL = build

# Get all dependencies
setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	go mod download
.PHONY: setup

# Build tb
build:
	go build
	go run build/build.go
.PHONY: build

# Clean all build artifacts
clean:
	rm -rf dist
	rm -rf coverage
	rm -f tb
.PHONY: clean

# Run the linter
lint:
	./bin/golangci-lint run ./...
.PHONY: lint

# Remove version of tb installed with go install
go-uninstall:
	rm $(shell go env GOPATH)/bin/tb
.PHONY: go-uninstall

# Run tests and collect coverage data
test:
	mkdir -p coverage
	go test -coverprofile=coverage/coverage.txt ./...
	go tool cover -html=coverage/coverage.txt -o coverage/coverage.html
.PHONY: test

# Run tests and print coverage data to stdout
test-ci:
	mkdir -p coverage
	go test -coverprofile=coverage/coverage.txt ./...
	go tool cover -func=coverage/coverage.txt
.PHONY: test-ci
