package hcore

import (
	"fmt"

	"github.com/sagernet/sing-box/experimental/libbox"
)

var _ libbox.PlatformInterface = (*HiddifyPlatformInterface)(nil)

type HiddifyPlatformInterface struct {
	platform libbox.PlatformInterface
}

func (h *HiddifyPlatformInterface) UsePlatformAutoDetectInterfaceControl() bool {
	if h.platform == nil {
		return true
	}
	return h.platform.UsePlatformAutoDetectInterfaceControl()
}

func (h *HiddifyPlatformInterface) AutoDetectInterfaceControl(fd int32) error {
	if h.platform == nil {
		return nil
	}
	return h.platform.AutoDetectInterfaceControl(fd)
}

func (h *HiddifyPlatformInterface) OpenTun(options libbox.TunOptions) (int32, error) {
	if h.platform == nil {
		return 0, fmt.Errorf("platform is nil")
	}
	return h.platform.OpenTun(options)
}

func (h *HiddifyPlatformInterface) WriteLog(message string) {
	Log(LogLevel_DEBUG, LogType_CORE, message)
}

func (h *HiddifyPlatformInterface) UseProcFS() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UseProcFS()
}

func (h *HiddifyPlatformInterface) FindConnectionOwner(ipProtocol int32, sourceAddress string, sourcePort int32, destinationAddress string, destinationPort int32) (int32, error) {
	if h.platform == nil {
		return 0, fmt.Errorf("platform is nil")
	}
	return h.platform.FindConnectionOwner(ipProtocol, sourceAddress, sourcePort, destinationAddress, destinationPort)
}

func (h *HiddifyPlatformInterface) PackageNameByUid(uid int32) (string, error) {
	if h.platform == nil {
		return "", fmt.Errorf("platform is nil")
	}
	return h.platform.PackageNameByUid(uid)
}

func (h *HiddifyPlatformInterface) UIDByPackageName(packageName string) (int32, error) {
	if h.platform == nil {
		return 0, fmt.Errorf("platform is nil")
	}
	return h.platform.UIDByPackageName(packageName)
}

func (h *HiddifyPlatformInterface) UsePlatformDefaultInterfaceMonitor() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UsePlatformDefaultInterfaceMonitor()
}

func (h *HiddifyPlatformInterface) StartDefaultInterfaceMonitor(listener libbox.InterfaceUpdateListener) error {
	if h.platform == nil {
		return fmt.Errorf("platform is nil")
	}
	return h.platform.StartDefaultInterfaceMonitor(listener)
}

func (h *HiddifyPlatformInterface) CloseDefaultInterfaceMonitor(listener libbox.InterfaceUpdateListener) error {
	if h.platform == nil {
		return fmt.Errorf("platform is nil")
	}
	return h.platform.CloseDefaultInterfaceMonitor(listener)
}

func (h *HiddifyPlatformInterface) UsePlatformInterfaceGetter() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UsePlatformInterfaceGetter()
}

func (h *HiddifyPlatformInterface) GetInterfaces() (libbox.NetworkInterfaceIterator, error) {
	if h.platform == nil {
		return nil, fmt.Errorf("platform is nil")
	}
	return h.platform.GetInterfaces()
}

func (h *HiddifyPlatformInterface) UnderNetworkExtension() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.UnderNetworkExtension()
}

func (h *HiddifyPlatformInterface) IncludeAllNetworks() bool {
	if h.platform == nil {
		return false
	}
	return h.platform.IncludeAllNetworks()
}

func (h *HiddifyPlatformInterface) ReadWIFIState() *libbox.WIFIState {
	if h.platform == nil {
		return nil
	}
	return h.platform.ReadWIFIState()
}

func (h *HiddifyPlatformInterface) ClearDNSCache() {
	if h.platform == nil {
		return
	}
	h.platform.ClearDNSCache()
}
