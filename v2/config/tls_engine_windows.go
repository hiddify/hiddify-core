//go:build windows

package config

import (
	"github.com/sagernet/sing-box/common/schannel"
	C "github.com/sagernet/sing-box/constant"
)

// systemTLSEngine is Schannel, on Windows 10 1809 (build 17763) or later.
func systemTLSEngine() string {
	if schannel.CheckPlatform() != nil {
		return ""
	}
	return C.TLSEngineWindows
}
