# hiddify-core on OpenWrt

hiddify-core runs on OpenWrt routers as a procd service. Every release ships packages for every
OpenWrt architecture, in both package formats:

| OpenWrt | Package manager | Package | Index in the feed |
|---|---|---|---|
| 24.10 and earlier | opkg | `hiddify-core_<version>_<arch>.ipk` | `Packages`, `Packages.gz`, `Packages.sig` |
| 25.12 and later | apk | `hiddify-core-<version>-r0_<arch>.apk` | `packages.adb` |

## Install

### The installer

```sh
sh <(wget -qO- https://raw.githubusercontent.com/hiddify/hiddify-core/main/installer.sh)
```

It tells opkg from apk, finds the router's architecture, installs the package of the latest
release and starts the service. Run it again to update.

### The feed

A feed keeps the package updated with `opkg upgrade` / `apk upgrade`. Each architecture has its
own feed, the assets of the release tagged `openwrt-<arch>` in this repository, replaced on every
version. Find the architecture with `opkg print-architecture` (the last line, e.g. `x86_64`,
`aarch64_cortex-a53`, `mipsel_24kc`) or `apk --print-arch`.

opkg (24.10 and earlier):

```sh
ARCH=$(opkg print-architecture | awk '{print $2}' | tail -n1)
FEED=https://github.com/hiddify/hiddify-core/releases/download/openwrt-$ARCH
wget -qO /tmp/hiddify.pub "$FEED/hiddify.pub" && opkg-key add /tmp/hiddify.pub
echo "src/gz hiddify $FEED" >> /etc/opkg/customfeeds.conf
opkg update && opkg install hiddify-core
```

apk (25.12 and later):

```sh
ARCH=$(apk --print-arch)
FEED=https://github.com/hiddify/hiddify-core/releases/download/openwrt-$ARCH
wget -qO /etc/apk/keys/hiddify-apk.pem "$FEED/hiddify-apk.pem"
echo "$FEED/packages.adb" > /etc/apk/repositories.d/hiddify.list
apk update && apk add hiddify-core
```

The keys verify the feed's index. They are kept across `sysupgrade` by the package's
`/lib/upgrade/keep.d/hiddify-core`, along with `/etc/hiddify-core/`.

### One package by hand

Download the package for the architecture from the
[release](https://github.com/hiddify/hiddify-core/releases/latest), then
`opkg install ./hiddify-core_*.ipk` or `apk add --allow-untrusted ./hiddify-core-*.apk`.

## Configure

| File | What |
|---|---|
| `/etc/config/hiddify-core` | UCI: `enabled`, `conffile`, `workdir`, `log_stderr` |
| `/etc/hiddify-core/config.json` | the sing-box config, or a Hiddify subscription link in a file |
| `/usr/share/hiddify-core/` | working directory: cache, geo data, logs |

```sh
uci set hiddify-core.main.conffile='/etc/hiddify-core/config.json'
uci commit hiddify-core
/etc/init.d/hiddify-core restart
logread -e hiddify-core
```

The service is `/etc/init.d/hiddify-core` (`start`, `stop`, `restart`, `enable`, `disable`);
it runs `hiddify-core run -c <conffile> -D <workdir>`. TUN mode needs `kmod-tun` and nftables
(`firewall4`), which the package depends on.

## Build from source with the OpenWrt SDK

The `Makefile` beside this file is an OpenWrt package. Add this directory to an SDK or
buildroot as `package/hiddify-core` (or point a feed at `platform/wrt`), then:

```sh
./scripts/feeds update -a && ./scripts/feeds install golang
make package/hiddify-core/compile V=s
```

It clones the repository at the version's tag, with the submodules the GitHub tarball lacks;
`make release` bumps `PKG_VERSION` with the rest of the release. `GO_PKG_TAGS` is the repository
Makefile's `WRT_TAGS`: no naive outbound (cronet is a native library) and no call tunnel
(WebRTC), with the low-memory build.

## How the packages are built

`release/openwrt/build-ipk.sh` and `build-apk.sh` package a cross-built binary for one
architecture (`make openwrt-packages WRT_ARCH=x86_64` does both for a local binary), and
`make-feed.sh` writes and signs the indexes the way OpenWrt's own build does: `usign` for
`Packages.sig`, `apk mkndx --sign` for `packages.adb`. The signing keys are described in
[keys/README.md](keys/README.md).
