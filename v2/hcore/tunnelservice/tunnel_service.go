package tunnelservice

import (
	"context"
	"net/netip"
	"os"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/hcore"
)

type TunnelService struct {
	UnimplementedTunnelServiceServer
	box *libbox.BoxService
}

func (s *TunnelService) Start(ctx context.Context, in *TunnelStartRequest) (*TunnelResponse, error) {
	if in.ServerPort == 0 {
		in.ServerPort = 12334
	}
	option := makeTunnelConfig(in)

	instance, err := hcore.NewService(option)
	if err != nil {
		return &TunnelResponse{
			Message: err.Error(),
		}, err
	}
	err = instance.Start()
	if err != nil {
		return &TunnelResponse{
			Message: err.Error(),
		}, err
	}

	return &TunnelResponse{
		Message: "OK",
	}, err
}

func makeTunnelConfig(in *TunnelStartRequest) option.Options {
	ipv6 := make([]netip.Prefix, 0)
	if in.Ipv6 {
		ipv6 = append(ipv6, netip.MustParsePrefix("fdfe:dcba:9876::1/126"))
	}
	return option.Options{
		Log: &option.LogOptions{Level: "warn"},
		Inbounds: []option.Inbound{
			{
				Type: C.TypeTun,
				Tag:  "tun-in",
				TunOptions: option.TunInboundOptions{
					EndpointIndependentNat: in.EndpointIndependentNat,
					StrictRoute:            in.StrictRoute,
					AutoRoute:              true,
					Inet4Address:           []netip.Prefix{netip.MustParsePrefix("172.20.0.1/30")},
					Inet6Address:           ipv6,
					InterfaceName:          "HiddifyTunnel",
					Stack:                  in.Stack,
				},
			},
		},
		Outbounds: []option.Outbound{
			{
				Type: C.TypeSOCKS,
				Tag:  "socks-out",
				SocksOptions: option.SocksOutboundOptions{
					ServerOptions: option.ServerOptions{
						Server:     "127.0.0.1",
						ServerPort: uint16(in.ServerPort),
					},
					Username: in.ServerUsername,
					Password: in.ServerPassword,
					Version:  "5",
				},
			},
			{
				Type: C.TypeDirect,
				Tag:  "direct-out",
			},
		},
		Route: &option.RouteOptions{
			Final: "socks-out",
			Rules: []option.Rule{
				{
					DefaultOptions: option.DefaultRule{
						ProcessName: []string{
							"Hiddify.exe",
							"Hiddify",
							"HiddifyCli",
							"HiddifyCli.exe",
						},
						Outbound: "direct-out",
					},
				},
			},
		},
	}
}

func (s *TunnelService) Stop(ctx context.Context, _ *hcommon.Empty) (*TunnelResponse, error) {
	if s.box == nil {
		return &TunnelResponse{
			Message: "Not Started",
		}, nil
	}
	err := s.box.Close()
	if err != nil {
		return &TunnelResponse{
			Message: err.Error(),
		}, err
	}

	return &TunnelResponse{
		Message: "OK",
	}, err
}

func (s *TunnelService) Status(ctx context.Context, _ *hcommon.Empty) (*TunnelResponse, error) {
	return &TunnelResponse{
		Message: "Not Implemented",
	}, nil
}

func (s *TunnelService) Exit(ctx context.Context, _ *hcommon.Empty) (*TunnelResponse, error) {
	if s.box != nil {
		s.box.Close()
	}
	go func() {
		<-time.After(time.Second * 1)
		os.Exit(0)
	}()
	return &TunnelResponse{
		Message: "OK",
	}, nil
}
