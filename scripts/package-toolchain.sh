#!/usr/bin/env bash
# Package a built GOROOT for one GOOS_GOARCH target.
set -euo pipefail

if [[ $# -lt 3 ]]; then
  echo "usage: $0 <goos_goarch> <version> <outdir>" >&2
  exit 1
fi

TARGET="$1"
VERSION="$2"
OUTDIR="$3"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

case "$TARGET" in
  windows_*|darwin_*|linux_*)
    ;;
  *)
    echo "unsupported target $TARGET" >&2
    exit 1
    ;;
esac

GOOS="${TARGET%%_*}"
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

NAME="go-legacy-winxp-${VERSION}.${TARGET}"
DEST="$STAGE/go-legacy-winxp"
mkdir -p "$DEST"

cp -a "$ROOT/." "$DEST/"
cd "$DEST"

rm -rf .git .github patches test doc misc skills agent-tools
rm -f .gitattributes .gitignore
find . -name '*.md' -type f -delete
find src -name '*_test.go' -delete
find src -type d -name testdata -prune -exec rm -rf {} +
rm -f src/*.bash src/*.bat src/*.rc src/make.* src/run.* 2>/dev/null || true

cd bin
if [[ "$TARGET" != "linux_amd64" ]]; then
  rm -f go gofmt
fi
if [[ -d "$TARGET" ]]; then
  mv "$TARGET"/* ./
fi
find . -mindepth 1 -maxdepth 1 -type d -exec rm -rf {} +

if [[ "$GOOS" == windows ]]; then
  if [[ ! -f go.exe ]]; then
    echo "missing go.exe for $TARGET" >&2
    exit 1
  fi
else
  if [[ ! -f go ]]; then
    echo "missing go binary for $TARGET" >&2
    exit 1
  fi
fi

cd ../pkg
find . -mindepth 1 -maxdepth 1 -type d ! \( -name include -o -name tool \) -exec rm -rf {} +
if [[ -d tool ]]; then
  cd tool
  find . -mindepth 1 -maxdepth 1 -type d ! -name "$TARGET" -exec rm -rf {} +
fi

mkdir -p "$OUTDIR"
cd "$STAGE"
if [[ "$GOOS" == windows ]]; then
  zip -r "$OUTDIR/${NAME}.zip" go-legacy-winxp
else
  tar -czf "$OUTDIR/${NAME}.tar.gz" go-legacy-winxp
fi

echo "wrote $OUTDIR/${NAME}.*"
