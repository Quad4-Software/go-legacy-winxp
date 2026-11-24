#!/usr/bin/env bash
# Exercise the Windows XP Ctrl+C paths inside the dockurr/windows guest.
#
# Requires the guest container to be running (see scripts/test-xp-docker.sh
# with XP_KEEP=1) with xp-agent.exe launched by run.bat/install.bat.
#
# Test 1: a Go program waiting on os.Interrupt must receive SIGINT when
#         CTRL_C_EVENT is generated for its console.
# Test 2: `go run` must terminate cleanly on CTRL_C_EVENT (issue #12).
#
# Env overrides:
#   XP_AGENT_DIR   agent dir on the host share (default docker/xp/shared/agent)
#   XP_AGENT_WAIT  per-command timeout seconds (default 150)
#   XP_GORUN_WAIT  seconds to wait for `go run` compile+READY (default 900)
#   XP_CTRL_WAIT   seconds to wait for processes to die after ctrl+c (default 90)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
AGENT="$ROOT/scripts/xp-agent.sh"
SHARE_GOROOT='\\host.lan\Data\go'
APPDIR='\\host.lan\Data\app'

export XP_AGENT_DIR="${XP_AGENT_DIR:-$ROOT/docker/xp/shared/agent}"
export XP_AGENT_WAIT="${XP_AGENT_WAIT:-150}"
GORUN_WAIT="${XP_GORUN_WAIT:-900}"
CTRL_WAIT="${XP_CTRL_WAIT:-90}"

fail() { echo "FAIL: $*" >&2; exit 1; }

agent() { "$AGENT" --timeout "$XP_AGENT_WAIT" "$1"; }

wait_file_grep() { # path pattern timeout
  local path="$1" pat="$2" deadline=$((SECONDS + ${3:-60}))
  while (( SECONDS < deadline )); do
    [[ -f "$path" ]] && grep -q "$pat" "$path" && return 0
    sleep 2
  done
  return 1
}

echo "== waiting for xp-agent =="
for i in $(seq 1 60); do
  if agent "ping" | grep -q pong; then break; fi
  [[ $i -eq 60 ]] && fail "xp-agent did not answer ping"
  sleep 5
done
echo "agent is up"

SHARED_HOST="$ROOT/docker/xp/shared"
rm -f "$SHARED_HOST/sigint.out" "$SHARED_HOST/gorun.out"

echo "== test 1: SIGINT delivery to a waiting Go program =="
agent 'sconsoles \\host.lan\Data\xp-smoke-386.exe __helper_sigint > \\host.lan\Data\sigint.out 2>&1' | tee /tmp/ctrlc-sigint-start.txt
PID1="$(sed -n 's/.*pid=\([0-9]*\).*/\1/p' /tmp/ctrlc-sigint-start.txt | head -1)"
[[ -n "$PID1" ]] || fail "no pid from sconsoles"
echo "sigint test pid=$PID1"

wait_file_grep "$SHARED_HOST/sigint.out" "READY" 120 || fail "sigint helper never printed READY"
echo "helper is waiting for a signal"

agent "ctrl c 0"
sleep 2

if wait_file_grep "$SHARED_HOST/sigint.out" "GOTINT" "$CTRL_WAIT"; then
  echo "PASS: SIGINT delivered, helper printed GOTINT"
else
  echo "FAIL: no GOTINT in sigint.out after ctrl+c" >&2
  cat "$SHARED_HOST/sigint.out" >&2 || true
  exit 1
fi

echo "== test 2: go run + ctrl+c (issue #12) =="
CMD2="set GOROOT=$SHARE_GOROOT& set CGO_ENABLED=0& $SHARE_GOROOT\\bin\\go.exe run $APPDIR\\gorun-app.go > \\\\host.lan\\Data\\gorun.out 2>&1"
agent "sconsoles $CMD2" | tee /tmp/ctrlc-gorun-start.txt
PID2="$(sed -n 's/.*pid=\([0-9]*\).*/\1/p' /tmp/ctrlc-gorun-start.txt | head -1)"
[[ -n "$PID2" ]] || fail "no pid from go run sconsoles"
echo "go run pid=$PID2 (compiling may take a while under TCG)"

if ! wait_file_grep "$SHARED_HOST/gorun.out" "APP READY" "$GORUN_WAIT"; then
  echo "FAIL: go run never reached APP READY" >&2
  cat "$SHARED_HOST/gorun.out" >&2 || true
  exit 1
fi
echo "go run app is up, sending ctrl+c"

agent "ctrl c 0"

# After ctrl+c, go.exe should kill the child and exit. Give it a bounded
# window, then verify no go/app processes remain.
deadline=$((SECONDS + CTRL_WAIT))
cmd_alive=""
while (( SECONDS < deadline )); do
  if agent "alive $PID2" | grep -q "state=running"; then
    cmd_alive="yes"
  else
    cmd_alive=""
    break
  fi
  sleep 3
done

PS="$(agent ps || true)"
echo "$PS" | grep -iE "go\.exe|gorun-app" && STILL="yes" || STILL="no"

if [[ -n "$cmd_alive" || "$STILL" == "yes" ]]; then
  echo "FAIL: processes still alive ${CTRL_WAIT}s after ctrl+c (issue #12 freeze?)" >&2
  echo "--- ps ---" >&2
  echo "$PS" >&2
  echo "--- gorun.out ---" >&2
  cat "$SHARED_HOST/gorun.out" >&2 || true
  exit 1
fi

echo "PASS: go run terminated after ctrl+c"
echo "== ctrl+c tests done =="
