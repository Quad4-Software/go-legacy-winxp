#!/usr/bin/env bash
# Send a command to xp-agent inside the XP guest via the shared folder and
# print its response. Usage:
#   scripts/xp-agent.sh "ping"
#   scripts/xp-agent.sh "runsh dir C:\\"
#   scripts/xp-agent.sh --timeout 60 "console C:\\GO\\bin\\go.exe run main.go"
#
# XP_AGENT_DIR overrides the agent directory (default docker/xp/shared/agent).
# XP_AGENT_WAIT overrides the response timeout in seconds (default 150).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
AGENT_DIR="${XP_AGENT_DIR:-$ROOT/docker/xp/shared/agent}"
WAIT="${XP_AGENT_WAIT:-150}"

if [[ "${1:-}" == "--timeout" ]]; then
  WAIT="$2"
  shift 2
fi

cmd="${1:?usage: xp-agent.sh [--timeout SEC] <command line>}"
[[ -n "$cmd" ]] || { echo "empty command" >&2; exit 2; }

mkdir -p "$AGENT_DIR"
id="$(date +%s%N)-$$"
cmdfile="$AGENT_DIR/cmd-$id.txt"
outfile="$AGENT_DIR/out-$id.txt"

printf '%s\n' "$cmd" > "$cmdfile"

deadline=$((SECONDS + WAIT))
while (( SECONDS < deadline )); do
  if [[ -f "$outfile" ]]; then
    cat "$outfile"
    rm -f "$outfile" "$cmdfile"
    exit 0
  fi
  # Agent may have failed the command and removed the cmd file without output.
  if [[ ! -f "$cmdfile" && ! -f "$outfile" ]]; then
    sleep 1
    [[ -f "$outfile" ]] && { cat "$outfile"; rm -f "$outfile"; exit 0; }
    echo "command file vanished without output (agent may be down)" >&2
    exit 1
  fi
  sleep 1
done

echo "timed out waiting for xp-agent (${WAIT}s)" >&2
rm -f "$cmdfile"
exit 1
