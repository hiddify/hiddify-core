# OpenWrt SDK package for hiddify-core: builds the CLI from source with OpenWrt's Go toolchain.
# Prebuilt packages for every OpenWrt architecture come from the release feeds instead; see
# README.md beside this file. `make release` in the repository bumps PKG_VERSION; CI checks
# GO_PKG_TAGS against `make print-WRT_TAGS` of the repository's Makefile.
include $(TOPDIR)/rules.mk

PKG_NAME:=hiddify-core
PKG_VERSION:=5.5.5
PKG_RELEASE:=1

# From git at the version's tag, with the submodules (hiddify-sing-box, ray2sing, the forks under
# replace/) that a GitHub tarball lacks and go.mod replaces into. No fixed hash: the tarball the
# buildroot makes from the clone is not reproducible across its versions.
PKG_SOURCE_PROTO:=git
PKG_SOURCE_URL:=https://github.com/hiddify/hiddify-core.git
PKG_SOURCE_VERSION:=v$(PKG_VERSION)
PKG_MIRROR_HASH:=skip

PKG_LICENSE:=GPL-3.0-or-later
PKG_LICENSE_FILES:=LICENSE.md
PKG_MAINTAINER:=Hiddify <wrt@hiddify.com>

PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_BUILD_FLAGS:=no-mips16

GO_PKG:=github.com/hiddify/hiddify-core
GO_PKG_BUILD_PKG:=$(GO_PKG)/cmd/main
GO_PKG_LDFLAGS:=-s -w
GO_PKG_LDFLAGS_X:= \
	$(GO_PKG)/v2/hcommon/constants.Version=$(PKG_VERSION) \
	github.com/sagernet/sing-box/constant.Version=$(PKG_VERSION)
# keep in sync with WRT_TAGS in the repository's Makefile
GO_PKG_TAGS:=with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_grpc,with_awg,tfogo_checklinkname0,with_conntrack,with_embedded_tor,with_openvpn,with_openconnect,with_masque,with_kcp,with_sudoku,with_bond,with_call,with_xdrive,with_v2ray_api,with_dhcp,with_low_memory

include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-package.mk

define Package/hiddify-core
  SECTION:=net
  CATEGORY:=Network
  SUBMENU:=Web Servers/Proxies
  TITLE:=Hiddify universal proxy platform
  URL:=https://hiddify.com/
  DEPENDS:=$(GO_ARCH_DEPENDS) +ca-bundle +kmod-inet-diag +kmod-tun +firewall4 +kmod-nft-queue
endef

define Package/hiddify-core/description
  Hiddify is a universal proxy platform built on sing-box: Hysteria, SOCKS, Shadowsocks,
  ShadowTLS, Tor, Trojan, VLESS, VMess, WireGuard, WARP, Psiphon and more, from a config file
  or a subscription link.
endef

define Package/hiddify-core/conffiles
/etc/config/hiddify-core
/etc/hiddify-core/config.json
endef

define Package/hiddify-core/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/main $(1)/usr/bin/hiddify-core
	$(INSTALL_DIR) $(1)/etc/hiddify-core
	$(INSTALL_CONF) $(PKG_BUILD_DIR)/release/config/config.json $(1)/etc/hiddify-core/config.json
	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/hiddify-core.conf $(1)/etc/config/hiddify-core
	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/hiddify-core.init $(1)/etc/init.d/hiddify-core
	$(INSTALL_DIR) $(1)/lib/upgrade/keep.d
	$(INSTALL_DATA) ./files/hiddify-core.keep $(1)/lib/upgrade/keep.d/hiddify-core
endef

$(eval $(call GoBinPackage,hiddify-core))
$(eval $(call BuildPackage,hiddify-core))
