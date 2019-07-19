.DEFAULT_GOAL = build
TB_ROOT = $(HOME)/.tb

# Get all dependencies
setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	go get github.com/gobuffalo/packr/v2/packr2
	go mod download
.PHONY: setup

# Build tb
build:
	packr2
	go build
	go run build/build.go
.PHONY: build

# Clean all build artifacts
clean:
	packr2 clean
	rm -rf dist
.PHONY: clean

# Run the linter
lint:
	./bin/golangci-lint run ./...
.PHONY: lint

# Remove version of tb installed with go install
go-uninstall:
	rm $(shell go env GOPATH)/bin/tb
.PHONY: go-uninstall

rm-files:
	rm -f $(TB_ROOT)/docker-compose.yml
	rm -f $(TB_ROOT)/localstack-entrypoint.sh
.PHONY: rm-files
