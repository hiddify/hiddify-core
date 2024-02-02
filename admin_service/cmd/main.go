package main

/*
#cgo LDFLAGS: bin/libcore.dll
#include <stdlib.h>
#include <stdint.h>

// Import the function from the DLL
char* AdminServiceStart(const char* arg);


*/
import "C"
import (
	"fmt"
	"os"
	"strings"
	"unsafe"
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

	result := C.AdminServiceStart(arg)
	goRes := C.GoString(result)
	defer C.free(unsafe.Pointer(result))

	parts := strings.SplitN(goRes, " ", 2)

	var parsedExitCode int
	_, err := fmt.Sscanf(parts[0], "%d", &parsedExitCode)
	parsedOutMessage := parts[1]
	if err != nil {
		fmt.Println("Error parsing the string:", err)
		return
	}
	fmt.Printf("%d %s", parsedExitCode, parsedOutMessage)

	if parsedExitCode != 0 {
		os.Exit(int(parsedExitCode))
	}

}
