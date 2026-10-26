// Package adminbg hosts the core for privileged macOS VPN modes: the packet-tunnel
// system extension and the root launchd admin_bgservice. Lifecycle for the extension
// goes through NetworkExtension; for the service, through adminbgservice.StartAdminBgService.
// Both expose the same authenticated loopback APIs for statistics and proxy selection.
package adminbg

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hiddify/hiddify-core/v2/hcore"
	"github.com/hiddify/hiddify-core/v2/hello"
	_ "github.com/sagernet/gomobile"
	"github.com/sagernet/sing-box/experimental/libbox"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const SecretHeader = "x-hiddify-secret"
const MinSecretLength = 32

type Options struct {
	BasePath           string
	WorkingPath        string
	TempPath           string
	ConfigContent      string
	ConfigName         string
	SettingsJSON       string
	Secret             string
	Port               int32
	DisableMemoryLimit bool
	// ProxyOnly runs the core without its TUN inbound: the extension only points the system proxy
	// at the core's local proxy, for the Mac App Store build, whose sandbox cannot set it itself.
	ProxyOnly bool
	// NativeTun lets the root admin_bgservice open the TUN itself. The Network Extension must pass
	// a platform interface instead, so macOS can install its packet-tunnel settings.
	NativeTun bool
}

var lifecycle sync.Mutex
var server *grpc.Server
var engineRunning bool

func Authenticate(ctx context.Context, secret string) error {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get(SecretHeader)
	if len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), []byte(secret)) != 1 {
		return status.Error(codes.Unauthenticated, "invalid tunnel credentials")
	}
	return nil
}

// AllowCoreMethod exposes only observability and proxy selection. Lifecycle, file paths and
// system proxy settings must never reach the privileged engine over these methods.
func AllowCoreMethod(method string) error {
	switch method {
	case "/hello.Hello/SayHello", hcore.Core_CoreInfoListener_FullMethodName,
		hcore.Core_OutboundsInfo_FullMethodName, hcore.Core_MainOutboundsInfo_FullMethodName,
		hcore.Core_GetSystemInfo_FullMethodName, hcore.Core_GetSystemInfoStream_FullMethodName,
		hcore.Core_LogListener_FullMethodName, hcore.Core_SelectOutbound_FullMethodName,
		hcore.Core_UrlTest_FullMethodName, hcore.Core_UrlTestActive_FullMethodName:
		return nil
	}
	return status.Error(codes.PermissionDenied, "method not allowed on the tunnel control service")
}

func ServerOptions(secret string, allow func(string) error) []grpc.ServerOption {
	if allow == nil {
		allow = AllowCoreMethod
	}
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			if err := Authenticate(ctx, secret); err != nil {
				return nil, err
			}
			if err := allow(info.FullMethod); err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}),
		grpc.StreamInterceptor(func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			if err := Authenticate(stream.Context(), secret); err != nil {
				return err
			}
			if err := allow(info.FullMethod); err != nil {
				return err
			}
			return handler(srv, stream)
		}),
	}
}

func RegisterControl(server grpc.ServiceRegistrar) {
	hcore.RegisterCoreServer(server, &hcore.CoreService{})
	hello.RegisterHelloServer(server, &hello.HelloService{})
}

func validate(options *Options, platform libbox.PlatformInterface) error {
	if options == nil || len(options.Secret) < MinSecretLength || options.ConfigContent == "" {
		return fmt.Errorf("missing tunnel configuration or credentials")
	}
	if platform == nil && !options.NativeTun {
		return fmt.Errorf("missing tunnel platform interface")
	}
	return nil
}

// Apply starts the engine with hardened VPN settings. It does not open a control server.
func Apply(options *Options, platform libbox.PlatformInterface) error {
	lifecycle.Lock()
	defer lifecycle.Unlock()
	return applyLocked(options, platform)
}

func applyLocked(options *Options, platform libbox.PlatformInterface) error {
	if err := validate(options, platform); err != nil {
		return err
	}
	if engineRunning {
		return fmt.Errorf("tunnel already running")
	}
	if err := hcore.Setup(&hcore.SetupRequest{
		BasePath: options.BasePath, WorkingDir: options.WorkingPath,
		TempDir: options.TempPath, Mode: hcore.SetupMode_OLD, Debug: false,
	}, platform); err != nil {
		return err
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(options.SettingsJSON), &settings); err != nil {
		hcore.Stop()
		return err
	}
	// VPN mode must not install a separate system proxy or a privileged listener
	// on the LAN, even if those options came from a custom profile.
	settings["enable-tun"] = !options.ProxyOnly
	settings["enable-tun-service"] = false
	settings["set-system-proxy"] = false
	settings["allow-connection-from-lan"] = false
	settings["tproxy-port"] = 0
	settingsBytes, err := json.Marshal(settings)
	if err != nil {
		hcore.Stop()
		return err
	}
	if _, err = hcore.ChangeHiddifySettings(&hcore.ChangeHiddifySettingsRequest{
		HiddifySettingsJson: string(settingsBytes),
	}, false); err != nil {
		hcore.Stop()
		return err
	}
	// StartService waits for the queue: it returns only when the background core
	// has finished starting (WAITING or STARTED) or failed. Callers such as
	// `HiddifyCli tunnel connect` must not report success before that.
	res, err := hcore.StartService(libbox.BaseContext(platform), &hcore.StartRequest{
		ConfigContent: options.ConfigContent, ConfigName: options.ConfigName,
		DisableMemoryLimit: options.DisableMemoryLimit,
	})
	if err != nil {
		hcore.Stop()
		return fmt.Errorf("start background core: %w", err)
	}
	if state := res.GetCoreState(); state != hcore.CoreStates_STARTED && state != hcore.CoreStates_WAITING && state != hcore.CoreStates_NO_AVAILABLE_PROXIES {
		hcore.Stop()
		detail := res.GetMessage()
		if detail == "" {
			detail = res.GetCoreState().String()
		}
		return fmt.Errorf("background core did not start: %s", detail)
	}
	engineRunning = true
	return nil
}

// Start applies the configuration and serves the authenticated control API on options.Port.
// Used by the Network Extension; the root admin_bgservice calls Apply and shares its own server.
func Start(options *Options, platform libbox.PlatformInterface) (err error) {
	lifecycle.Lock()
	defer lifecycle.Unlock()
	if options.Port < 1024 || options.Port > 65535 {
		return fmt.Errorf("invalid control port")
	}
	if server != nil {
		return fmt.Errorf("tunnel already running")
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(options.Port))))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			listener.Close()
			if engineRunning {
				hcore.Stop()
				engineRunning = false
			}
		}
	}()
	if err = applyLocked(options, platform); err != nil {
		return err
	}
	control := grpc.NewServer(ServerOptions(options.Secret, AllowCoreMethod)...)
	RegisterControl(control)
	server = control
	go control.Serve(listener)
	return nil
}

// StopEngine stops the core without closing a control server owned by the caller.
func StopEngine() error {
	lifecycle.Lock()
	defer lifecycle.Unlock()
	if !engineRunning {
		return nil
	}
	started := time.Now()
	_, err := hcore.Stop()
	engineRunning = false
	fmt.Printf("adminbg: hcore.Stop took %s (err=%v)\n", time.Since(started), err)
	return err
}

func Stop() error {
	lifecycle.Lock()
	defer lifecycle.Unlock()
	if server != nil {
		server.Stop()
		server = nil
	}
	if !engineRunning {
		return nil
	}
	_, err := hcore.Stop()
	engineRunning = false
	return err
}

func Pause() { hcore.Pause() }
func Wake()  { hcore.Wake() }
