package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"

	"github.com/hiddify/hiddify-core/cmd"
)

//export parseCli
func parseCli(argc C.int, argv **C.char) *C.char {
	args := make([]string, argc)
	for i := 0; i < int(argc); i++ {
		// fmt.Println("parseCli", C.GoString(*argv))
		args[i] = C.GoString(*argv)
		argv = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(argv)) + uintptr(unsafe.Sizeof(*argv))))
	}
	err := cmd.ParseCli(args[1:])
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}
