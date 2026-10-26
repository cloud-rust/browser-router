#!/usr/bin/env bash
# Run the Linux end-to-end test in a container; needs the binaries from scripts/build.sh.
set -euo pipefail
cd "$(dirname "$0")/.."
DOCKER=(docker ${DOCKER_CONTEXT:+--context "$DOCKER_CONTEXT"})
"${DOCKER[@]}" build -q -t browser-router-e2e scripts/e2e >/dev/null
"${DOCKER[@]}" run --rm --init -v "$PWD":/src:ro browser-router-e2e \
  dbus-run-session -- bash /src/scripts/e2e-linux.sh
