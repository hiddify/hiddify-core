@echo off
set TAGS=with_gvisor,with_quic,with_wireguard,with_ech,with_utls,with_clash_api,with_grpc
@REM set  TAGS=with_dhcp,with_low_memory,with_conntrack
go run --tags %TAGS% ./cli %*