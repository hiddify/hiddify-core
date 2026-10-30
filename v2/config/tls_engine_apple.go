//go:build darwin && cgo

package config

import C "github.com/sagernet/sing-box/constant"

// systemTLSEngine is the system's TLS on macOS and iOS (sing-box's Apple engine needs cgo).
func systemTLSEngine() string {
	return C.TLSEngineApple
}
