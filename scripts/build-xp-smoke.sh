#!/usr/bin/env bash
# Cross-compile the XP smoke integration binary for the Docker guest.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GO="${GO:-$ROOT/bin/go}"
OUTDIR="${1:-$ROOT/docker/xp/oem}"

if [[ ! -x "$GO" ]]; then
  echo "missing go binary at $GO (build with cd src && ./make.bash first)" >&2
  exit 1
fi

export CGO_ENABLED=0
mkdir -p "$OUTDIR"

echo "building windows/386 XP integration smoke test"
GOOS=windows GOARCH=386 "$GO" build -C "$ROOT/testdata/xp-smoke" -o "$OUTDIR/xp-smoke-386.exe" .

echo "built $OUTDIR/xp-smoke-386.exe"
echo "run on XP via ./scripts/test-xp-docker.sh or copy to docker/xp/shared/"
