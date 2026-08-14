#!/usr/bin/env bash
# Build XP smoke binaries and run them inside dockur/windows VERSION=xp.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
XP_DIR="$ROOT/docker/xp"
SHARED="$XP_DIR/shared"
RESULT="$SHARED/result.txt"
COMPOSE=(docker compose -f "$XP_DIR/docker-compose.yml")

resolve_storage_dir() {
  local storage="$XP_DIR/storage"
  if [[ -n "${XP_STORAGE:-}" ]]; then
    echo "$XP_STORAGE"
    return
  fi
  if [[ -L "$storage" ]]; then
    storage="$(readlink -f "$storage")"
  fi
  echo "$storage"
}

setup_storage_dir() {
  local storage_fs alt candidates avail
  storage_fs="$(df -T "$XP_DIR" | awk 'NR==2 {print $2}')"
  candidates=()
  if [[ "$storage_fs" == "btrfs" ]]; then
    candidates+=(/var/tmp/go-legacy-winxp-xp-storage /tmp/go-legacy-winxp-xp-storage)
  else
    candidates+=("$XP_DIR/storage")
  fi
  for alt in "${candidates[@]}"; do
    avail="$(df -BG "$(dirname "$alt")" | awk 'NR==2 {gsub(/G/,"",$4); print $4}')"
    if [[ "${avail:-0}" -ge 12 ]]; then
      mkdir -p "$alt"
      export XP_STORAGE="$alt"
      echo "using $alt for XP disk (${avail}G free on $(dirname "$alt"))"
      return
    fi
  done
  alt="${candidates[0]}"
  mkdir -p "$alt"
  export XP_STORAGE="$alt"
  echo "warning: low disk space for XP storage, using $alt" >&2
}

has_xp_disk_image() {
  local storage="$1"
  compgen -G "$storage/data.*" >/dev/null \
    || [[ -f "$storage/windows.img" ]] \
    || [[ -f "$storage/windows.base" ]]
}

validate_xp_storage() {
  local storage
  storage="$(resolve_storage_dir)"
  mkdir -p "$storage"

  if has_xp_disk_image "$storage" && [[ ! -f "$storage/windows.boot" ]]; then
    echo "wiping stale XP storage at $storage (disk image without windows.boot)"
    find "$storage" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
  fi
}

if [[ -f "$(resolve_storage_dir)/windows.boot" ]]; then
  TIMEOUT_SECS="${XP_TEST_TIMEOUT:-1800}"
else
  TIMEOUT_SECS="${XP_TEST_TIMEOUT:-7200}"
fi

if [[ ! -e /dev/kvm ]]; then
  echo "KVM device /dev/kvm not available" >&2
  echo "dockur/windows needs KVM for usable XP testing" >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "docker is not available or not permitted for this user" >&2
  exit 1
fi

"$ROOT/scripts/check-xp-pe.sh" "$XP_DIR/oem"

rm -f "$RESULT" "$SHARED/smoke.out" "$SHARED/smoke-amd64.out"
rm -f "$XP_DIR/oem/xp-smoke-amd64.exe" "$SHARED/xp-smoke-amd64.exe"
mkdir -p "$SHARED"
: > "$SHARED/.keep"

"$ROOT/scripts/fetch-xp-iso.sh"

setup_storage_dir

validate_xp_storage

echo "starting Windows XP container (web UI on http://127.0.0.1:8006/)"
echo "first boot downloads and installs XP then runs docker/xp/oem/install.bat"
"${COMPOSE[@]}" up -d

cleanup() {
  "${COMPOSE[@]}" down --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

deadline=$((SECONDS + TIMEOUT_SECS))
echo "waiting up to ${TIMEOUT_SECS}s for PASS/FAIL in $RESULT"
while (( SECONDS < deadline )); do
  if [[ -f "$RESULT" ]]; then
    status="$(tr -d '[:space:]' < "$RESULT" || true)"
    case "$status" in
      PASS|FAIL)
        echo "guest reported: $status"
        if [[ -f "$SHARED/smoke.out" ]]; then
          echo "---- smoke.out ----"
          cat "$SHARED/smoke.out"
          echo "-------------------"
        fi
        if [[ "$status" == "PASS" ]]; then
          echo "Windows XP Docker smoke test passed"
          exit 0
        fi
        echo "Windows XP Docker smoke test failed" >&2
        "${COMPOSE[@]}" logs --no-color | tail -n 80 >&2 || true
        exit 1
        ;;
      *)
        # Intermediate markers such as "starting" are ignored.
        ;;
    esac
  fi
  sleep 10
done

echo "timed out waiting for XP smoke result" >&2
"${COMPOSE[@]}" logs --no-color | tail -n 120 >&2 || true
exit 1
