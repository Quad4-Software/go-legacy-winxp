#!/usr/bin/env python3
"""Type Win-R cmd and Z:\\live.bat into the XP guest via QEMU monitor."""

import socket
import time

MONITOR = "/run/shm/monitor.sock"

s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.settimeout(5)
s.connect(MONITOR)


def recv_some():
    s.settimeout(0.05)
    try:
        while True:
            if not s.recv(4096):
                break
    except socket.timeout:
        pass
    s.settimeout(5)


recv_some()


def cmd(line, pause=0.02):
    s.sendall((line + "\n").encode())
    time.sleep(pause)
    recv_some()


def sendkey(k, pause=0.1):
    cmd("sendkey %s" % k, pause)


SHIFT = {
    ":": "shift-semicolon",
    "\\": "backslash",
    "/": "slash",
    ".": "dot",
    "-": "minus",
    " ": "spc",
    "=": "equal",
}


def type_text(text):
    for ch in text:
        if ch in SHIFT:
            sendkey(SHIFT[ch])
            continue
        if ch.isupper():
            sendkey("shift-%s" % ch.lower())
            continue
        if ch.isalnum():
            sendkey(ch.lower())
            continue
        raise SystemExit("unmapped %r" % ch)


sendkey("meta_l-r", 1.0)
type_text("cmd")
sendkey("ret", 1.4)
sendkey("alt-spc", 0.4)
sendkey("x", 0.6)
type_text("cd /d C")
sendkey("shift-semicolon")
sendkey("backslash")
sendkey("ret", 0.4)
type_text("cls")
sendkey("ret", 0.3)
type_text("Z")
sendkey("shift-semicolon")
sendkey("backslash")
type_text("live.bat")
sendkey("ret", 0.4)
print("typed Z:\\live.bat")
