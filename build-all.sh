#!/bin/bash

set -euo pipefail

ARCHS="i386 i686 x86_64 armv7 aarch64 loongarch64" # in `uname -m` format

ROOT_DIR=$(dirname "$0")

cd "$ROOT_DIR"
for arch in ${ARCHS}; do
    ./build.sh "$arch"
done

./build.sh source
