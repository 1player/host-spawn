#!/bin/bash

set -euo pipefail

ARCH="$1" # in `uname -m` format
ROOT_DIR=$(dirname "$0")
VERSION=${VERSION:-HEAD}

cd "$ROOT_DIR"
mkdir -p build

case $ARCH in
    source)
        git clean -fdx -e build
        go mod vendor
        tar --create --zst --exclude build --file build/host-spawn-vendor.tar.zst "$ROOT_DIR"
        exit
        ;;

    i386 | i686)
        GOARCH=386
        ;;

    x86_64)
        GOARCH=amd64
        ;;

    armv7)
        GOARCH=arm
        ;;

    aarch64)
        GOARCH=arm64
        ;;
    loongarch64)
        GOARCH=loong64
	;;
    *)
        GOARCH=$ARCH
        ;;
esac

export GOARCH
CGO_ENABLED=0 go build \
		      -ldflags  "-X main.Version=$VERSION" \
		      -o "build/host-spawn-$ARCH"
