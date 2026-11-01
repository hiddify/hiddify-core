#!/usr/bin/env bash
# Turns a directory of hiddify-core packages for one OpenWrt architecture into a package feed,
# as OpenWrt's own package/Makefile does:
#   opkg (24.10 and earlier): Packages, Packages.gz and, with a usign key, Packages.sig
#   apk  (25.12 and later):   packages.adb, signed with the apk key when given (apk refuses an
#                             unsigned index unless the user allows untrusted packages)
# plus sha256sums of everything in the directory.
#
#   release/openwrt/make-feed.sh feed/x86_64 [usign private key] [apk private key (EC PEM)]

set -e -o pipefail

DIR="$1"; USIGN_KEY="${2:-}"; APK_KEY="${3:-}"
[ -d "$DIR" ] || { echo "Usage: $0 <feed directory> [usign key] [apk key]" >&2; exit 1; }
cd "$DIR"

sha256() {
  if command -v sha256sum >/dev/null; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi
}
size() { if stat -c%s "$1" >/dev/null 2>&1; then stat -c%s "$1"; else stat -f%z "$1"; fi; }

# --- opkg: one control block per package, with where and how big it is
rm -f Packages Packages.gz Packages.sig
for ipk in $(ls *.ipk 2>/dev/null | sort); do
  tar -xzOf "$ipk" ./control.tar.gz | tar -xzOf - ./control | sed -e "s|^Description:|Filename: $ipk\\
Size: $(size "$ipk")\\
SHA256sum: $(sha256 "$ipk")\\
Description:|"
  echo ""
done > Packages
# the usign of older firmwares hashes a file of these sizes wrong; OpenWrt pads them too
case "$(( (64 + $(size Packages)) % 128 ))" in
  110|111) echo "padding Packages for the usign SHA-512 bug" >&2; { echo ""; echo ""; } >> Packages ;;
esac
gzip -9nc Packages > Packages.gz
if [ -n "$USIGN_KEY" ]; then
  command -v usign >/dev/null || { echo "usign is not on the PATH" >&2; exit 1; }
  usign -S -m Packages -s "$USIGN_KEY"
fi

# --- apk: a signed binary index of the .apk files
rm -f packages.adb
if ls *.apk >/dev/null 2>&1; then
  command -v apk >/dev/null || { echo "apk (apk-tools 3) is not on the PATH" >&2; exit 1; }
  sign=()
  [ -n "$APK_KEY" ] && sign=(--sign "$APK_KEY")
  apk mkndx --root . --keys-dir . --allow-untrusted "${sign[@]}" --output packages.adb *.apk
fi

rm -f sha256sums
for f in $(ls | sort); do
  [ "$f" = sha256sums ] && continue
  echo "$(sha256 "$f") *$f"
done > sha256sums
echo "feed ready in $DIR: $(ls *.ipk 2>/dev/null | wc -l | tr -d ' ') ipk, $(ls *.apk 2>/dev/null | wc -l | tr -d ' ') apk" >&2
