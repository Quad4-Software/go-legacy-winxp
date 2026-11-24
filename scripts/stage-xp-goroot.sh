#!/usr/bin/env bash
# Assemble a windows/386 GOROOT for the XP guest under docker/xp/shared/go.
# The guest reaches it as \\host.lan\Data\go (mapped as Z: or the share root).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SHARE="${XP_SHARE_DIR:-$ROOT/docker/xp/shared}"
DEST="$SHARE/go"
TARGET="windows_386"

if [[ ! -x "$ROOT/bin/$TARGET/go.exe" ]]; then
  echo "missing $ROOT/bin/$TARGET/go.exe" >&2
  echo "build it with: cd src && GOOS=windows GOARCH=386 ./make.bash" >&2
  exit 1
fi
if [[ ! -d "$ROOT/pkg/tool/$TARGET" ]]; then
  echo "missing $ROOT/pkg/tool/$TARGET" >&2
  exit 1
fi

echo "staging GOROOT into $DEST"
mkdir -p "$DEST/bin" "$DEST/pkg/tool" "$DEST/pkg/include"

install -m 755 "$ROOT/bin/$TARGET/go.exe" "$DEST/bin/go.exe"
install -m 755 "$ROOT/bin/$TARGET/gofmt.exe" "$DEST/bin/gofmt.exe" 2>/dev/null || true
cp -a "$ROOT/pkg/tool/$TARGET" "$DEST/pkg/tool/"
cp -a "$ROOT/pkg/include/." "$DEST/pkg/include/" 2>/dev/null || true

# Sources: everything the compiler may need for std builds, minus cmd/
# (go tool binaries live in pkg/tool) and test files.
mkdir -p "$DEST/src"
rsync -a --delete \
  --exclude 'cmd/' \
  --exclude '*_test.go' \
  --exclude 'testdata/' \
  --exclude '.git' \
  "$ROOT/src/" "$DEST/src/"

# A couple of files go looks for at the root.
cp "$ROOT/VERSION" "$DEST/VERSION" 2>/dev/null || true

echo "staged $(du -sh "$DEST" | cut -f1) at $DEST"
echo "in the guest use: set GOROOT=\\host.lan\Data\go or Z:\go"
