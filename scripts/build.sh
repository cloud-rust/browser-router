#!/usr/bin/env bash
# Test and build the router for all targets in a Go container, then the macOS app bundle locally.
set -euo pipefail
cd "$(dirname "$0")/.."
GO_IMAGE=golang:1.23.4
DOCKER=(docker ${DOCKER_CONTEXT:+--context "$DOCKER_CONTEXT"})

"${DOCKER[@]}" run --rm -v "$PWD":/src -w /src -e CGO_ENABLED=0 "$GO_IMAGE" sh -euc '
  go vet ./...
  go test ./...
  for t in darwin/arm64 linux/arm64 linux/amd64 windows/amd64; do
    os=${t%/*}; arch=${t#*/}; out=build/$os-$arch/router; flags=
    [ "$os" = windows ] && { out=$out.exe; flags=-H=windowsgui; }
    GOOS=$os GOARCH=$arch go build -trimpath -ldflags "-s -w $flags" -o "$out" .
  done'

if [ "$(uname)" = Darwin ]; then
  app=build/BrowserRouterPoC.app
  rm -rf "$app"; mkdir -p "$app/Contents/MacOS"
  cp mac/Info.plist "$app/Contents/"
  swiftc -O mac/main.swift -o "$app/Contents/MacOS/BrowserRouterPoC"
  cp build/darwin-arm64/router "$app/Contents/MacOS/router"
  codesign --force --deep -s - "$app"
fi
ls -l build/*/router* 2>/dev/null
