#!/usr/bin/env bash
# Test and build the router for all targets in a Go container (no local Go needed), then the
# macOS app bundle locally. CI does the same natively (.github/workflows).
set -euo pipefail
cd "$(dirname "$0")/.."
GO_IMAGE=golang:1.23.4
VERSION=${VERSION:-dev}
DOCKER=(docker ${DOCKER_CONTEXT:+--context "$DOCKER_CONTEXT"})

"${DOCKER[@]}" run --rm -v "$PWD":/src -w /src -e CGO_ENABLED=0 -e VERSION="$VERSION" "$GO_IMAGE" sh -euc '
  test -z "$(gofmt -l .)" || { gofmt -l .; exit 1; }
  go vet ./...
  go test ./...
  for t in darwin/arm64 linux/arm64 linux/amd64 windows/amd64; do
    os=${t%/*}; arch=${t#*/}; out=build/$os-$arch/browser-router; flags=
    [ "$os" = windows ] && { out=$out.exe; flags=-H=windowsgui; }
    GOOS=$os GOARCH=$arch go build -trimpath -ldflags "-s -w -X main.version=$VERSION $flags" -o "$out" .
  done'

if [ "$(uname)" = Darwin ]; then
  scripts/mac-app.sh build/darwin-arm64/browser-router "build/Browser Router.app" "$VERSION"
fi
ls -l build/*/browser-router* 2>/dev/null
