go mod tidy
TAGS=with_v2ray_api,with_gvisor,with_quic,with_dhcp,with_wireguard,with_utls,with_acme,with_clash_api,with_tailscale,with_ccm,with_ocm,tfogo_checklinkname0
# TAGS=with_dhcp,with_low_memory,with_conntrack
go run --tags $TAGS ./cmd/main  $@