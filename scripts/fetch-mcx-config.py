#!/usr/bin/env python3
"""Build a reticulum-go INI from MeshChatX directory interfaces.

Picks online clearnet IPv4 TCP and backbone hubs. Skips I2P, Yggdrasil,
and hostname-only entries so Windows XP does not need extra overlays.
"""

from __future__ import annotations

import argparse
import ipaddress
import json
import os
import sys
import urllib.request
from datetime import datetime, timezone

DEFAULT_API = "https://meshchatx.com/api/mcx-interfaces"

PREFERRED = [
    "Catz Node (TCP)",
    "RNS_Transport_US-East",
    "RNS TCP Node Germany 002",
    "rns.h.acked.co.uk",
    "RNS4All",
    "Air Barcelona",
    "ZHULONG1 - Hong Kong Node",
]


def is_ipv4(host: str) -> bool:
    try:
        ipaddress.IPv4Address(host)
        return True
    except ValueError:
        return False


def load_api(url: str) -> dict:
    req = urllib.request.Request(url, headers={"User-Agent": "go-legacy-winxp-mcx/1.0"})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode("utf-8"))


def choose(interfaces: list[dict], limit: int) -> list[dict]:
    by_name = {item.get("name"): item for item in interfaces}
    picked: list[dict] = []
    seen: set[tuple[str, int]] = set()

    def add(item: dict) -> None:
        if item is None:
            return
        if item.get("status") != "online":
            return
        if item.get("network") != "clearnet":
            return
        host = item.get("host") or ""
        if not is_ipv4(host):
            return
        kind = item.get("type")
        if kind not in ("tcp", "backbone"):
            return
        port = int(item.get("port") or 0)
        key = (host, port)
        if key in seen:
            return
        seen.add(key)
        picked.append(item)

    for name in PREFERRED:
        if len(picked) >= limit:
            break
        add(by_name.get(name))

    for item in interfaces:
        if len(picked) >= limit:
            break
        add(item)
    return picked


def render(interfaces: list[dict], source: str, fetched_at: str) -> str:
    lines = [
        "# reticulum-go config for Windows XP live tests.",
        "# Generated from %s at %s" % (source, fetched_at),
        "# Sandbox off: Windows XP cannot use the full Go sandbox profile.",
        "# share_instance uses TCP because XP has no Unix sockets.",
        "",
        "[reticulum]",
        "  enable_transport = yes",
        "  share_instance = yes",
        "  shared_instance_type = tcp",
        "  shared_instance_port = 37428",
        "  instance_control_port = 37429",
        "  enable_sandbox = no",
        "  enable_seccomp = no",
        "  sandbox_strict = no",
        "  panic_on_interface_error = no",
        "",
        "[logging]",
        "  loglevel = 4",
        "  destination = file",
        "  format = text",
        "",
    ]
    for item in interfaces:
        raw = (item.get("config") or "").strip("\n")
        if not raw:
            name = item["name"]
            if item.get("type") == "backbone":
                raw = (
                    "[[%s]]\n type = BackboneInterface\n enabled = yes\n"
                    " remote = %s\n target_port = %s"
                    % (name, item["host"], item["port"])
                )
            else:
                raw = (
                    "[[%s]]\n type = TCPClientInterface\n enabled = yes\n"
                    " target_host = %s\n target_port = %s"
                    % (name, item["host"], item["port"])
                )
        # Keep API blocks, then force unlimited reconnects for a long-lived guest.
        if "max_reconnect_tries" not in raw:
            raw = raw + "\n max_reconnect_tries = -1"
        lines.append(raw)
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--api", default=DEFAULT_API)
    parser.add_argument("--limit", type=int, default=7)
    parser.add_argument(
        "-o",
        "--output",
        default=os.path.join(
            os.path.dirname(__file__), "..", "testdata", "xp-reticulum", "config"
        ),
    )
    parser.add_argument("--json-in", default="", help="use a saved API JSON file")
    args = parser.parse_args()

    if args.json_in:
        data = json.loads(open(args.json_in, encoding="utf-8").read())
        source = args.json_in
    else:
        data = load_api(args.api)
        source = args.api

    interfaces = data.get("interfaces") or []
    picked = choose(interfaces, args.limit)
    if not picked:
        print("no IPv4 clearnet TCP/backbone hubs found", file=sys.stderr)
        return 1

    fetched_at = data.get("fetchedAt") or datetime.now(timezone.utc).isoformat()
    text = render(picked, source, fetched_at)
    out = os.path.abspath(args.output)
    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, "w", encoding="utf-8", newline="\n") as fh:
        fh.write(text)
    print("wrote %s (%d interfaces)" % (out, len(picked)))
    for item in picked:
        print("  %s %s %s:%s" % (item["typeName"], item["name"], item["host"], item["port"]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
