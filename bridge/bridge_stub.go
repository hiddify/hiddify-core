//go:build !cgo
// +build !cgo

package bridge

import "unsafe"

func InitializeDartApi(api unsafe.Pointer) {
}
func SendStringToPort(port int64, msg string) {
}
