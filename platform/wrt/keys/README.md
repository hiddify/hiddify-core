# Feed signing keys

The public halves of the keys that sign an OpenWrt package feed built with
`release/openwrt/make-feed.sh`. The releases publish the packages alone, with no feed; these keys are
for a feed hosted separately. A router installs them once and then verifies every index it fetches.
Never commit the private halves.

| File | For | Made with |
|---|---|---|
| `hiddify.pub` | opkg (OpenWrt 24.10 and earlier), `Packages.sig` | `usign -G -c "Hiddify OpenWrt feed" -s hiddify.sec -p hiddify.pub` |
| `hiddify-apk.pem` | apk (OpenWrt 25.12 and later), `packages.adb` | `openssl ecparam -name prime256v1 -genkey -noout -out hiddify-apk.key` then `openssl ec -in hiddify-apk.key -pubout -out hiddify-apk.pem` |

Pass the private keys to `make-feed.sh`. Without them the indexes are unsigned: routers then need
`option check_signature 0` (opkg) or `--allow-untrusted` (apk). The apk key must be an EC key in
SEC1 PEM form, as above; a PKCS#8 key from `openssl genpkey` is refused by `apk mkndx`.
