#!/usr/bin/env bash
# Stage MeshChatX reticulum-go config on the XP samba share and launch it.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SHARED="$ROOT/docker/xp/shared"
CONFIG_SRC="$ROOT/testdata/xp-reticulum/config"
LIVE_BAT="$ROOT/testdata/xp-reticulum/live.bat"
CONTAINER="${XP_CONTAINER:-go-legacy-winxp-test}"
RESULT="$SHARED/live-result.txt"
TIMEOUT="${XP_LIVE_TIMEOUT:-240}"

if [[ ! -f "$CONFIG_SRC" ]]; then
  echo "missing $CONFIG_SRC (run scripts/fetch-mcx-config.py first)" >&2
  exit 1
fi
if [[ ! -f "$SHARED/reticulum-go-winxp.exe" ]]; then
  echo "missing $SHARED/reticulum-go-winxp.exe (run scripts/build-reticulum-xp.sh first)" >&2
  exit 1
fi
if ! docker inspect "$CONTAINER" >/dev/null 2>&1; then
  echo "container $CONTAINER is not running" >&2
  exit 1
fi

hubs_up() {
  python3 - "$SHARED/live-status.json" <<'PY'
import json, re, os, sys
path = sys.argv[1]
if not os.path.exists(path):
    print(0)
    raise SystemExit
raw = open(path, encoding="utf-8", errors="replace").read()
m = re.search(r"\{.*\}\s*$", raw, re.S)
if not m:
    print(0)
    raise SystemExit
data = json.loads(m.group(0))
n = 0
for iface in data.get("interfaces") or []:
    if iface.get("type") == "LocalServerInterface":
        continue
    if iface.get("status"):
        n += 1
print(n)
PY
}

rm -f "$RESULT" "$SHARED/live-status.txt" "$SHARED/live-status.json" \
  "$SHARED/live-status.exit" "$SHARED/live-daemon.log" "$SHARED/live-version.txt"
cp -f "$CONFIG_SRC" "$SHARED/config"
cp -f "$LIVE_BAT" "$SHARED/live.bat"
python3 - "$SHARED/live.bat" <<'PY'
import sys
path = sys.argv[1]
data = open(path, "rb").read().replace(b"\r\n", b"\n").replace(b"\n", b"\r\n")
open(path, "wb").write(data)
PY

# Host can reach MeshChatX TCP. The compose network cannot, so relay + DNAT.
if ! pgrep -f "scripts/mcx-host-relay.py" >/dev/null 2>&1; then
  python3 "$ROOT/scripts/mcx-host-relay.py" >/tmp/mcx-host-relay.log 2>&1 &
  sleep 0.4
fi
"$ROOT/scripts/xp-guest-nat.sh"

docker cp "$ROOT/scripts/xp-sendkeys-live.py" "$CONTAINER:/tmp/sendkeys-live.py"
docker exec "$CONTAINER" python3 /tmp/sendkeys-live.py

echo "waiting up to ${TIMEOUT}s for $RESULT"
deadline=$((SECONDS + TIMEOUT))
while (( SECONDS < deadline )); do
  if [[ -f "$RESULT" ]]; then
    status="$(tr -d '[:space:]' < "$RESULT" || true)"
    if [[ "$status" == "PASS" || "$status" == "FAIL" || "$status" == "DAEMON" ]]; then
      up="$(hubs_up)"
      echo "live-result=$status hubs_up=$up"
      [[ -f "$SHARED/live-status.txt" ]] && cat "$SHARED/live-status.txt"
      if [[ "$status" == "FAIL" ]]; then
        exit 1
      fi
      if [[ "${up:-0}" -lt 1 ]]; then
        echo "daemon answered but no MeshChatX hub is Up" >&2
        exit 1
      fi
      echo PASS > "$RESULT"
      exit 0
    fi
  fi
  sleep 2
done

echo "timeout waiting for XP live reticulum-go result" >&2
[[ -f "$SHARED/live-status.txt" ]] && cat "$SHARED/live-status.txt" >&2
exit 1
