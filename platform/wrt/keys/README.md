# Feed signing keys

The public halves of the keys that sign the OpenWrt package feeds. A router installs them once
and then verifies every index it fetches. The private halves live only in the repository's
GitHub secrets, `OPENWRT_USIGN_KEY` and `OPENWRT_APK_KEY`; never commit them.

| File | For | Made with |
|---|---|---|
| `hiddify.pub` | opkg (OpenWrt 24.10 and earlier), `Packages.sig` | `usign -G -c "Hiddify OpenWrt feed" -s hiddify.sec -p hiddify.pub` |
| `hiddify-apk.pem` | apk (OpenWrt 25.12 and later), `packages.adb` | `openssl ecparam -name prime256v1 -genkey -noout -out hiddify-apk.key` then `openssl ec -in hiddify-apk.key -pubout -out hiddify-apk.pem` |

Store `hiddify.sec` as the secret `OPENWRT_USIGN_KEY` and `hiddify-apk.key` as `OPENWRT_APK_KEY`,
whole file contents each. Until both are set, the feed job still publishes, with a warning, and
the indexes are unsigned: routers then need `option check_signature 0` (opkg) or
`--allow-untrusted` (apk). The apk key must be an EC key in SEC1 PEM form, as above; a PKCS#8
key from `openssl genpkey` is refused by `apk mkndx`.
