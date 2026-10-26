#!/usr/bin/env bash
# Assemble the macOS app bundle around a router binary and sign it ad hoc.
# usage: mac-app.sh <router binary> <app path> <version>
set -euo pipefail
bin=$1 app=$2 version=$3
here=$(cd "$(dirname "$0")/.." && pwd)
rm -rf "$app"; mkdir -p "$app/Contents/MacOS"
sed "s/VERSION/${version#v}/g" "$here/mac/Info.plist" > "$app/Contents/Info.plist"
swiftc -O "$here/mac/main.swift" -o "$app/Contents/MacOS/BrowserRouter"
cp "$bin" "$app/Contents/MacOS/browser-router"
codesign --force --deep -s - "$app"
