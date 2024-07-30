package mobile

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hiddify/hiddify-core/config"
	"github.com/hiddify/hiddify-core/mobile/warp"
	"github.com/hiddify/hiddify-core/mobile/ws"
	"github.com/sagernet/sing-box/option"
)

func WarpSetupFree() error {
	// make primary identity
	license := "notset"
	_license := ""
	warp.UpdatePath("./warp-primary")
	if !warp.CheckProfileExists(license) {
		err := warp.LoadOrCreateIdentity(_license)
		if err != nil {
			log.Printf("error: %v", err)
			return fmt.Errorf("error: %v", err)
		}
	}
	// make secondary identity
	warp.UpdatePath("./warp-secondary")
	if !warp.CheckProfileExists(license) {
		err := warp.LoadOrCreateIdentity(_license)
		if err != nil {
			log.Printf("error: %v", err)
			return fmt.Errorf("error: %v", err)
		}
	}
	return nil
}

func convertConfig(device *ws.DeviceConfig) (*option.WireGuardOutboundOptions, error) {
	peers := []option.WireGuardPeer{}
	for _, peer := range device.Peers {
		address, port, found := strings.Cut(*peer.Endpoint, ":")
		if !found {
			return nil, fmt.Errorf("endpoint has no port")
		}
		portUInt64, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return nil, err
		}
		ips := []string{}
		for _, allowedIP := range peer.AllowedIPs {
			ips = append(ips, allowedIP.String())
		}
		peers = append(peers, option.WireGuardPeer{
			AllowedIPs:   ips,
			PublicKey:    peer.PublicKey,
			PreSharedKey: peer.PreSharedKey,
			ServerOptions: option.ServerOptions{
				Server:     address,
				ServerPort: uint16(portUInt64),
			},
		})
	}

	return &option.WireGuardOutboundOptions{
		MTU:          uint32(device.MTU),
		Peers:        peers,
		PrivateKey:   device.SecretKey,
		LocalAddress: device.Endpoint,
	}, nil
}

func WarpGetOutbounds(tag string, endpoint string, nested bool) (string, error) {
	options := []option.Outbound{}
	primaryTag := tag
	if nested {
		primaryTag = "primary"
		conf, err := ws.ParseConfig("./warp-secondary/wgcf-profile.ini", endpoint)
		if err != nil {
			return "", err
		}
		wgOptions, err := convertConfig(conf.Device)
		if err != nil {
			return "", err
		}
		wgOptions.Detour = primaryTag
		options = append(options, option.Outbound{
			Type:             "wireguard",
			Tag:              tag,
			WireGuardOptions: *wgOptions,
		})
	}
	conf, err := ws.ParseConfig("./warp-primary/wgcf-profile.ini", endpoint)
	if err != nil {
		return "", err
	}
	wgOptions, err := convertConfig(conf.Device)
	if err != nil {
		return "", err
	}
	options = append(options, option.Outbound{
		Type:             "wireguard",
		Tag:              primaryTag,
		WireGuardOptions: *wgOptions,
	})
	jsonData, err := json.Marshal(options)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func toLink(config *ws.Configuration) (string, error) {
	conf, err := convertConfig(config.Device)
	if err != nil {
		return "", err
	}
	endpoint := conf.Peers[0].ServerOptions.Server + ":" + strconv.Itoa(int(conf.Peers[0].ServerOptions.ServerPort))
	pk := conf.PrivateKey
	pub := conf.Peers[0].PublicKey
	psk := conf.Peers[0].PreSharedKey
	mtu := strconv.Itoa(int(conf.MTU))
	reserved := []string{}
	for _, r := range conf.Reserved {
		reserved = append(reserved, strconv.Itoa(int(r)))
	}
	ip := []string{}
	for _, a := range conf.LocalAddress {
		ip = append(ip, a.String())
	}
	return "wg://" + endpoint + "?pk=" + pk + "&pub=" + pub + "&psk=" + psk + "&mtu=" + mtu + "&reserved=" + strings.Join(reserved, ",") + "&ip=" + strings.Join(ip, ","), nil
}

func formatOpts(opts string) string {
	if len(opts) == 0 {
		return ""
	}
	return "&" + opts
}

func WarpGetOutboundsAlt(tag string, endpoint string, nested bool, optsPrimary string, optsSecondary string) (string, error) {
	confSecondary, err := ws.ParseConfig("./warp-secondary/wgcf-profile.ini", endpoint)
	if err != nil {
		return "", err
	}
	confSecondaryString, err := toLink(confSecondary)
	if err != nil {
		return "", err
	}
	confPrimary, err := ws.ParseConfig("./warp-primary/wgcf-profile.ini", endpoint)
	if err != nil {
		return "", err
	}
	confPrimaryString, err := toLink(confPrimary)
	if err != nil {
		return "", err
	}
	configStr := ""
	if nested {
		configStr = confPrimaryString + formatOpts(optsPrimary) + "#proxy&&detour=" + confSecondaryString + formatOpts(optsSecondary) + "#secondary"
	} else {
		configStr = confPrimaryString + formatOpts(optsPrimary) + "#proxy"
	}
	config, err := config.ParseConfigContent(configStr, false, nil, false)
	if err != nil {
		return "", err
	}
	return string(config[:]), nil
}
