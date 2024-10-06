package tunnelservice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"

	common "github.com/hiddify/hiddify-core/v2/common"
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
	option := option.Options{}
	err := json.Unmarshal([]byte(makeTunnelConfig(in.Ipv6, in.ServerPort, in.StrictRoute, in.EndpointIndependentNat, in.Stack)), &option)
	if err != nil {
		return &TunnelResponse{
			Message: err.Error(),
		}, err
	}
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

func makeTunnelConfig(Ipv6 bool, ServerPort int32, StrictRoute bool, EndpointIndependentNat bool, Stack string) string {
	var ipv6 string
	if Ipv6 {
		ipv6 = `      "inet6_address": "fdfe:dcba:9876::1/126",`
	} else {
		ipv6 = ""
	}
	base := `{
		"log":{
			"level": "warn"
		},
		"inbounds": [
		  {
			"type": "tun",
			"tag": "tun-in",
			"interface_name": "HiddifyTunnel",
			"inet4_address": "172.19.0.1/30",
			` + ipv6 + `
			"auto_route": true,
			"strict_route": ` + fmt.Sprintf("%t", StrictRoute) + `,
			"endpoint_independent_nat": ` + fmt.Sprintf("%t", EndpointIndependentNat) + `,
			"stack": "` + Stack + `"
		  }
		],
		"outbounds": [
		  {
			"type": "socks",
			"tag": "socks-out",
			"server": "127.0.0.1",
			"server_port": ` + fmt.Sprintf("%d", ServerPort) + `,
			"version": "5"
		  },
		  {
			"type": "direct",
			"tag": "direct-out"
		  }
		],
		"route": {
		  "rules": [
			{
				"process_name":[
					"Hiddify.exe",
					"Hiddify",
					"HiddifyCli",
					"HiddifyCli.exe"
					],
				"outbound": "direct-out"
			}
		  ]
		}
	  }`

	return base
}

func (s *TunnelService) Stop(ctx context.Context, _ *common.Empty) (*TunnelResponse, error) {
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

func (s *TunnelService) Status(ctx context.Context, _ *common.Empty) (*TunnelResponse, error) {
	return &TunnelResponse{
		Message: "Not Implemented",
	}, nil
}

func (s *TunnelService) Exit(ctx context.Context, _ *common.Empty) (*TunnelResponse, error) {
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
