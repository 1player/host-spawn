#!/bin/bash

set -euo pipefail

ARCH="
    386-i386
    386-i686
    amd64-x86_64
    arm-armv7
    arm64-aarch64
"

ROOT_DIR=$(dirname "$0")
VERSION=${VERSION:-HEAD}

cd "$ROOT_DIR"

mkdir -p build

echo "$VERSION"

for architecture in ${ARCH}; do
	CGO_ENABLED=0 GOARCH="$(echo "${architecture}" | cut -d'-' -f1)" go build \
		-ldflags  "-X main.Version=$VERSION" \
		-o build/host-spawn-"$(echo "${architecture}" | cut -d'-' -f2)"
done

# Create source tarball including vendored dependencies
git clean -fdx -e build
go mod vendor
tar --create --zst --exclude build --file build/host-spawn-vendor.tar.zst "$ROOT_DIR"
