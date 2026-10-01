#!/usr/bin/env bash
# DNAT XP guest MeshChatX hub IPs to the host TCP relay on 172.18.0.1.
# The compose network on this VM cannot originate internet TCP. The host can.
set -euo pipefail

CONTAINER="${XP_CONTAINER:-go-legacy-winxp-test}"

docker exec "$CONTAINER" sh -c '
sysctl -w net.ipv4.ip_forward=1 >/dev/null
add() { iptables -t nat -C PREROUTING "$@" 2>/dev/null || iptables -t nat -I PREROUTING 1 "$@"; }
add -s 172.30.0.2 -p tcp -d 77.37.146.243 --dport 4242 -j DNAT --to-destination 172.18.0.1:14242
add -s 172.30.0.2 -p tcp -d 45.77.109.86 --dport 4965 -j DNAT --to-destination 172.18.0.1:14965
add -s 172.30.0.2 -p tcp -d 193.26.158.230 --dport 4965 -j DNAT --to-destination 172.18.0.1:14966
add -s 172.30.0.2 -p tcp -d 188.166.83.139 --dport 4242 -j DNAT --to-destination 172.18.0.1:14243
add -s 172.30.0.2 -p tcp -d 172.245.119.67 --dport 4242 -j DNAT --to-destination 172.18.0.1:14244
add -s 172.30.0.2 -p tcp -d 176.84.102.17 --dport 4242 -j DNAT --to-destination 172.18.0.1:14245
add -s 172.30.0.2 -p tcp -d 103.195.4.226 --dport 4242 -j DNAT --to-destination 172.18.0.1:14246
iptables -t nat -C POSTROUTING -d 172.18.0.1 -p tcp -j MASQUERADE 2>/dev/null || \
  iptables -t nat -I POSTROUTING 1 -d 172.18.0.1 -p tcp -j MASQUERADE
iptables -C FORWARD -i docker -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -i docker -j ACCEPT
iptables -C FORWARD -o docker -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -o docker -j ACCEPT
iptables -C FORWARD -i qemu -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -i qemu -j ACCEPT
echo guest DNAT ready
'
