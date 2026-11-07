# Router mode

Router mode makes a Linux router or gateway a transparent proxy for its whole network. Every
device behind it goes through the proxy without any setup on the device: phones, TVs, consoles,
IoT. That covers TCP, UDP (DNS, QUIC, games, calls) and ICMP (ping). Connections that come in
from outside still work: to the router itself (SSH, its web UI, a VPN server on it) and port
forwards to a device behind it.

| Where | How |
|---|---|
| OpenWrt | the `hiddify-core` package, `option mode 'router'` (the default of a new install) |
| Any Linux gateway | `hiddify-core run -c <profile> --router`, as root |
| Docker on a Linux gateway | `ROUTER_MODE=1`, `network_mode: host`, `NET_ADMIN`, `/dev/net/tun` |
| MikroTik (RouterOS 7) | hiddify-core on a box beside it (or a RouterOS container), and `platform/mikrotik/hiddify-gateway.rsc` |
| Hiddify app on Linux | Settings → Service mode → Router |

## How it works

Router mode is the TUN with sing-box `auto_redirect`, which does the routing with nftables:

- **TCP** the router forwards or makes itself is redirected to the proxy by the `prerouting` and
  `output` chains.
- **UDP and ICMP** are marked by those chains and routed into the TUN by an `ip rule` on that mark.
  DNS on port 53 is answered by the proxy's DNS, so names are not resolved by the ISP.
- **Connections from outside** are left alone: the chains skip the reply direction of a connection
  (`ct direction reply`), and only marked packets are routed into the TUN. Replies to SSH on the
  router, or to a port forward, keep the main routing table and go out the way they came in.
- **The router's own networks** (its addresses and the subnets on its interfaces) and private
  destinations stay direct, so the LAN keeps reaching the router and its other devices.
- On OpenWrt, sing-tun adds the TUN to firewall4 (`/etc/nftables.d/0-sing-box-auto-redirect.nft`).

When it starts, router mode turns on IPv4 and IPv6 forwarding if they are off (an interface that
took router advertisements keeps taking them), and loads the nfqueue modules. It puts forwarding
back as it was when it stops.

Router mode never runs without `auto_redirect`. The plain TUN is not a router: on OpenWrt the
firewall drops the traffic the network sends to it, and replies to port forwards are routed into it.
Where `auto_redirect` can not start, router mode does not start, and says what is missing.

### Requirements

- Linux, as root (it changes the routing and the firewall).
- nftables in the kernel and the nfqueue module: on OpenWrt `kmod-nft-queue` (a dependency of the
  package); on Debian and Ubuntu the `nfnetlink_queue` module, loaded by router mode where `modprobe`
  is there. A container can not load modules: load them on the host.
- `/dev/net/tun`.

Without the nfqueue module the start fails with:

```
router mode needs nftables and the nfqueue kernel module (OpenWrt: kmod-nft-queue; Debian, Ubuntu: nftables and the nfnetlink_queue module): ...
```

## OpenWrt

Install the package (see [platform/wrt/README.md](../platform/wrt/README.md)), put the profile in
`/etc/hiddify-core/config.json` (a subscription link in the file works), and:

```sh
uci set hiddify-core.main.mode='router'   # 'proxy' for the proxy ports only
uci commit hiddify-core
/etc/init.d/hiddify-core restart
logread -e hiddify-core
```

An install from before this option has no `mode`, and keeps running as a proxy until it is set.

## Any Linux gateway, or Docker

```sh
sudo hiddify-core run -c 'https://example.com/sub' --router
```

With a settings file (`-d hiddify.json`), `--router` still holds, or put `"router-mode": true` in
the file. With Docker, on the gateway itself:

```yaml
services:
  hiddify:
    image: ghcr.io/hiddify/hiddify-core:latest
    network_mode: host
    cap_add: [NET_ADMIN]
    devices: ["/dev/net/tun:/dev/net/tun"]
    environment:
      CONFIG: "https://example.com/sub"
      ROUTER_MODE: "1"
```

## MikroTik

RouterOS can not run hiddify-core itself. It runs on a box the MikroTik sends its LAN to: another
Linux machine, an OpenWrt router, or a container on the MikroTik. The box runs router mode; the
MikroTik keeps routing, NAT and the WAN.

1. Give the box an address on a subnet of its own: a container's veth, a VLAN, or another port.
   Not the LAN: there its replies reach the devices directly, and RouterOS drops the connections
   it sees only half of.
2. Run router mode on the box, with its default route through the MikroTik.
3. On the MikroTik, upload `platform/mikrotik/hiddify-gateway.rsc` and run:

   ```
   :global hiddifyGateway 172.17.0.2
   /import hiddify-gateway.rsc
   ```

The script, everything marked with the comment `hiddify`:

- adds a routing table `hiddify` whose default route is the box, with `check-gateway=ping`: while
  the box does not answer, the LAN goes out directly;
- marks connections from the WAN when they start (`connection-state=new`), so their replies keep
  the main table: the router's own services and port forwards keep working;
- routes the LAN's other traffic to the internet through the box, all protocols;
- points the router's DNS at the box, so names are resolved through the proxy;
- turns fasttrack off: a fasttracked packet skips the routing mark.

`/import hiddify-gateway-remove.rsc` takes it all out, and gives the router back its DNS servers and
fasttrack. The script covers IPv4; for IPv6, mark and route it the same way under `/ipv6`, or turn
IPv6 off on the LAN, or it goes out directly.

### As a RouterOS container

On a MikroTik with the container package and enough memory (512 MB or more), the box can be a
container. Enable containers (`/system/device-mode/update container=yes`, then confirm with the
power button or a cold boot), then:

```
/interface/veth/add name=veth-hiddify address=172.17.0.2/24 gateway=172.17.0.1
/interface/bridge/add name=containers
/ip/address/add address=172.17.0.1/24 interface=containers
/interface/bridge/port/add bridge=containers interface=veth-hiddify
/ip/firewall/nat/add chain=srcnat src-address=172.17.0.0/24 out-interface-list=WAN action=masquerade
/container/config/set registry-url=https://ghcr.io tmpdir=disk1/tmp
/container/envs/add list=hiddify key=CONFIG value="https://example.com/sub"
/container/envs/add list=hiddify key=ROUTER_MODE value=1
/container/add remote-image=ghcr.io/hiddify/hiddify-core:latest interface=veth-hiddify envlists=hiddify root-dir=disk1/hiddify start-on-boot=yes logging=yes
/container/start [find interface=veth-hiddify]
```

Leave the `containers` bridge out of the `LAN` interface list. The container needs a TUN device and
the nfqueue module in the RouterOS kernel; check its log (`/log/print where topics~"container"`)
for the router mode error above.

## Testing it

From a device behind the router:

```sh
curl -s https://ipinfo.io/ip      # the proxy's address, not the ISP's
nslookup example.com 8.8.8.8      # answered through the proxy
ping -c 3 1.1.1.1                 # ICMP through the proxy
```

From outside, the router's SSH and a port forward to a device behind it still answer. The core's
debug log (`--log debug`) shows each connection of the LAN: `inbound redirect connection` (TCP),
`inbound packet connection` and `inbound DNS packet` (UDP), `forward icmp connection` (ping).
