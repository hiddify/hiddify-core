// +build cgo
package bridge

// #include "stdint.h"
// #include "include/dart_api_dl.c"
//
// // Go does not allow calling C function pointers directly. So we are
// // forced to provide a trampoline.
// bool GoDart_PostCObject(Dart_Port_DL port, Dart_CObject* obj) {
//   return Dart_PostCObject_DL(port, obj);
// }
import "C"
import (
	"fmt"
	"unsafe"
)

func InitializeDartApi(api unsafe.Pointer) {
	if C.Dart_InitializeApiDL(api) != 0 {
		panic("failed to initialize Dart DL C API: version mismatch. " +
			"must update include/ to match Dart SDK version")
	}
}

func SendStringToPort(port int64, msg string) {
	var obj C.Dart_CObject
	obj._type = C.Dart_CObject_kString
	msg_obj := C.CString(msg) // go string -> char*s
	// union type, we do a force conversion
	ptr := unsafe.Pointer(&obj.value[0])
	*(**C.char)(ptr) = msg_obj
	ret := C.GoDart_PostCObject(C.Dart_Port_DL(port), &obj)
	if !ret {
		fmt.Println("ERROR: post to port ", port, " failed", msg)
	}
}
