#@IgnoreInspection BashAddShebang
ROOT=$(realpath $(dir $(lastword $(MAKEFILE_LIST))))
CGO_ENABLED?=0

# Install by `go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@<SET VERSION>`
GOLANGCI_LINT_CMD=go tool golangci-lint

# Install by `go get -tool github.com/4meepo/tagalign/cmd/tagalign@<SET VERSION>`
TAGALIGN_CMD=go tool tagalign

.DEFAULT_GOAL := .default

.default: format build lint test

.PHONY: help
help: ## Show help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.which-go:
	@which go > /dev/null || (echo "Install Go from https://go.dev/doc/install" & exit 1)

.PHONY: build
build: .which-go ## Build binary
	CGO_ENABLED=1 go build -v -o $(ROOT)/bin/api $(ROOT)/cmd/api

.PHONY: format
format: .which-go ## Format files
	go mod tidy
	gofmt -s -w $(ROOT)
	$(TAGALIGN_CMD) -fix $(ROOT)/... || echo "tags aligned"

.PHONY: lint
lint: .which-go ## Check lint
	$(GOLANGCI_LINT_CMD) run

.PHONY: test
test: .which-go ## Run tests
	CGO_ENABLED=1 go test -race -cover -coverprofile=coverage.out -covermode=atomic $(ROOT)/...
