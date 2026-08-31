#!/usr/bin/env bash
# Sync this tree onto a go-legacy-win7 tag while preserving XP sources and fork identity.
# Usage: scaffold-version.sh TAG [--work-dir DIR] [--skip-build] [--skip-pe] [--force]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
XP_LIST="$ROOT/scripts/xp-files.list"
FORK_LIST="$ROOT/scripts/fork-files.list"
UPSTREAM_URL="https://github.com/thongtech/go-legacy-win7.git"
DEFAULT_WORK="/tmp/go-legacy-win7"

TAG=""
WORK_DIR="$DEFAULT_WORK"
SKIP_BUILD=0
SKIP_PE=0
FORCE=0

usage() {
  cat >&2 <<EOF
usage: $0 TAG [--work-dir DIR] [--skip-build] [--skip-pe] [--force]

  TAG           win7 tag such as v1.27.0-2
  --work-dir    checkout directory for win7 (default: $DEFAULT_WORK)
  --skip-build  skip src/make.bash
  --skip-pe     skip scripts/check-xp-pe.sh
  --force       allow dirty worktree
EOF
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --work-dir)
      [[ $# -ge 2 ]] || usage
      WORK_DIR="$2"
      shift 2
      ;;
    --skip-build)
      SKIP_BUILD=1
      shift
      ;;
    --skip-pe)
      SKIP_PE=1
      shift
      ;;
    --force)
      FORCE=1
      shift
      ;;
    -h|--help)
      usage
      ;;
    -*)
      echo "unknown flag: $1" >&2
      usage
      ;;
    *)
      if [[ -n "$TAG" ]]; then
        echo "unexpected argument: $1" >&2
        usage
      fi
      TAG="$1"
      shift
      ;;
  esac
done

[[ -n "$TAG" ]] || usage

read_list() {
  local f="$1"
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    printf '%s\n' "$line"
  done <"$f"
}

require_lists() {
  [[ -f "$XP_LIST" ]] || { echo "missing $XP_LIST" >&2; exit 1; }
  [[ -f "$FORK_LIST" ]] || { echo "missing $FORK_LIST" >&2; exit 1; }
}

ensure_upstream() {
  if ! git -C "$ROOT" remote get-url upstream >/dev/null 2>&1; then
    echo "adding upstream remote $UPSTREAM_URL"
    git -C "$ROOT" remote add upstream "$UPSTREAM_URL"
  fi
}

check_clean() {
  if [[ "$FORCE" -eq 1 ]]; then
    return 0
  fi
  if [[ -n "$(git -C "$ROOT" status --porcelain)" ]]; then
    echo "working tree is dirty (commit/stash or pass --force)" >&2
    git -C "$ROOT" status --short >&2
    exit 1
  fi
}

checkout_win7() {
  local dir="$1"
  local tag="$2"
  if [[ -d "$dir/.git" ]]; then
    echo "updating existing checkout at $dir"
    git -C "$dir" fetch --tags origin
    git -C "$dir" checkout --force "$tag"
    git -C "$dir" reset --hard "$tag"
  else
    echo "cloning $UPSTREAM_URL into $dir"
    rm -rf "$dir"
    git clone "$UPSTREAM_URL" "$dir"
    git -C "$dir" checkout "$tag"
  fi
  echo "win7 VERSION:"
  cat "$dir/VERSION"
}

preserve_paths() {
  local dest="$1"
  mkdir -p "$dest"

  mapfile -t xp_files < <(read_list "$XP_LIST")
  for rel in "${xp_files[@]}"; do
    if [[ ! -e "$ROOT/$rel" ]]; then
      echo "missing XP file to preserve: $rel" >&2
      exit 1
    fi
    mkdir -p "$dest/$(dirname "$rel")"
    cp -a "$ROOT/$rel" "$dest/$rel"
  done

  mapfile -t fork_paths < <(read_list "$FORK_LIST")
  for rel in "${fork_paths[@]}"; do
    # Strip trailing slash for existence check
    local src="${rel%/}"
    if [[ ! -e "$ROOT/$src" ]]; then
      echo "warning: fork path missing (skipping): $rel" >&2
      continue
    fi
    mkdir -p "$dest/$(dirname "$src")"
    if [[ -d "$ROOT/$src" ]]; then
      mkdir -p "$dest/$src"
      # Preserve directory contents under dest/$src
      rsync -a "$ROOT/$src/" "$dest/$src/"
    else
      cp -a "$ROOT/$src" "$dest/$src"
    fi
  done
}

rsync_win7() {
  local src="$1"
  echo "rsync win7 tree into $ROOT"
  rsync -a --delete \
    --exclude='.git' \
    --exclude='bin' \
    --exclude='pkg' \
    --exclude='AGENTS.md' \
    --exclude='skills' \
    --exclude='scripts' \
    --exclude='docker' \
    --exclude='testdata' \
    "$src/" "$ROOT/"
}

restore_paths() {
  local src="$1"
  echo "restoring XP sources and fork identity"
  rsync -a "$src/" "$ROOT/"
}

sanity_checks() {
  local win7="$1"
  if ! grep -q 'go-legacy-winxp' "$ROOT/README.md"; then
    echo "fork branding missing from README.md" >&2
    exit 1
  fi
  if ! grep -q 'branch=master' "$ROOT/.github/workflows/go-build.yml"; then
    echo "go-build.yml missing master fallback" >&2
    exit 1
  fi
  if ! diff -q "$win7/src/runtime/os_windows.go" "$ROOT/src/runtime/os_windows.go" >/dev/null 2>&1; then
    :
  else
    echo "warning: os_windows.go matches win7 (XP restore may have failed)" >&2
  fi
}

require_lists
check_clean
ensure_upstream

preserve_dir="$(mktemp -d "${TMPDIR:-/tmp}/winxp-preserve.XXXXXX")"
cleanup() {
  rm -rf "$preserve_dir"
}
trap cleanup EXIT

echo "preserving XP and fork files"
preserve_paths "$preserve_dir"

checkout_win7 "$WORK_DIR" "$TAG"
rsync_win7 "$WORK_DIR"
restore_paths "$preserve_dir"

echo "regenerating patches/0010"
"$ROOT/scripts/regenerate-xp-patch.sh" "$WORK_DIR"

sanity_checks "$WORK_DIR"

echo
echo "VERSION after sync:"
cat "$ROOT/VERSION"

if [[ "$SKIP_BUILD" -eq 0 ]]; then
  echo "building toolchain (src/make.bash)"
  (cd "$ROOT/src" && ./make.bash)
  "$ROOT/bin/go" version
else
  echo "skipped build (--skip-build)"
fi

if [[ "$SKIP_PE" -eq 0 ]]; then
  if [[ ! -x "$ROOT/bin/go" ]]; then
    echo "no ./bin/go for PE check (build first or pass --skip-pe)" >&2
    exit 1
  fi
  "$ROOT/scripts/check-xp-pe.sh"
else
  echo "skipped PE check (--skip-pe)"
fi

cat <<EOF

Scaffold complete for $TAG.

Next steps:
  - review git status / diff
  - optional: ./scripts/test-xp-docker.sh
  - commit when ready (do not commit bin/ pkg/ docker/xp/storage)
  - release via workflow_dispatch on go-build.yml
EOF
