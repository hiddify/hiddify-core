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

### One package by hand

Every release carries a package for each OpenWrt architecture: `hiddify-core_<version>_<arch>.ipk`
for opkg (24.10 and earlier) and `hiddify-core-<version>-r0_<arch>.apk` for apk (25.12 and
later). Find the architecture with `opkg print-architecture` (the last line, e.g. `x86_64`,
`aarch64_cortex-a53`, `mipsel_24kc`) or `apk --print-arch`, download its package from the
[latest release](https://github.com/hiddify/hiddify-core/releases/latest), then:

```sh
opkg install ./hiddify-core_*.ipk                  # opkg
apk add --allow-untrusted ./hiddify-core-*.apk     # apk
```

To update, run the installer again, or install the next release's package the same way. The
package keeps `/etc/hiddify-core/` across `sysupgrade` (`/lib/upgrade/keep.d/hiddify-core`).

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
`make-feed.sh` writes and signs a feed's indexes the way OpenWrt's own build does: `usign` for
`Packages.sig`, `apk mkndx --sign` for `packages.adb`, for anyone hosting a feed of their own; the
releases publish the packages alone. The signing keys are described in
[keys/README.md](keys/README.md).
