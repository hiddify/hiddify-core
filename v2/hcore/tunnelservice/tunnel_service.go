package tunnelservice

import (
	"context"
	"net/netip"
	"os"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/option"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/hcore"
)

type TunnelService struct {
	UnimplementedTunnelServiceServer
	box *daemon.StartedService
}

func (s *TunnelService) Start(ctx context.Context, in *TunnelStartRequest) (*TunnelResponse, error) {
	if in.ServerPort == 0 {
		in.ServerPort = 12334
	}
	option := makeTunnelConfig(in)

	box, err := hcore.NewService(ctx, option)
	s.box = box
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
	ips := []netip.Prefix{netip.MustParsePrefix("172.20.0.1/30")}
	if in.Ipv6 {
		ips = append(ips, netip.MustParsePrefix("fdfe:dcba:9876::1/126"))
	}
	return option.Options{
		Log: &option.LogOptions{Level: "warn"},
		Inbounds: []option.Inbound{
			{
				Type: C.TypeTun,
				Tag:  "tun-in",
				Options: option.TunInboundOptions{
					EndpointIndependentNat: in.EndpointIndependentNat,
					StrictRoute:            in.StrictRoute,
					AutoRoute:              true,
					Address:                ips,
					InterfaceName:          "HiddifyTunnel",
					Stack:                  in.Stack,
				},
			},
		},
		Outbounds: []option.Outbound{
			{
				Type: C.TypeSOCKS,
				Tag:  "socks-out",
				Options: option.SOCKSOutboundOptions{
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
						RawDefaultRule: option.RawDefaultRule{
							ProcessName: []string{
								"Hiddify.exe",
								"Hiddify",
								"HiddifyCli",
								"HiddifyCli.exe",
							},
						},
						RuleAction: option.RuleAction{
							Action: C.RuleActionTypeDirect,
							RouteOptions: option.RouteActionOptions{
								Outbound: "direct-out",
							},
						},
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
	err := s.box.CloseService()
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
		s.box.CloseService()
	}
	go func() {
		<-time.After(time.Second * 1)
		os.Exit(0)
	}()
	return &TunnelResponse{
		Message: "OK",
	}, nil
}
