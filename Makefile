BINARY  := sunbeams
PKG     := ./cmd/sunbeams
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the sunbeams binary for the host platform
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

.PHONY: build-linux
build-linux: ## Cross-compile static Linux binaries for amd64 and arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64 $(PKG)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-arm64 $(PKG)

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: test-race
test-race: ## Run tests with the race detector
	go test -race ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	go test -v ./...

.PHONY: cover
cover: ## Produce and open an HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Run gofmt -w on the tree
	gofmt -w .

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: generate-edid
generate-edid: build ## Regenerate virtual_display.bin and helper scripts
	./$(BINARY) generate

.PHONY: verify-golden
verify-golden: build ## Regenerate EDID and diff against the golden reference
	@tmp=$$(mktemp -d); \
	./$(BINARY) generate --output-dir "$$tmp" >/dev/null; \
	diff "$$tmp/virtual_display.bin" testdata/virtual_display_reference.bin \
		&& echo "golden parity OK" \
		|| (echo "GOLDEN DIFF — $$tmp/virtual_display.bin does not match testdata/virtual_display_reference.bin"; exit 1)

.PHONY: snapshot
snapshot: ## Build a goreleaser snapshot into dist/ (no publish)
	goreleaser release --snapshot --clean --skip=publish

.PHONY: check
check: fmt lint test verify-golden ## Run fmt, lint, tests, and golden-file verification

.PHONY: clean
clean: ## Remove build artifacts and generated files
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-linux-arm64
	rm -f virtual_display.bin add_custom_modes.sh sunshine_commands.txt
	rm -f coverage.out
	rm -rf dist/
