// Package tunnel is the gomobile bind surface for HiddifyTunnelCore.xcframework.
// ObjC symbols stay Tunnel* (TunnelStart, TunnelOptions, …) for the packet-tunnel
// extension. Implementation lives in adminbg.
package tunnel

import (
	"github.com/hiddify/hiddify-core/v2/hcore/adminbg"
	_ "github.com/sagernet/gomobile"
	"github.com/sagernet/sing-box/experimental/libbox"
)

// Options is the Network Extension configuration passed across the gomobile boundary.
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
	ProxyOnly          bool
	NativeTun          bool
}

func (o *Options) toAdmin() *adminbg.Options {
	if o == nil {
		return nil
	}
	return &adminbg.Options{
		BasePath:           o.BasePath,
		WorkingPath:        o.WorkingPath,
		TempPath:           o.TempPath,
		ConfigContent:      o.ConfigContent,
		ConfigName:         o.ConfigName,
		SettingsJSON:       o.SettingsJSON,
		Secret:             o.Secret,
		Port:               o.Port,
		DisableMemoryLimit: o.DisableMemoryLimit,
		ProxyOnly:          o.ProxyOnly,
		NativeTun:          o.NativeTun,
	}
}

func Apply(options *Options, platform libbox.PlatformInterface) error {
	return adminbg.Apply(options.toAdmin(), platform)
}

func Start(options *Options, platform libbox.PlatformInterface) error {
	return adminbg.Start(options.toAdmin(), platform)
}

func Stop() error { return adminbg.Stop() }

func Pause() { adminbg.Pause() }
func Wake()  { adminbg.Wake() }
