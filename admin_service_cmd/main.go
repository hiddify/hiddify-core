package main

/*
#cgo LDFLAGS: bin/libcore.dll
#include <stdlib.h>
#include <stdint.h>

// Import the function from the DLL
extern void AdminServiceStart(char *arg);

*/
import "C"
import (
	"os"
)

func main() {
	args := os.Args
	// Check if there is at least one command-line argument
	if len(args) < 2 {
		println("Usage: hiddify-service.exe empty/start/stop/uninstall/install")
		// os.Exit(1)
		args = append(args, "")
	}
	// fmt.Printf("os.Args: %+v", args)
	os.Chdir(os.Args[0])
	// Convert the Go string to a C string
	arg := C.CString(args[1])
	// defer C.free(unsafe.Pointer(arg))

	// Call AdminServiceStart with the C string
	C.AdminServiceStart(arg)
}
