package main

/*
#include <stdlib.h>
#include <stdint.h>

// Import the function from the DLL
char* parseCli(int argc, char** argv);
*/
import "C"

import (
	"fmt"
	"os"
	"unsafe"
)

func main() {
	args := os.Args

	// Convert []string to []*C.char
	var cArgs []*C.char
	for _, arg := range args {
		cArgs = append(cArgs, C.CString(arg))
	}
	defer func() {
		for _, arg := range cArgs {
			C.free(unsafe.Pointer(arg))
		}
	}()

	// Call the C function
	result := C.parseCli(C.int(len(cArgs)), (**C.char)(unsafe.Pointer(&cArgs[0])))
	fmt.Println(C.GoString(result))
}
