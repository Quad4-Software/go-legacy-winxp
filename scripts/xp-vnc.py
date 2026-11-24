#!/usr/bin/env python3
"""Minimal VNC/RFB client for driving the dockurr/windows XP guest headlessly.

Connects to the QEMU VNC server (default 127.0.0.1:5900), performs the RFB
handshake, requests a 32bpp true-color pixel format and raw encoding only,
then can send key events, type text, or capture the screen as PNG.

Usage:
  xp-vnc.py shot out.png [--host H] [--port P]
  xp-vnc.py key ctrl-c
  xp-vnc.py combo ctrl-alt-delete
  xp-vnc.py type "hello world"
  xp-vnc.py wait-pixel R G B [--timeout N]

Key names: letters/digits, F1-F12, Return/Enter, Escape, Tab, Space,
BackSpace, Delete, Insert, Home, End, Page_Up, Page_Down, Up, Down, Left,
Right, Control_L/Control_R, Alt_L/Alt_R, Shift_L/Shift_R, Caps_Lock,
Num_Lock, Scroll_Lock, Pause/Break, Minus, Plus, Equal, Comma, Period,
Slash, Semicolon, Apostrophe, Backquote, BracketLeft, BracketRight,
Backslash.
"""

import argparse
import socket
import struct
import sys
import time
import zlib

KEYSYMS = {
    "ctrl": 0xFFE3, "control": 0xFFE3, "control_l": 0xFFE3, "ctrl_l": 0xFFE3,
    "ctrl_r": 0xFFE4, "control_r": 0xFFE4,
    "alt": 0xFFE9, "alt_l": 0xFFE9, "alt_r": 0xFFEA,
    "shift": 0xFFE1, "shift_l": 0xFFE1, "shift_r": 0xFFE2,
    "super": 0xFFEB, "super_l": 0xFFEB, "super_r": 0xFFEC,
    "return": 0xFF0D, "enter": 0xFF0D, "kp_enter": 0xFF8D,
    "escape": 0xFF1B, "esc": 0xFF1B, "tab": 0xFF09,
    "space": 0x20, "backspace": 0xFF08, "delete": 0xFFFF,
    "insert": 0xFF63, "home": 0xFF50, "end": 0xFF57,
    "page_up": 0xFF55, "prior": 0xFF55, "page_down": 0xFF56, "next": 0xFF56,
    "up": 0xFF52, "down": 0xFF54, "left": 0xFF51, "right": 0xFF53,
    "caps_lock": 0xFFE5, "num_lock": 0xFF7F, "scroll_lock": 0xFF14,
    "pause": 0xFF13, "break": 0xFF6D,
    "minus": 0x002D, "plus": 0x002B, "equal": 0x003D,
    "comma": 0x002C, "period": 0x002E, "slash": 0x002F,
    "semicolon": 0x003B, "apostrophe": 0x0027, "backquote": 0x0060,
    "bracketleft": 0x005B, "bracketright": 0x005D, "backslash": 0x005C,
}
for i in range(1, 13):
    KEYSYMS[f"f{i}"] = 0xFFBD + i


def keysym(name):
    n = name.lower()
    if n in KEYSYMS:
        return KEYSYMS[n]
    if len(n) == 1:
        c = n
        if "a" <= c <= "z":
            return ord(c.upper())
        if c.isdigit():
            return ord(c)
        return ord(c)
    raise SystemExit(f"unknown key: {name}")


class RFB:
    def __init__(self, host, port, timeout=15):
        self.sock = socket.create_connection((host, port), timeout=timeout)
        self.sock.settimeout(timeout)
        self.buf = b""

    def read(self, n):
        while len(self.buf) < n:
            chunk = self.sock.recv(1 << 20)
            if not chunk:
                raise ConnectionError("vnc closed connection")
            self.buf += chunk
        out, self.buf = self.buf[:n], self.buf[n:]
        return out

    def write(self, data):
        self.sock.sendall(data)

    def handshake(self):
        ver = self.read(12)
        if not ver.startswith(b"RFB "):
            raise SystemExit(f"bad RFB greeting: {ver!r}")
        self.write(b"RFB 003.008\n")
        ntypes = self.read(1)[0]
        types = list(self.read(ntypes))
        if 1 in types:  # None
            self.write(b"\x01")
        elif 2 in types:  # VNC auth; try empty password
            self.write(b"\x02")
            self.read(16)  # challenge
            self.write(b"\x00" * 16)
        else:
            raise SystemExit(f"no supported security types: {types}")
        # RFB 3.8 sends a SecurityResult for every security type.
        res = struct.unpack(">I", self.read(4))[0]
        if res != 0:
            raise SystemExit(f"vnc security result {res}")
        self.write(b"\x01")  # ClientInit, shared
        # ServerInit
        data = self.read(24)
        self.width, self.height = struct.unpack(">HH", data[:4])
        nlen = struct.unpack(">I", data[20:24])[0]
        self.name = self.read(nlen).decode("utf-8", "replace")
        self._configure()

    def _configure(self):
        # SetPixelFormat: 32bpp, depth 24, little-endian, true-color RGB.
        pf = struct.pack(
            ">BBBBHHHBBBxxx", 32, 24, 0, 1, 255, 255, 255, 16, 8, 0
        )
        self.write(b"\x00\x00\x00\x00" + pf)
        # SetEncodings: raw only.
        self.write(b"\x02\x00" + struct.pack(">H", 1) + struct.pack(">i", 0))

    def drain(self, timeout=0.3):
        """Read and discard pending server messages."""
        self.sock.settimeout(timeout)
        try:
            while True:
                self.sock.recv(1 << 20)
        except socket.timeout:
            pass
        finally:
            self.sock.settimeout(15)

    def framebuffer(self):
        """Request a full (non-incremental) update, return RGB bytearray."""
        self.write(struct.pack(">BBhhhh", 3, 0, 0, 0, self.width, self.height))
        img = bytearray(self.width * self.height * 3)
        while True:
            msg = self.read(1)[0]
            if msg == 0:  # FramebufferUpdate
                # Server sends: padding(1) + number-of-rects(2)
                hdr = self.read(3)
                nrects = struct.unpack(">H", hdr[1:3])[0]
                for _ in range(nrects):
                    x, y, w, h, enc = struct.unpack(">hhhhI", self.read(12))
                    if enc != 0:
                        raise SystemExit(f"unexpected encoding {enc} (raw only requested)")
                    data = self.read(w * h * 4)
                    # 32bpp true-color LE (B,G,R,X per pixel).
                    for row in range(h):
                        src = row * w * 4
                        dst = ((y + row) * self.width + x) * 3
                        for col in range(w):
                            s = src + col * 4
                            img[dst + col * 3: dst + col * 3 + 3] = (
                                data[s + 2], data[s + 1], data[s]
                            )
                return img
            elif msg == 1:  # SetColourMapEntries
                self.read(5)
            elif msg == 2:  # Bell
                pass
            elif msg == 3:  # ServerCutText
                self.read(3)
                n = struct.unpack(">I", self.read(4))[0]
                self.read(n)
            elif msg == 150:  # EndOfContinuousUpdates
                self.read(3)
            elif msg == 248:  # Fence: padding(3) flags(4) length(1) payload
                self.read(3)
                flags = struct.unpack(">I", self.read(4))[0]
                ln = self.read(1)[0]
                self.read(ln)
                if not (flags & 1):  # not "block before" -> keep reading
                    pass
            else:
                raise SystemExit(f"unexpected server message type {msg}")

    def pointer_event(self, x, y, button_mask=0):
        # PointerEvent: type(1)=5 button-mask(1) x(2) y(2)
        self.write(struct.pack(">BBHH", 5, button_mask, x, y))

    def move(self, x, y, delay=0.05):
        self.pointer_event(x, y)
        time.sleep(delay)

    def click(self, x, y, button=1, delay=0.08):
        self.pointer_event(x, y, button)
        time.sleep(delay)
        self.pointer_event(x, y, 0)
        time.sleep(delay)

    def key_event(self, ks, down):
        # KeyEvent: type(1) down-flag(1) pad(2) keysym(4) pad(4)
        self.write(struct.pack(">BBHII", 4, 1 if down else 0, 0, ks, 0))

    def press(self, name, delay=0.04):
        ks = keysym(name)
        self.key_event(ks, True)
        time.sleep(delay)
        self.key_event(ks, False)
        time.sleep(delay)

    def combo(self, names, delay=0.05):
        kss = [keysym(n) for n in names]
        for ks in kss:
            self.key_event(ks, True)
            time.sleep(delay)
        for ks in reversed(kss):
            self.key_event(ks, False)
            time.sleep(delay)

    def type_text(self, text):
        for ch in text:
            if ch == "\n":
                self.press("enter")
                continue
            if ch == " ":
                self.press("space")
                continue
            ks = keysym(ch)
            if ch.isalpha() and ch.isupper():
                self.combo(["shift", ch])
            else:
                self.key_event(ks, True)
                time.sleep(0.02)
                self.key_event(ks, False)
                time.sleep(0.02)

    def close(self):
        try:
            self.sock.close()
        except Exception:
            pass


def write_png(path, w, h, rgb):
    raw = bytearray()
    stride = w * 3
    for y in range(h):
        raw.append(0)
        raw += rgb[y * stride:(y + 1) * stride]
    comp = zlib.compress(bytes(raw), 6)

    def chunk(tag, data):
        return (struct.pack(">I", len(data)) + tag + data +
                struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF))

    png = (b"\x89PNG\r\n\x1a\n" +
           chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)) +
           chunk(b"IDAT", comp) +
           chunk(b"IEND", b""))
    with open(path, "wb") as f:
        f.write(png)


def main():
    ap = argparse.ArgumentParser(description="headless VNC driver for the XP guest")
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=5900)
    sub = ap.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("shot", help="capture screen to PNG")
    p.add_argument("out")

    p = sub.add_parser("key", help="press a single key")
    p.add_argument("name")

    p = sub.add_parser("combo", help="press keys together (e.g. ctrl-c)")
    p.add_argument("keys", nargs="+")

    p = sub.add_parser("type", help="type text")
    p.add_argument("text")

    p = sub.add_parser("wait-pixel", help="poll until a pixel matches RGB")
    p.add_argument("x", type=int)
    p.add_argument("y", type=int)
    p.add_argument("r", type=int)
    p.add_argument("g", type=int)
    p.add_argument("b", type=int)
    p.add_argument("--timeout", type=float, default=60)

    args = ap.parse_args()

    rfb = RFB(args.host, args.port)
    rfb.handshake()
    print(f"connected: {rfb.width}x{rfb.height} '{rfb.name}'", file=sys.stderr)

    if args.cmd == "key":
        rfb.press(args.name)
    elif args.cmd == "combo":
        seq = []
        for part in args.keys:
            seq.extend(part.split("-"))
        rfb.combo(seq)
    elif args.cmd == "type":
        rfb.type_text(args.text)
    elif args.cmd == "shot":
        img = rfb.framebuffer()
        write_png(args.out, rfb.width, rfb.height, img)
        print(f"wrote {args.out} ({rfb.width}x{rfb.height})", file=sys.stderr)
    elif args.cmd == "wait-pixel":
        deadline = time.time() + args.timeout
        idx = (args.y * rfb.width + args.x) * 3
        while time.time() < deadline:
            img = rfb.framebuffer()
            r, g, b = img[idx], img[idx + 1], img[idx + 2]
            if (r, g, b) == (args.r, args.g, args.b):
                print(f"pixel matched at {args.x},{args.y}", file=sys.stderr)
                return
            time.sleep(0.5)
        raise SystemExit("pixel not found before timeout")
    rfb.close()


if __name__ == "__main__":
    main()
