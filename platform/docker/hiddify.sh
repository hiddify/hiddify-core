#!/bin/sh
# CONFIG: the profile (a subscription link or a config).
# ROUTER_MODE=1: router mode, the whole network behind this host goes through the proxy (TCP, UDP
# and ICMP) - with network_mode host on a Linux gateway, or as a RouterOS container that the router
# sends its LAN to (docs/router-mode.md). It needs NET_ADMIN, /dev/net/tun, and the host's nfqueue
# kernel module (a container can not load it).

set -- run --config "$CONFIG"
[ -f "/hiddify/hiddify.json" ] && set -- "$@" -d /hiddify/hiddify.json
case "$ROUTER_MODE" in
1 | true | yes) set -- "$@" --router ;;
esac
exec /hiddify/hiddify-core "$@"
