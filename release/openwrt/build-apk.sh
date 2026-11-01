#!/usr/bin/env bash
# Builds the OpenWrt .apk (apk, OpenWrt 25.12 and later) of hiddify-core for one OpenWrt
# architecture, from a cross-built binary, with `apk mkpkg` as OpenWrt's own build does. Needs the
# static apk of apk-tools 3 on the PATH (CI takes it from the alpine:edge image).
#
#   release/openwrt/build-apk.sh x86_64 5.3.0 dist/hiddify-core dist/
#
# writes dist/hiddify-core-5.3.0-r0_x86_64.apk: apk fetches a package from a feed as
# <name>-<version>.apk, so the feed keeps the part before "_"; the architecture after it tells
# the packages of one release apart.
#
# APK_SIGN_KEY, when set, is the feed's private key: each package is signed too, as OpenWrt's
# CONFIG_SIGN_EACH_PACKAGE does; the index is signed in make-feed.sh either way.

. "$(dirname "$0")/common.sh"

ARCHITECTURE="$1"; VERSION="$2"; BINARY_PATH="$3"; OUTPUT_DIR="$4"
[ -n "$ARCHITECTURE" ] && [ -n "$VERSION" ] && [ -f "$BINARY_PATH" ] && [ -n "$OUTPUT_DIR" ] || usage
command -v apk >/dev/null || { echo "apk (apk-tools 3) is not on the PATH" >&2; exit 1; }

# apk versions: 5.3.0-r0; 5.3.0-beta.2 -> 5.3.0_beta2-r0, 5.3.0.dev -> 5.3.0_pre0-r0
APK_VERSION="${VERSION#v}"
APK_VERSION=$(echo "$APK_VERSION" | sed -E 's/-([a-z]+)\.?([0-9]+)/_\1\2/; s/\.dev$/_pre0/')
APK_VERSION="${APK_VERSION}-r0"
OUTPUT_PATH="$OUTPUT_DIR/${PKG_NAME}-${APK_VERSION}_${ARCHITECTURE}.apk"

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
ROOT="$WORK/root"
stage_payload "$ROOT"

# apk mkpkg resolves file owners through --root's passwd and group
mkdir -p "$WORK/apkroot/etc"
printf 'root:x:%s:%s:root:/root:/sbin/nologin\n' "$(id -u)" "$(id -g)" > "$WORK/apkroot/etc/passwd"
printf 'root:x:%s:root\n' "$(id -g)" > "$WORK/apkroot/etc/group"

# the package's own metadata, as OpenWrt's package-pack.mk writes it
PACKAGES_DIR="$ROOT/lib/apk/packages"
mkdir -p "$PACKAGES_DIR"
printf '%s\n' $PKG_CONFFILES > "$PACKAGES_DIR/.conffiles"
while IFS= read -r conffile; do
  echo "$conffile $(sha256 "$ROOT$conffile")"
done < "$PACKAGES_DIR/.conffiles" > "$PACKAGES_DIR/.conffiles_static"
( cd "$ROOT" && find . -type f -o -type l ) | sed 's|^\./|/|' | grep -v '^/lib/apk/packages/' | sort > "$PACKAGES_DIR/.list"

sign=()
[ -n "${APK_SIGN_KEY:-}" ] && sign=(--sign "$APK_SIGN_KEY")
mkdir -p "$(dirname "$OUTPUT_PATH")"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-0}" apk --root "$WORK/apkroot" mkpkg \
  --info "name:$PKG_NAME" \
  --info "version:$APK_VERSION" \
  --info "description:$PKG_DESCRIPTION" \
  --info "arch:$ARCHITECTURE" \
  --info "license:$PKG_LICENSE" \
  --info "origin:$PKG_NAME" \
  --info "url:$PKG_URL" \
  --info "maintainer:$PKG_MAINTAINER" \
  --info "depends:$PKG_DEPENDS" \
  --script "post-install:$WRT_FILES/hiddify-core.postinst" \
  --script "pre-deinstall:$WRT_FILES/hiddify-core.prerm" \
  --files "$ROOT" \
  "${sign[@]}" \
  --output "$OUTPUT_PATH"
echo "built $OUTPUT_PATH ($PKG_NAME $APK_VERSION, $ARCHITECTURE)" >&2
