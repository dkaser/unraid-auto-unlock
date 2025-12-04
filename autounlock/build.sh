#!/usr/bin/env bash
set -euo pipefail

GIT_TAG=$(git describe --tags --always 2>/dev/null || echo "unknown")

# Build static linux/amd64 binary
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/dkaser/unraid-auto-unlock/autounlock/version.Tag=${GIT_TAG}" -o autounlock

mkdir -p ../src/usr/local/php/unraid-auto-unlock/bin
cp autounlock ../src/usr/local/php/unraid-auto-unlock/bin/