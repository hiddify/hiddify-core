include $(TOPDIR)/rules.mk

PKG_NAME:=hiddify-core
PKG_VERSION:=3.1.7
PKG_RELEASE:=1

PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz
PKG_SOURCE_URL:=https://codeload.github.com/hiddify/hiddify-core/tar.gz/v$(PKG_VERSION)?
PKG_HASH:=e45f258a1b2fff13ede4b19493a2dc0ed4fd3e34887a3a9f054b6bbcbdf40e65

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
  USERID:=hiddify=15566:hiddify=15566
endef

define Package/hiddify-core/description
  Hiddify is a universal proxy platform which supports hysteria, SOCKS, Shadowsocks,
  ShadowTLS, Tor, trojan, VLess, VMess, WireGuard and so on.
endef

GO_PKG_TAGS:=with_gvisor,with_quic,with_wireguard,with_ech,with_utls,with_clash_api,with_grpc,with_low_memory,with_conntrack

define Package/hiddify-core/conffiles
/etc/config/hiddify
/etc/hiddify/
endef

define Package/hiddify-core/install

	$(INSTALL_DIR) $(1)/usr/bin/
	$(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/cli $(1)/usr/bin/hiddify-cli

	$(INSTALL_DIR) $(1)/etc/hiddify
	$(INSTALL_DATA) ./files/hiddify.json $(1)/etc/hiddify/appconfig.json

	$(INSTALL_DIR) $(1)/etc/config/
	$(INSTALL_CONF) ./files/hiddify.conf $(1)/etc/config/hiddify

	$(INSTALL_DIR) $(1)/etc/init.d/
	$(INSTALL_BIN) ./files/hiddify.init $(1)/etc/init.d/hiddify

	$(CP) ./files/webui $(1)/usr/bin/
endef

$(eval $(call BuildPackage,hiddify-core))
