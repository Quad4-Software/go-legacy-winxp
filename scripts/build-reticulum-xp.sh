#!/usr/bin/env bash
# Cross-compile Reticulum-Go for Windows XP (windows/386) using this tree's go.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GO="${GO:-$ROOT/bin/go}"
OUTDIR="${1:-$ROOT/docker/xp/oem}"
SRC="${RETICULUM_GO_SRC:-/tmp/Reticulum-Go}"
REPO_URL="${RETICULUM_GO_URL:-https://github.com/Quad4-Software/Reticulum-Go.git}"

if [[ ! -x "$GO" ]]; then
  echo "missing go binary at $GO (build with cd src && ./make.bash first)" >&2
  exit 1
fi

if [[ ! -d "$SRC/.git" && ! -f "$SRC/go.mod" ]]; then
  echo "cloning $REPO_URL into $SRC"
  git clone --depth 1 "$REPO_URL" "$SRC"
fi

mkdir -p "$OUTDIR"
SHARED="$ROOT/docker/xp/shared"
mkdir -p "$SHARED"

export CGO_ENABLED=0
export GOFLAGS="${GOFLAGS:--mod=vendor}"
export GOTOOLCHAIN=local

echo "building reticulum-go windows/386 with $GO"
GOOS=windows GOARCH=386 "$GO" build -C "$SRC" -ldflags="-s -w" -o "$OUTDIR/reticulum-go-winxp.exe" ./cmd/reticulum-go
cp -f "$OUTDIR/reticulum-go-winxp.exe" "$SHARED/reticulum-go-winxp.exe"
echo "built $OUTDIR/reticulum-go-winxp.exe"
