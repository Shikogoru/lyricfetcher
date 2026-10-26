VERSION ?= 1.0.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE ?= $(shell date -u +"%Y-%m-%d" 2>/dev/null || echo "2026-10-25")
LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

TARGETS = \
	windows/amd64/.exe \
	windows/arm64/.exe \
	linux/amd64/ \
	linux/arm64/ \
	darwin/amd64/ \
	darwin/arm64/

.PHONY: all build build-cli build-gui build-all test clean release help

all: build

help:
	@echo "LyricFetcher Makefile"
	@echo "Targets:"
	@echo "  build       Build CLI and GUI binaries in bin/"
	@echo "  build-cli   Build CLI binary in bin/"
	@echo "  build-gui   Build GUI binary in bin/ (requires CGO)"
	@echo "  build-all   Cross-compile 6 target platforms in dist/ (CLI)"
	@echo "  test        Run unit tests with -race"
	@echo "  clean       Remove bin/ and dist/"
	@echo "  release     Test, cross-compile, and generate checksums"

build: build-cli build-gui

build-cli:
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o bin/lyricfetcher ./cmd/cli

build-gui:
	@mkdir -p bin
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/lyricfetcher-gui ./cmd/gui

test:
	go test -race ./internal/...

clean:
	rm -rf bin dist

build-all: clean
	@mkdir -p dist
	@for target in $(TARGETS); do \
		OS=$$(echo $$target | cut -d'/' -f1); \
		ARCH=$$(echo $$target | cut -d'/' -f2); \
		EXT=$$(echo $$target | cut -d'/' -f3); \
		OUT="dist/lyricfetcher-$(VERSION)-$$OS-$$ARCH$$EXT"; \
		echo "Building $$OUT..."; \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build -ldflags="$(LDFLAGS)" -o $$OUT ./cmd/cli || exit 1; \
		if command -v upx >/dev/null 2>&1 && [ "$$OS" != "darwin" ]; then \
			upx --best --quiet $$OUT 2>/dev/null || true; \
		fi; \
	done
	@echo "All binaries built in dist/"

release: test build-all
	@echo "Generating SHA256 checksums..."
	@cd dist && (shasum -a 256 lyricfetcher-* > checksums.txt 2>/dev/null || sha256sum lyricfetcher-* > checksums.txt 2>/dev/null)
	@echo "Release prepared in dist/"
