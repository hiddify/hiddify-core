go mod tidy
TAGS=with_v2ray_api,with_gvisor,with_quic,with_dhcp,with_wireguard,with_utls,with_acme,with_clash_api,with_tailscale,with_ccm,with_ocm,tfogo_checklinkname0,with_awg,with_embedded_tor,with_openvpn,with_openconnect,with_naive_outbound
# TAGS=with_dhcp,with_low_memory,with_conntrack
go run --tags $TAGS ./cmd/main  $@