package config

import (
    "encoding/base64"
    "fmt"
    "log/slog"
    "net/netip"
    "os"
    "strings"

    "github.com/bepass-org/warp-plus/warp"
    "github.com/hiddify/hiddify-core/v2/common"
    C "github.com/sagernet/sing-box/constant"
    "github.com/sagernet/sing-box/option"
)

func GenerateWarpInfo(license string, oldAccountId string, oldAccessToken string) (*warp.Identity, string, *WarpWireguardConfig, error) {
    if oldAccountId != "" && oldAccessToken != "" {
        _ = warp.DeleteDevice(oldAccessToken, oldAccountId)
    }
    l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
    identity, err := warp.CreateIdentityOnly(l, license)
    res := "Error!"
    var warpcfg WarpWireguardConfig
    if err == nil {
        res = "Success"
        res += fmt.Sprintf("Warp+ enabled: %t\n", identity.Account.WarpPlus)
        res += fmt.Sprintf("\nAccount type: %s\n", identity.Account.AccountType)
        warpcfg = WarpWireguardConfig{
            PrivateKey:       identity.PrivateKey,
            PeerPublicKey:    identity.Config.Peers[0].PublicKey,
            LocalAddressIPv4: identity.Config.Interface.Addresses.V4,
            LocalAddressIPv6: identity.Config.Interface.Addresses.V6,
            ClientID:         identity.Config.ClientID,
        }
    }
    return &identity, res, &warpcfg, err
}

func wireGuardToSingbox(wgConfig WarpWireguardConfig, server string, port uint16) (*option.Outbound, error) {
    clientID, _ := base64.StdEncoding.DecodeString(wgConfig.ClientID)
    if len(clientID) < 3 {
        clientID = []byte{0, 0, 0}
    }
    opt := option.LegacyWireGuardOutboundOptions{
        ServerOptions: option.ServerOptions{Server: server, ServerPort: port},
        PrivateKey:    wgConfig.PrivateKey,
        PeerPublicKey: wgConfig.PeerPublicKey,
        Reserved:      []uint8{clientID[0], clientID[1], clientID[2]},
        MTU:           1330,
    }
    ips := []string{wgConfig.LocalAddressIPv4 + "/24", wgConfig.LocalAddressIPv6 + "/128"}
    for _, addr := range ips {
        if addr == "" {
            continue
        }
        prefix, err := netip.ParsePrefix(addr)
        if err != nil {
            return nil, err
        }
        opt.LocalAddress = append(opt.LocalAddress, prefix)
    }
    out := option.Outbound{Type: C.TypeWireGuard, Tag: "WARP", Options: opt}
    return &out, nil
}

func GenerateWarpSingbox(wgConfig WarpWireguardConfig, host string, port uint16, fakePackets string, fakePacketsSize string, fakePacketsDelay string, fakePacketMode string) (*option.Outbound, error) {
    if host == "" {
        host = "auto4"
    }
    return wireGuardToSingbox(wgConfig, host, port)
}

func patchWarp(base *option.Outbound, configOpt *HiddifyOptions, final bool, staticIpsDns map[string][]string) error {
    if !final || base.Type != C.TypeWireGuard {
        return nil
    }
    lw, ok := base.Options.(option.LegacyWireGuardOutboundOptions)
    if !ok {
        return nil
    }
    host := lw.ServerOptions.Server
    if host == "default" || host == "random" || host == "auto" || host == "auto4" || host == "auto6" || isBlockedDomain(host) {
        rndDomain := strings.ToLower(generateRandomString(20))
        if staticIpsDns != nil {
            staticIpsDns[rndDomain] = []string{}
            if host != "auto4" {
                if host == "auto6" || common.CanConnectIPv6() {
                    ip6, _ := warp.RandomWarpEndpoint(false, true)
                    staticIpsDns[rndDomain] = append(staticIpsDns[rndDomain], ip6.Addr().String())
                }
            }
            if host != "auto6" {
                ip4, _ := warp.RandomWarpEndpoint(true, false)
                staticIpsDns[rndDomain] = append(staticIpsDns[rndDomain], ip4.Addr().String())
            }
        }
        lw.ServerOptions.Server = rndDomain
    }
    if lw.ServerOptions.ServerPort == 0 {
        lw.ServerOptions.ServerPort = warp.RandomWarpPort()
    }
    if lw.DialerOptions.Detour != "" && lw.MTU < 100 {
        lw.MTU = 1280
    }
    base.Options = lw
    return nil
}
