#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-build}"
VERSION="${2:-1.0.0}"

COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE=$(date -u +"%Y-%m-%d")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

clean_dirs() {
    echo "Cleaning bin/ and dist/..."
    rm -rf bin dist
}

run_tests() {
    echo "Running tests..."
    go test -race ./internal/...
}

build_cli() {
    echo "Building LyricFetcher CLI (Version: ${VERSION}, Commit: ${COMMIT})..."
    mkdir -p bin
    local out="bin/lyricfetcher"
    if [[ "${OSTYPE:-}" == "msys"* || "${OSTYPE:-}" == "cygwin"* ]]; then
        out="bin/lyricfetcher.exe"
    fi
    go build -ldflags="${LDFLAGS}" -o "${out}" ./cmd/cli
    echo "Built successfully -> ${out}"
}

build_gui() {
    echo "Building LyricFetcher GUI (Version: ${VERSION}, Commit: ${COMMIT})..."
    mkdir -p bin
    local out="bin/lyricfetcher-gui"
    if [[ "${OSTYPE:-}" == "msys"* || "${OSTYPE:-}" == "cygwin"* ]]; then
        out="bin/lyricfetcher-gui.exe"
    fi
    CGO_ENABLED=1 go build -ldflags="${LDFLAGS}" -o "${out}" ./cmd/gui
    echo "Built successfully -> ${out}"
}

build_current() {
    build_cli
    if command -v gcc >/dev/null 2>&1 || command -v clang >/dev/null 2>&1; then
        build_gui
    else
        echo "C compiler not found, skipping GUI build."
    fi
}

build_all() {
    echo "Building CLI for all 6 target platforms..."
    mkdir -p dist

    local targets=(
        "windows amd64 .exe"
        "windows arm64 .exe"
        "linux amd64 "
        "linux arm64 "
        "darwin amd64 "
        "darwin arm64 "
    )

    for item in "${targets[@]}"; do
        read -r os arch ext <<< "${item}"
        local name="lyricfetcher-${VERSION}-${os}-${arch}${ext}"
        local out="dist/${name}"
        echo "  -> Building ${out} (GOOS=${os} GOARCH=${arch})..."
        CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build -ldflags="${LDFLAGS}" -o "${out}" ./cmd/cli

        if command -v upx >/dev/null 2>&1 && [[ "${os}" == "windows" || "${os}" == "linux" ]]; then
            echo "     [UPX] Compressing ${out}..."
            upx --best --quiet "${out}" 2>/dev/null || true
        fi
    done
    echo "All targets built successfully in dist/."
}

generate_checksums() {
    echo "Generating SHA256 checksums in dist/checksums.txt..."
    (
        cd dist
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum lyricfetcher-* > checksums.txt
        elif command -v shasum >/dev/null 2>&1; then
            shasum -a 256 lyricfetcher-* > checksums.txt
        fi
    )
    echo "Checksums saved to dist/checksums.txt"
}

case "${TARGET}" in
    clean)
        clean_dirs
        ;;
    test)
        run_tests
        ;;
    build-cli)
        build_cli
        ;;
    build-gui)
        build_gui
        ;;
    build)
        build_current
        ;;
    build-all)
        clean_dirs
        build_all
        ;;
    release)
        clean_dirs
        run_tests
        build_all
        generate_checksums
        ;;
    *)
        echo "Unknown target: ${TARGET}. Usage: ./build.sh [build|build-cli|build-gui|build-all|test|clean|release] [version]"
        exit 1
        ;;
esac
