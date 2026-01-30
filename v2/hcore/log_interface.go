package hcore

import (
	"github.com/hiddify/hiddify-core/v2/service_manager"
	daemon "github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/log"
)

var _ log.PlatformWriter = (*LogInterface)(nil)

// var (
// 	// _ platform.Interface = (*DesktopPlatformInterface)(nil)
// 	_ log.PlatformWriter = (*DesktopPlatformInterface)(nil)
// )

type LogInterface struct{}

func (h *LogInterface) ServiceStop() error {
	return service_manager.OnMainServiceClose()
}
func (h *LogInterface) ServiceReload() error {
	return service_manager.OnMainServiceStart()

}
func (h *LogInterface) SystemProxyStatus() (*daemon.SystemProxyStatus, error) {
	return nil, nil
}
func (h *LogInterface) SetSystemProxyEnabled(enabled bool) error {
	return nil
}

func (h *LogInterface) WriteDebugMessage(message string) {
	h.WriteMessage(log.LevelDebug, message)
}
func (h *LogInterface) WriteMessage(level log.Level, message string) {
	Log(convertLogLevel(level), LogType_SERVICE, message)
}
func convertLogLevel(level log.Level) LogLevel {
	switch level {
	case log.LevelDebug:
		return LogLevel_DEBUG
	case log.LevelInfo:
		return LogLevel_INFO
	case log.LevelWarn:
		return LogLevel_WARNING
	case log.LevelError:
		return LogLevel_ERROR
	case log.LevelFatal:
		return LogLevel_FATAL
	}
	return LogLevel(log.LevelDebug)
}

// 	router          adapter.Router
// 	SocksPort       uint16
// 	processSearcher process.Searcher
// }

// // DisableColors implements log.PlatformWriter.
// func (h *DesktopPlatformInterface) DisableColors() bool {
// 	return true
// }

// func (h *DesktopPlatformInterface) WriteMessage(level uint8, message string) {
// 	Log(LogLevel(level), LogType_SERVICE, message)
// }

// func (h *DesktopPlatformInterface) Initialize(ctx context.Context, router adapter.Router) error {
// 	h.router = router
// 	searcher, err := process.NewSearcher(process.Config{
// 		PackageManager: router.PackageManager(),
// 	})
// 	h.processSearcher = searcher
// 	return err
// }

// func (h *DesktopPlatformInterface) AutoDetectInterfaceControl() func(network string, address string, conn syscall.RawConn) error {
// 	return nil
// }

// func (h *DesktopPlatformInterface) CreateDefaultInterfaceMonitor(logger logger.Logger) tun.DefaultInterfaceMonitor {
// 	return nil
// }

// func (h *DesktopPlatformInterface) FindProcessInfo(ctx context.Context, network string, source netip.AddrPort, destination netip.AddrPort) (*process.Info, error) {
// 	return h.processSearcher.FindProcessInfo(ctx, network, source, destination)
// }

// func (h *DesktopPlatformInterface) Interfaces() ([]control.Interface, error) { return nil, nil }

// func (h *DesktopPlatformInterface) OpenTun(options *tun.Options, platformOptions option.TunPlatformOptions) (tun.Tun, error) {
// 	if hutils.IsAdmin() {
// 		return tun.New(*options)
// 	}

// 	return nil, E.New("Tun needs admin permission")
// }

// func (h *DesktopPlatformInterface) ReadWIFIState() adapter.WIFIState {
// 	return adapter.WIFIState{}
// }

// func (h *DesktopPlatformInterface) UsePlatformAutoDetectInterfaceControl() bool { return false }

// func (h *DesktopPlatformInterface) WriteLog(message string) {
// 	Log(LogLevel_DEBUG, LogType_CORE, message)
// }

// func (h *DesktopPlatformInterface) UsePlatformDefaultInterfaceMonitor() bool { return false }

// func (h *DesktopPlatformInterface) UsePlatformInterfaceGetter() bool { return false }

// func (h *DesktopPlatformInterface) UnderNetworkExtension() bool { return false }

// func (h *DesktopPlatformInterface) IncludeAllNetworks() bool { return false }

// func (h *DesktopPlatformInterface) ClearDNSCache() {}
