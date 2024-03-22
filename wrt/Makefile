include $(TOPDIR)/rules.mk

PKG_NAME:=hiddify-core
PKG_VERSION:=1.2.0
PKG_RELEASE:=1

PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz
PKG_SOURCE_URL:=https://codeload.github.com/hiddify/hiddify-core/tar.gz/v$(PKG_VERSION)?
PKG_HASH:=481098a29b0b08d225598e5e74eddfe4cbdd4560b7ac74d210dcc923792a3e24

PKG_LICENSE:=https://github.com/hiddify/hiddify-next/edit/main/LICENSE.md
PKG_LICENSE_FILES:=LICENSE
PKG_MAINTAINER:=Hiddify <wrt@hiddify.com>

PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_BUILD_FLAGS:=no-mips16

GO_PKG:=github.com/hiddify/hiddify-core
GO_PKG_BUILD_PKG:=$(GO_PKG)/cli

GO_PKG_LDFLAGS_X:=$(GO_PKG)/constant.Version=$(PKG_VERSION)

include $(INCLUDE_DIR)/package.mk
include ../../lang/golang/golang-package.mk

define Package/hiddify-core
  TITLE:=Hiddify universal proxy platform
  SECTION:=net
  CATEGORY:=Network
  URL:=https://hiddify.com/
  DEPENDS:=$(GO_ARCH_DEPENDS) +ca-bundle +kmod-inet-diag +kmod-tun
  USERID:=hiddify-core=5566:hiddify-core=5566
endef

define Package/hiddify-core/description
  Hiddify is a universal proxy platform which supports hysteria, SOCKS, Shadowsocks,
  ShadowTLS, Tor, trojan, VLess, VMess, WireGuard and so on.
endef

GO_PKG_TAGS:=with_gvisor,with_quic,with_wireguard,with_ech,with_utls,with_clash_api,with_grpc,with_low_memory,with_conntrack

define Package/hiddify-core/conffiles
/etc/config/hiddify-core
/etc/hiddify-core/
endef

define Package/hiddify-core/install

	$(INSTALL_DIR) $(1)/usr/bin/
	$(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/HiddifyCli $(1)/usr/bin/hiddify-cli
	$(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/HiddifyCli $(1)/usr/bin/HiddifyCli

	$(INSTALL_DIR) $(1)/etc/hiddify-core
	$(INSTALL_DATA) $(PKG_BUILD_DIR)/release/config/config.json $(1)/etc/hiddify-core

	$(INSTALL_DIR) $(1)/etc/config/
	$(INSTALL_CONF) ./files/hiddify.conf $(1)/etc/config/hiddify-core
	$(INSTALL_DIR) $(1)/etc/init.d/
	$(INSTALL_BIN) ./files/hiddify.init $(1)/etc/init.d/hiddify-core
endef

$(eval $(call BuildPackage,hiddify-core))
