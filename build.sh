#!/bin/sh

ARCH="
    386-i386
    386-i686
    amd64-x86_64
    arm-armv7
    arm64-aarch64
"

mkdir -p build

for architecture in ${ARCH}; do
	CGO_ENABLED=0   GOARCH="$(echo "${architecture}" | cut -d'-' -f1)" go build \
		-o     build/host-spawn-"$(echo "${architecture}" | cut -d'-' -f2)"
done
