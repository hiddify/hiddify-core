package v2

import (
	"github.com/sagernet/sing-box/experimental/libbox"
)

type platformStub struct{}

// Local DNS
func (p *platformStub) LocalDNSTransport() libbox.LocalDNSTransport { return nil }

// Interface control
func (p *platformStub) UsePlatformAutoDetectInterfaceControl() bool { return false }
func (p *platformStub) AutoDetectInterfaceControl(fd int32) error   { return nil }

// TUN
func (p *platformStub) OpenTun(options libbox.TunOptions) (int32, error) { return 0, nil }

// Logs
func (p *platformStub) WriteLog(message string) {}

// ProcFS / UID
func (p *platformStub) UseProcFS() bool { return false }
func (p *platformStub) FindConnectionOwner(ipProtocol int32, sourceAddress string, sourcePort int32, destinationAddress string, destinationPort int32) (int32, error) {
	return 0, nil
}
func (p *platformStub) PackageNameByUid(uid int32) (string, error) { return "", nil }
func (p *platformStub) UIDByPackageName(packageName string) (int32, error) {
	return 0, nil
}

// Default interface monitor
func (p *platformStub) StartDefaultInterfaceMonitor(listener libbox.InterfaceUpdateListener) error { return nil }
func (p *platformStub) CloseDefaultInterfaceMonitor(listener libbox.InterfaceUpdateListener) error { return nil }

// Interface getter
func (p *platformStub) GetInterfaces() (libbox.NetworkInterfaceIterator, error) { return nil, nil }

// NE extension / network flags
func (p *platformStub) UnderNetworkExtension() bool { return false }
func (p *platformStub) IncludeAllNetworks() bool    { return true }

// WIFI / Certs / DNS cache
func (p *platformStub) ReadWIFIState() *libbox.WIFIState { return nil }
func (p *platformStub) SystemCertificates() libbox.StringIterator { return nil }
func (p *platformStub) ClearDNSCache()                         {}

// Notifications
func (p *platformStub) SendNotification(notification *libbox.Notification) error { return nil }
