.PHONY: build install test clean fmt lint build-all release dist-clean

BINARY := zap
# Use TAG if provided (for releases), otherwise use git describe
VERSION := $(or $(TAG),$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev"))
BUILD_DATE := $(shell date -u +%Y-%m-%d)
LDFLAGS := -ldflags "-s -w -X github.com/itda-work/zap/internal/cli.Version=$(VERSION) -X github.com/itda-work/zap/internal/cli.BuildDate=$(BUILD_DATE)"
DIST := dist

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/zap/

install:
	go install $(LDFLAGS) ./cmd/zap/

test:
	go test -v ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f $(BINARY) coverage.out coverage.html

dist-clean:
	rm -rf $(DIST)

fmt:
	go fmt ./...

lint:
	golangci-lint run

# Cross-compilation (all platforms)
build-all: dist-clean
	@mkdir -p $(DIST)
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 ./cmd/zap/
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64 ./cmd/zap/
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-macos-amd64 ./cmd/zap/
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-macos-arm64 ./cmd/zap/
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/zap/
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-windows-arm64.exe ./cmd/zap/
	@echo "Creating checksums..."
	cd $(DIST) && sha256sum * > checksums.txt
	@echo "Done! Binaries in $(DIST)/"

# Create GitHub release (requires gh CLI)
release: build-all
	@if [ -z "$(TAG)" ]; then echo "Usage: make release TAG=v0.2.0"; exit 1; fi
	@git fetch --tags -q 2>/dev/null || true
	@if ! git rev-parse $(TAG) >/dev/null 2>&1; then \
		echo "Creating git tag $(TAG)..."; \
		git tag -a $(TAG) -m "Release $(TAG)"; \
		git push origin $(TAG); \
	fi
	@echo "Generating release notes..."
	@PREV_TAG=$$(git describe --tags --abbrev=0 $(TAG)^ 2>/dev/null || echo ""); \
	if [ -n "$$PREV_TAG" ]; then \
		./$(BINARY) release-notes "$$PREV_TAG" $(TAG) > $(DIST)/release-notes.md; \
	else \
		./$(BINARY) release-notes $(TAG) > $(DIST)/release-notes.md; \
	fi
	gh release create $(TAG) $(DIST)/* --title "$(TAG)" --notes-file $(DIST)/release-notes.md
