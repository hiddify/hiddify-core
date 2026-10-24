package tunnelservice

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	"github.com/hiddify/hiddify-core/v2/hcore/tunnel"
)

type TunnelService struct {
	UnimplementedTunnelServiceServer
	mutex sync.Mutex
	box   *daemon.StartedService
	// closed when the box or full core stops, to end its watchdog
	watching chan struct{}
	fullCore bool
}

func (s *TunnelService) Start(ctx context.Context, in *TunnelStartRequest) (response *TunnelResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("start tunnel: %v", recovered)
			response = &TunnelResponse{Message: err.Error()}
		}
	}()
	if in.GetConfigContent() != "" {
		if err := s.startFullCore(in); err != nil {
			return &TunnelResponse{Message: err.Error()}, err
		}
		return &TunnelResponse{Message: "OK"}, nil
	}
	return s.startBridge(in)
}

func (s *TunnelService) startBridge(in *TunnelStartRequest) (*TunnelResponse, error) {
	s.closeRunning()
	if in.ServerPort == 0 {
		in.ServerPort = 12334
	}
	option := makeTunnelConfig(in)
	box, err := hcore.NewService(libbox.BaseContext(nil), option, nil)
	if err != nil {
		return &TunnelResponse{Message: err.Error()}, err
	}
	s.box = box
	s.fullCore = false
	s.watching = make(chan struct{})
	go s.watchPort(in.ServerPort, s.watching)
	return &TunnelResponse{Message: "OK"}, nil
}

func (s *TunnelService) startFullCore(in *TunnelStartRequest) error {
	s.closeRunning()
	dataDirectory := serviceDataDirectory()
	base := filepath.Join(dataDirectory, "core")
	work := filepath.Join(base, "work")
	temp := filepath.Join(base, "temp")
	for _, directory := range []string{base, work, temp, filepath.Join(work, "data")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
	}
	if err := writeResources(filepath.Join(work, "resources"), in.GetResources()); err != nil {
		return err
	}
	secret, err := LoadControlSecret()
	if err != nil {
		return err
	}
	settings := in.GetSettingsJson()
	if settings == "" {
		settings = "{}"
	}
	if err := tunnel.Apply(&tunnel.Options{
		BasePath:           base,
		WorkingPath:        work,
		TempPath:           temp,
		ConfigContent:      in.GetConfigContent(),
		ConfigName:         in.GetConfigName(),
		SettingsJSON:       settings,
		Secret:             secret,
		DisableMemoryLimit: in.GetDisableMemoryLimit(),
		NativeTun:          true,
	}, nil); err != nil {
		return err
	}
	s.fullCore = true
	s.watching = make(chan struct{})
	go s.watchOwner(in.GetOwnerPid(), s.watching)
	return nil
}

var resourceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func writeResources(directory string, resources map[string][]byte) error {
	if err := os.RemoveAll(directory); err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	var total int
	for name, bytes := range resources {
		if !resourceNamePattern.MatchString(name) {
			return fmt.Errorf("invalid saved VPN resource name")
		}
		total += len(bytes)
		if total > 32*1024*1024 {
			return fmt.Errorf("saved VPN resources exceed the 32 MB limit")
		}
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, bytes, 0o600); err != nil {
			return err
		}
	}
	return nil
}

// watchPort stops the TUN bridge once the app's proxy stops answering.
func (s *TunnelService) watchPort(port int32, done <-chan struct{}) {
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
				s.closeRunning()
			}
			s.mutex.Unlock()
			return
		}
		conn.Close()
		failures = 0
	}
}

// watchOwner stops the full core and exits the service once the connecting app is gone.
func (s *TunnelService) watchOwner(pid int32, done <-chan struct{}) {
	if pid <= 0 {
		return
	}
	failures := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
		}
		if processAlive(pid) {
			failures = 0
			continue
		}
		if failures++; failures < 2 {
			continue
		}
		s.mutex.Lock()
		select {
		case <-done:
		default:
			s.closeRunning()
		}
		s.mutex.Unlock()
		go func() {
			time.Sleep(time.Second)
			os.Exit(0)
		}()
		return
	}
}

func processAlive(pid int32) bool {
	process, err := os.FindProcess(int(pid))
	if err != nil {
		return false
	}
	return processSignalAlive(process)
}

func (s *TunnelService) closeRunning() error {
	if s.watching != nil {
		close(s.watching)
		s.watching = nil
	}
	if s.fullCore {
		s.fullCore = false
		return tunnel.StopEngine()
	}
	if s.box == nil {
		return nil
	}
	err := s.box.CloseService()
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
	if s.box == nil && !s.fullCore {
		return &TunnelResponse{Message: "Not Started"}, nil
	}
	if err := s.closeRunning(); err != nil {
		return &TunnelResponse{Message: err.Error()}, err
	}
	return &TunnelResponse{Message: "OK"}, nil
}

func (s *TunnelService) Status(ctx context.Context, _ *hcommon.Empty) (*TunnelResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.fullCore || s.box != nil {
		return &TunnelResponse{Message: "started"}, nil
	}
	return &TunnelResponse{Message: "stopped"}, nil
}

func (s *TunnelService) Exit(ctx context.Context, _ *hcommon.Empty) (*TunnelResponse, error) {
	s.mutex.Lock()
	s.closeRunning()
	s.mutex.Unlock()
	go func() {
		<-time.After(time.Second * 1)
		os.Exit(0)
	}()
	return &TunnelResponse{Message: "OK"}, nil
}

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

func tunInterfaceName() string {
	if runtime.GOOS == "darwin" {
		return ""
	}
	return "HiddifyTunnel"
}
