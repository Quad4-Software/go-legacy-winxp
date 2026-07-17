#!/usr/bin/env bash
# Build XP smoke binaries and run them inside dockur/windows VERSION=xp.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
XP_DIR="$ROOT/docker/xp"
SHARED="$XP_DIR/shared"
RESULT="$SHARED/result.txt"
TIMEOUT_SECS="${XP_TEST_TIMEOUT:-3600}"
COMPOSE=(docker compose -f "$XP_DIR/docker-compose.yml")

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
mkdir -p "$SHARED"
: > "$SHARED/.keep"

# Windows Setup is unreliable on btrfs-backed disks. Prefer ext4/xfs under /tmp.
storage_fs="$(df -T "$XP_DIR" | awk 'NR==2 {print $2}')"
if [[ "$storage_fs" == "btrfs" ]]; then
  alt="/tmp/go-legacy-winxp-xp-storage"
  mkdir -p "$alt"
  if [[ -L "$XP_DIR/storage" || ! -e "$XP_DIR/storage" ]]; then
    rm -rf "$XP_DIR/storage"
    ln -sfn "$alt" "$XP_DIR/storage"
    echo "using $alt for XP disk (project filesystem is $storage_fs)"
  fi
else
  mkdir -p "$XP_DIR/storage"
fi

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
