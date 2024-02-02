package main

/*
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
	if len(args) < 2 {
		fmt.Println("Usage: hiddify-service.exe empty/start/stop/uninstall/install")
		args = append(args, "")
	}

	os.Chdir(os.Args[0])

	arg := C.CString(args[1])
	defer C.free(unsafe.Pointer(arg))

	result := C.AdminServiceStart(arg)
	defer C.free(unsafe.Pointer(result))

	goRes := C.GoString(result)

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
		os.Exit(parsedExitCode)
	}
}
