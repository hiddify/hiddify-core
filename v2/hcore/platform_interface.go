package hcore

import (
	"fmt"

	"github.com/sagernet/sing-box/experimental/libbox"
)

var _ libbox.PlatformInterface = (*MobilePlatformInterface)(nil)

type MobilePlatformInterface struct {
	platform libbox.PlatformInterface
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

func (h *MobilePlatformInterface) FindConnectionOwner(ipProtocol int32, sourceAddress string, sourcePort int32, destinationAddress string, destinationPort int32) (int32, error) {
	if h.platform == nil {
		return 0, fmt.Errorf("platform is nil")
	}
	return h.platform.FindConnectionOwner(ipProtocol, sourceAddress, sourcePort, destinationAddress, destinationPort)
}

func (h *MobilePlatformInterface) PackageNameByUid(uid int32) (string, error) {
	if h.platform == nil {
		return "", fmt.Errorf("platform is nil")
	}
	return h.platform.PackageNameByUid(uid)
}

func (h *MobilePlatformInterface) UIDByPackageName(packageName string) (int32, error) {
	if h.platform == nil {
		return 0, fmt.Errorf("platform is nil")
	}
	return h.platform.UIDByPackageName(packageName)
}

func (h *MobilePlatformInterface) UsePlatformDefaultInterfaceMonitor() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UsePlatformDefaultInterfaceMonitor()
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

func (h *MobilePlatformInterface) UsePlatformInterfaceGetter() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UsePlatformInterfaceGetter()
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
