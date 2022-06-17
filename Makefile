.DEFAULT_GOAL = build
TB = go run main.go

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

setup: ## Get all dependencies
# Only install if missing
ifeq (,$(wildcard bin/golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.46.2
endif

	go mod tidy
.PHONY: setup

build: ## Build tb
	go build
.PHONY: build

artifacts: ## Generate artifacts for distribution
	@mkdir -p artifacts
	@$(TB) completion bash > artifacts/tb.bash
	@$(TB) completion zsh > artifacts/_tb
.PHONY: artifacts

clean: ## Clean all build artifacts
	rm -rf artifacts
	rm -rf coverage
	rm -rf dist
	rm -f tb
.PHONY: clean

lint: ## Run the linter
	./bin/golangci-lint run ./...
.PHONY: lint

go-uninstall: ## Remove version of tb installed with go install
	rm $(shell go env GOPATH)/bin/tb
.PHONY: go-uninstall

test: ## Run tests and collect coverage data
	mkdir -p coverage
	go test -coverprofile=coverage/coverage.txt ./...
	go tool cover -html=coverage/coverage.txt -o coverage/coverage.html
.PHONY: test

test-ci: ## Run tests and print coverage data to stdout
	mkdir -p coverage
	go test -coverprofile=coverage/coverage.txt ./...
	go tool cover -func=coverage/coverage.txt
.PHONY: test-ci
