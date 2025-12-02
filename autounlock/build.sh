#!/usr/bin/env bash
set -euo pipefail

# Build static linux/amd64 binary
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o autounlock

mkdir -p ../src/usr/local/php/unraid-auto-unlock/bin
cp autounlock ../src/usr/local/php/unraid-auto-unlock/bin/