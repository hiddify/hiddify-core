package tunnelservice

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/hcore"
)

type TunnelService struct {
	UnimplementedTunnelServiceServer
	mutex sync.Mutex
	box   *daemon.StartedService
	// closed when the box stops, to end its watchdog
	watching chan struct{}
}

func (s *TunnelService) Start(ctx context.Context, in *TunnelStartRequest) (response *TunnelResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// A configuration the box rejects by panicking must not take the root service down with it.
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("start tunnel: %v", recovered)
			response = &TunnelResponse{Message: err.Error()}
		}
	}()
	s.closeBox()
	if in.ServerPort == 0 {
		in.ServerPort = 12334
	}
	option := makeTunnelConfig(in)

	// The box outlives this call, so it must not take the call's context.
	box, err := hcore.NewService(libbox.BaseContext(nil), option, nil)
	if err != nil {
		return &TunnelResponse{
			Message: err.Error(),
		}, err
	}
	s.box = box
	s.watching = make(chan struct{})
	go s.watch(in.ServerPort, s.watching)

	return &TunnelResponse{
		Message: "OK",
	}, err
}

// watch stops the tunnel once the app's proxy, which all its traffic goes to, stops answering: when
// the app quits or crashes, the system would otherwise be left without a connection.
func (s *TunnelService) watch(port int32, done <-chan struct{}) {
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port)))
	failures := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
		}
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			if failures++; failures < 3 {
				continue
			}
			s.mutex.Lock()
			select {
			case <-done:
			default:
				s.closeBox()
			}
			s.mutex.Unlock()
			return
		}
		conn.Close()
		failures = 0
	}
}

// closeBox stops the running tunnel, if any. The caller holds the mutex.
func (s *TunnelService) closeBox() error {
	if s.watching != nil {
		close(s.watching)
		s.watching = nil
	}
	if s.box == nil {
		return nil
	}
	err := s.box.CloseService()
	// CloseService leaves the box's observer goroutines running; Close ends them.
	s.box.Close()
	s.box = nil
	return err
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
				Options: &option.TunInboundOptions{
					EndpointIndependentNat: in.EndpointIndependentNat,
					StrictRoute:            in.StrictRoute,
					AutoRoute:              true,
					Address:                ips,
					InterfaceName:          tunInterfaceName(),
					Stack:                  in.Stack,
				},
			},
		},
		Outbounds: []option.Outbound{
			{
				Type: C.TypeSOCKS,
				Tag:  "socks-out",
				Options: &option.SOCKSOutboundOptions{
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
				Type:    C.TypeDirect,
				Tag:     "direct-out",
				Options: &option.DirectOutboundOptions{},
			},
		},
		Route: &option.RouteOptions{
			// The tunnel's own connections, and the app's, go out past the TUN: bound to the
			// physical interface, or to another VPN's when that is how the system reaches the internet.
			AutoDetectInterface: true,
			DefaultInterface:    routedVPNInterface(),
			Final:               "socks-out",
			Rules: []option.Rule{
				{
					Type: C.RuleTypeDefault,
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
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.box == nil {
		return &TunnelResponse{
			Message: "Not Started",
		}, nil
	}
	if err := s.closeBox(); err != nil {
		return &TunnelResponse{
			Message: err.Error(),
		}, err
	}

	return &TunnelResponse{
		Message: "OK",
	}, nil
}

func (s *TunnelService) Status(ctx context.Context, _ *hcommon.Empty) (*TunnelResponse, error) {
	return &TunnelResponse{
		Message: "Not Implemented",
	}, nil
}

func (s *TunnelService) Exit(ctx context.Context, _ *hcommon.Empty) (*TunnelResponse, error) {
	s.mutex.Lock()
	s.closeBox()
	s.mutex.Unlock()
	go func() {
		<-time.After(time.Second * 1)
		os.Exit(0)
	}()
	return &TunnelResponse{
		Message: "OK",
	}, nil
}

// routedVPNInterface returns, on macOS, the utun interface of another VPN the system sends public
// traffic through, so that the tunnel goes out through it instead of past it; "" otherwise.
func routedVPNInterface() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	output, err := exec.Command("/sbin/route", "-n", "get", "-inet", "1.1.1.1").Output()
	if err != nil {
		return ""
	}
	return vpnInterface(string(output))
}

func vpnInterface(routeOutput string) string {
	for _, line := range strings.Split(routeOutput, "\n") {
		if name, found := strings.CutPrefix(strings.TrimSpace(line), "interface:"); found {
			if name = strings.TrimSpace(name); strings.HasPrefix(name, "utun") {
				return name
			}
			return ""
		}
	}
	return ""
}

// tunInterfaceName names the TUN, except on macOS, which allows only utun<n>: there sing-box takes the
// next free one.
func tunInterfaceName() string {
	if runtime.GOOS == "darwin" {
		return ""
	}
	return "HiddifyTunnel"
}
