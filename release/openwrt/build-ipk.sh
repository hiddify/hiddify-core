#!/usr/bin/env bash
# Builds the OpenWrt .ipk (opkg, OpenWrt 24.10 and earlier) of hiddify-core for one OpenWrt
# architecture, from a cross-built binary. An .ipk is a gzipped tar of debian-binary,
# control.tar.gz and data.tar.gz; it is written here directly, with the control fields opkg and
# the feed index read, instead of converting a Debian package.
#
#   release/openwrt/build-ipk.sh x86_64 5.3.0 dist/hiddify-core dist/hiddify-core_5.3.0_x86_64.ipk

. "$(dirname "$0")/common.sh"

ARCHITECTURE="$1"; VERSION="$2"; BINARY_PATH="$3"; OUTPUT_PATH="$4"
[ -n "$ARCHITECTURE" ] && [ -n "$VERSION" ] && [ -f "$BINARY_PATH" ] && [ -n "$OUTPUT_PATH" ] || usage

# opkg versions: 5.3.0-r1; a pre-release keeps its tag text, which sorts before the release
IPK_VERSION="${VERSION#v}-r1"

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$WORK/data" "$WORK/control"
stage_payload "$WORK/data"

installed_size=$(du -sk "$WORK/data" | cut -f1)
cat > "$WORK/control/control" <<CONTROL
Package: $PKG_NAME
Version: $IPK_VERSION
Depends: ${PKG_DEPENDS// /, }
Source: https://github.com/hiddify/hiddify-core
SourceName: $PKG_NAME
License: $PKG_LICENSE
Section: $PKG_SECTION
Architecture: $ARCHITECTURE
Installed-Size: $((installed_size * 1024))
Maintainer: $PKG_MAINTAINER
Description: $PKG_DESCRIPTION
CONTROL
printf '%s\n' $PKG_CONFFILES > "$WORK/control/conffiles"
install -m755 "$WRT_FILES/hiddify-core.postinst" "$WORK/control/postinst"
install -m755 "$WRT_FILES/hiddify-core.prerm" "$WORK/control/prerm"

tar_root "$WORK/control" "$WORK/control.tar.gz"
tar_root "$WORK/data" "$WORK/data.tar.gz"
echo "2.0" > "$WORK/debian-binary"
mkdir -p "$(dirname "$OUTPUT_PATH")"
( cd "$WORK" && if tar --version 2>/dev/null | grep -q GNU; then
    tar --owner=0 --group=0 --numeric-owner --mtime="@${SOURCE_DATE_EPOCH:-0}" -cf - ./debian-binary ./control.tar.gz ./data.tar.gz
  else
    tar --uid 0 --gid 0 --uname root --gname root -cf - ./debian-binary ./control.tar.gz ./data.tar.gz
  fi ) | gzip -9n > "$OUTPUT_PATH"
echo "built $OUTPUT_PATH ($PKG_NAME $IPK_VERSION, $ARCHITECTURE)" >&2
