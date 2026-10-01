#!/usr/bin/env python3
"""Relay MeshChatX TCP hubs through the host so the XP guest can reach them.

The docker-compose network on this VM cannot originate internet TCP.
The host can. Listen on the docker-bridge IP and dial the real hubs.
"""

from __future__ import annotations

import socket
import threading

BIND = "172.18.0.1"

# listen_port -> (hub_ip, hub_port, name)
MAP = {
    14242: ("77.37.146.243", 4242, "Catz Node (TCP)"),
    14965: ("45.77.109.86", 4965, "RNS_Transport_US-East"),
    14966: ("193.26.158.230", 4965, "RNS TCP Node Germany 002"),
    14243: ("188.166.83.139", 4242, "rns.h.acked.co.uk"),
    14244: ("172.245.119.67", 4242, "RNS4All"),
    14245: ("176.84.102.17", 4242, "Air Barcelona"),
    14246: ("103.195.4.226", 4242, "ZHULONG1 - Hong Kong Node"),
}


def pipe(a: socket.socket, b: socket.socket) -> None:
    try:
        while True:
            data = a.recv(65536)
            if not data:
                break
            b.sendall(data)
    except OSError:
        pass
    finally:
        try:
            a.shutdown(socket.SHUT_RDWR)
        except OSError:
            pass
        try:
            b.shutdown(socket.SHUT_RDWR)
        except OSError:
            pass


def handle(client: socket.socket, dest: tuple[str, int], name: str, peer: str) -> None:
    remote = None
    try:
        remote = socket.create_connection(dest, timeout=8)
        remote.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        client.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        print("relay %s %s -> %s:%d" % (peer, name, dest[0], dest[1]), flush=True)
        t = threading.Thread(target=pipe, args=(remote, client), daemon=True)
        t.start()
        pipe(client, remote)
        t.join(timeout=1)
    except OSError as exc:
        print("relay fail %s %s: %s" % (peer, name, exc), flush=True)
    finally:
        try:
            client.close()
        except OSError:
            pass
        if remote is not None:
            try:
                remote.close()
            except OSError:
                pass


def listen(port: int, dest: tuple[str, int], name: str) -> None:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind((BIND, port))
    s.listen(64)
    print("listen %s:%d for %s %s:%d" % (BIND, port, name, dest[0], dest[1]), flush=True)
    while True:
        client, addr = s.accept()
        threading.Thread(
            target=handle,
            args=(client, dest, name, "%s:%d" % addr),
            daemon=True,
        ).start()


def main() -> None:
    for port, (host, hub_port, name) in MAP.items():
        threading.Thread(
            target=listen,
            args=(port, (host, hub_port), name),
            daemon=True,
        ).start()
    threading.Event().wait()


if __name__ == "__main__":
    main()
