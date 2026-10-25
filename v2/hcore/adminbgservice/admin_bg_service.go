package adminbgservice

import (
	"context"
	"fmt"
	"log"
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
	"github.com/hiddify/hiddify-core/v2/hcore/adminbg"
)

type hiddifyAdminBG struct {
	UnimplementedHiddifyAdminBGServer
	mutex sync.Mutex
	box   *daemon.StartedService
	// closed when the box or full core stops, to end its watchdog
	watching chan struct{}
	fullCore bool
}

func (s *hiddifyAdminBG) Start(ctx context.Context, in *AdminBgStartRequest) (response *AdminBgResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("start tunnel: %v", recovered)
			response = &AdminBgResponse{Message: err.Error()}
		}
	}()
	if in.GetConfigContent() != "" {
		if err := s.startFullCore(in); err != nil {
			return &AdminBgResponse{Message: err.Error()}, err
		}
		return &AdminBgResponse{Message: "OK"}, nil
	}
	return s.startBridge(in)
}

func (s *hiddifyAdminBG) startBridge(in *AdminBgStartRequest) (*AdminBgResponse, error) {
	s.closeRunning()
	if in.ServerPort == 0 {
		in.ServerPort = 12334
	}
	option := makeTunnelConfig(in)
	box, err := hcore.NewService(libbox.BaseContext(nil), option, nil)
	if err != nil {
		return &AdminBgResponse{Message: err.Error()}, err
	}
	s.box = box
	s.fullCore = false
	s.watching = make(chan struct{})
	go s.watchPort(in.ServerPort, s.watching)
	return &AdminBgResponse{Message: "OK"}, nil
}

func (s *hiddifyAdminBG) startFullCore(in *AdminBgStartRequest) error {
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
	if err := adminbg.Apply(&adminbg.Options{
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
func (s *hiddifyAdminBG) watchPort(port int32, done <-chan struct{}) {
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

// watchOwner stops the VPN engine once the connecting app is gone, but leaves the service
// process listening. Exiting the process would require another administrator prompt to start
// it again; an idle authenticated listener is enough to avoid a background VPN.
func (s *hiddifyAdminBG) watchOwner(pid int32, done <-chan struct{}) {
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

func (s *hiddifyAdminBG) closeRunning() error {
	if s.watching != nil {
		close(s.watching)
		s.watching = nil
	}
	if s.fullCore {
		// Clear fullCore only after StopEngine finishes. Clearing first made Status
		// report "stopped" while the TUN was still tearing down, so a reconnect could
		// race and leave a dead interface (Connected UI, IP unchanged).
		started := time.Now()
		err := adminbg.StopEngine()
		log.Printf("admin_bgservice: StopEngine took %s (err=%v)", time.Since(started), err)
		s.fullCore = false
		return err
	}
	if s.box == nil {
		return nil
	}
	err := s.box.CloseService()
	s.box.Close()
	s.box = nil
	return err
}

func makeTunnelConfig(in *AdminBgStartRequest) option.Options {
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

func (s *hiddifyAdminBG) Stop(ctx context.Context, _ *hcommon.Empty) (*AdminBgResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.box == nil && !s.fullCore {
		return &AdminBgResponse{Message: "Not Started"}, nil
	}
	if err := s.closeRunning(); err != nil {
		return &AdminBgResponse{Message: err.Error()}, err
	}
	return &AdminBgResponse{Message: "OK"}, nil
}

func (s *hiddifyAdminBG) Status(ctx context.Context, _ *hcommon.Empty) (*AdminBgResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.fullCore || s.box != nil {
		return &AdminBgResponse{Message: "started"}, nil
	}
	return &AdminBgResponse{Message: "stopped"}, nil
}

func (s *hiddifyAdminBG) Exit(ctx context.Context, _ *hcommon.Empty) (*AdminBgResponse, error) {
	s.mutex.Lock()
	s.closeRunning()
	s.mutex.Unlock()
	go func() {
		<-time.After(time.Second * 1)
		os.Exit(0)
	}()
	return &AdminBgResponse{Message: "OK"}, nil
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
	return "HiddifyAdminBg"
}
