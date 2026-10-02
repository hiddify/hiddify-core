package hcore

import (
	"fmt"
	"os"

	"github.com/sagernet/sing-box/experimental/libbox"
)

var _ libbox.PlatformInterface = (*MobilePlatformInterface)(nil)

type MobilePlatformInterface struct {
	platform libbox.PlatformInterface
}

func (h *MobilePlatformInterface) SystemCertificates() libbox.StringIterator {
	if h.platform == nil {
		return nil
	}
	return h.platform.SystemCertificates()
}
func (h *MobilePlatformInterface) LocalDNSTransport() libbox.LocalDNSTransport {
	if h.platform == nil {
		return nil
	}
	return h.platform.LocalDNSTransport()
}
func (h *MobilePlatformInterface) UsePlatformAutoDetectInterfaceControl() bool {
	if h.platform == nil {
		return true
	}
	return h.platform.UsePlatformAutoDetectInterfaceControl()
}

func (h *MobilePlatformInterface) AutoDetectInterfaceControl(fd int32) error {
	if h.platform == nil {
		return nil
	}
	return h.platform.AutoDetectInterfaceControl(fd)
}

func (h *MobilePlatformInterface) OpenTun(options libbox.TunOptions) (int32, error) {
	if h.platform == nil {
		return 0, fmt.Errorf("platform is nil")
	}
	return h.platform.OpenTun(options)
}

func (h *MobilePlatformInterface) WriteLog(message string) {
	Log(LogLevel_DEBUG, LogType_CORE, message)
}

func (h *MobilePlatformInterface) UseProcFS() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UseProcFS()
}

func (h *MobilePlatformInterface) FindConnectionOwner(ipProtocol int32, sourceAddress string, sourcePort int32, destinationAddress string, destinationPort int32) (*libbox.ConnectionOwner, error) {
	if h.platform == nil {
		return nil, fmt.Errorf("platform is nil")
	}
	return h.platform.FindConnectionOwner(ipProtocol, sourceAddress, sourcePort, destinationAddress, destinationPort)
}

func (h *MobilePlatformInterface) StartDefaultInterfaceMonitor(listener libbox.InterfaceUpdateListener) error {
	if h.platform == nil {
		return fmt.Errorf("platform is nil")
	}
	return h.platform.StartDefaultInterfaceMonitor(listener)
}

func (h *MobilePlatformInterface) CloseDefaultInterfaceMonitor(listener libbox.InterfaceUpdateListener) error {
	if h.platform == nil {
		return fmt.Errorf("platform is nil")
	}
	return h.platform.CloseDefaultInterfaceMonitor(listener)
}

func (h *MobilePlatformInterface) GetInterfaces() (libbox.NetworkInterfaceIterator, error) {
	if h.platform == nil {
		return nil, fmt.Errorf("platform is nil")
	}
	return h.platform.GetInterfaces()
}

func (h *MobilePlatformInterface) UnderNetworkExtension() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UnderNetworkExtension()
}

func (h *MobilePlatformInterface) IncludeAllNetworks() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.IncludeAllNetworks()
}

func (h *MobilePlatformInterface) ReadWIFIState() *libbox.WIFIState {
	if h.platform == nil {
		return nil
	}
	return h.platform.ReadWIFIState()
}

func (h *MobilePlatformInterface) ClearDNSCache() {
	if h.platform == nil {
		return
	}
	h.platform.ClearDNSCache()
}

func (h *MobilePlatformInterface) SendNotification(notification *libbox.Notification) error {
	if h.platform == nil {
		return nil
	}
	return h.platform.SendNotification(notification)
}

func (h *MobilePlatformInterface) CancelNotification(identifier string, typeID int32) error {
	if h.platform == nil {
		return nil
	}
	return h.platform.CancelNotification(identifier, typeID)
}

func (h *MobilePlatformInterface) StartNeighborMonitor(listener libbox.NeighborUpdateListener) error {
	if h.platform == nil {
		return nil
	}
	return h.platform.StartNeighborMonitor(listener)
}

func (h *MobilePlatformInterface) CloseNeighborMonitor(listener libbox.NeighborUpdateListener) error {
	if h.platform == nil {
		return nil
	}
	return h.platform.CloseNeighborMonitor(listener)
}

func (h *MobilePlatformInterface) RegisterMyInterface(name string) {
	if h.platform == nil {
		return
	}
	h.platform.RegisterMyInterface(name)
}

func (h *MobilePlatformInterface) UsePlatformShell() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UsePlatformShell()
}

func (h *MobilePlatformInterface) CheckPlatformShell() error {
	if h.platform == nil {
		return os.ErrInvalid
	}
	return h.platform.CheckPlatformShell()
}

func (h *MobilePlatformInterface) OpenShellSession(user *libbox.PlatformUser, command string, environ libbox.StringIterator, term string, rows int32, cols int32) (libbox.ShellSession, error) {
	if h.platform == nil {
		return nil, os.ErrInvalid
	}
	return h.platform.OpenShellSession(user, command, environ, term, rows, cols)
}

func (h *MobilePlatformInterface) LookupUser(username string) (*libbox.PlatformUser, error) {
	if h.platform == nil {
		return nil, os.ErrInvalid
	}
	return h.platform.LookupUser(username)
}

func (h *MobilePlatformInterface) LookupSFTPServer() (*libbox.StringBox, error) {
	if h.platform == nil {
		return nil, os.ErrInvalid
	}
	return h.platform.LookupSFTPServer()
}

func (h *MobilePlatformInterface) ReadSystemSSHHostKey() (*libbox.StringBox, error) {
	if h.platform == nil {
		return nil, os.ErrInvalid
	}
	return h.platform.ReadSystemSSHHostKey()
}

func (h *MobilePlatformInterface) TailscaleHostname() string {
	if h.platform == nil {
		return ""
	}
	return h.platform.TailscaleHostname()
}

func (h *MobilePlatformInterface) UsePlatformBridge() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UsePlatformBridge()
}

func (h *MobilePlatformInterface) CreateBridge(options *libbox.BridgeOptions) (libbox.BridgeSession, error) {
	if h.platform == nil {
		return nil, os.ErrInvalid
	}
	return h.platform.CreateBridge(options)
}

func (h *MobilePlatformInterface) UsePlatformAutoRedirect() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UsePlatformAutoRedirect()
}

func (h *MobilePlatformInterface) CreateAutoRedirect(options []byte, handler libbox.AutoRedirectHandler) (libbox.AutoRedirectSession, error) {
	if h.platform == nil {
		return nil, os.ErrInvalid
	}
	return h.platform.CreateAutoRedirect(options, handler)
}
