package main

/*
#include <stdlib.h>
#include <signal.h>
#include "stdint.h"
*/
import "C"

import (
	// "os"
	// "os/signal"
	"runtime"
	// "syscall"
	"unsafe"

	hcore "github.com/hiddify/hiddify-core/v2/hcore"
	"github.com/sagernet/sing-box/log"
)

// func init() {
// 	runtime.LockOSThread()
// 	C.init_signals()
// 	runtime.UnlockOSThread()

// 	go handleSignals()

// 	// Your other initialization code can go here
// }

// // Signal handling function
// func handleSignals() {
// 	signalChan := make(chan os.Signal, 1)
// 	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGURG)

// 	for {
// 		<-signalChan
// 		// switch sig {
// 		// case syscall.SIGINT, syscall.SIGTERM:
// 		// 	// runtime.LockOSThread() // Lock to the current OS thread
// 		// 	// defer runtime.UnlockOSThread()
// 		// 	log.Info("Received signal:", sig)

// 		// 	// Call stop function or perform cleanup
// 		// 	if err := stop(); err != nil {
// 		// 		log.Error("Error stopping the application:", err)
// 		// 	}
// 		// 	log.Info("Application stopped gracefully.")
// 		// }
// 	}
// }

func main() {}

//export cleanup
func cleanup() {
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()
	// C.cleanup_signals()
}

func emptyOrErrorC(err error) *C.char {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err == nil {
		return C.CString("")
	}
	log.Error(err.Error())
	str := C.CString(err.Error())
	defer C.free(unsafe.Pointer(str))
	return str
}

//export setup
func setup(baseDir *C.char, workingDir *C.char, tempDir *C.char, mode C.int, listen *C.char, secret *C.char, statusPort C.longlong, debug bool) *C.char {
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()

	// // Ensure signals are initialized
	// C.init_signals()

	params := hcore.SetupRequest{
		BasePath:          C.GoString(baseDir),
		WorkingDir:        C.GoString(workingDir),
		TempDir:           C.GoString(tempDir),
		FlutterStatusPort: int64(statusPort),
		Debug:             bool(debug),
		Mode:              hcore.SetupMode(mode),
		Listen:            C.GoString(listen),
		Secret:            C.GoString(secret),
	}
	err := hcore.Setup(&params)
	return emptyOrErrorC(err)
}

//export freeString
func freeString(str *C.char) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	C.free(unsafe.Pointer(str))
}

//export start
func start(configPath *C.char, disableMemoryLimit bool) *C.char {
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()

	_, err := hcore.Start(&hcore.StartRequest{
		ConfigPath:             C.GoString(configPath),
		EnableOldCommandServer: true,
		DisableMemoryLimit:     bool(disableMemoryLimit),
	})
	return emptyOrErrorC(err)
}

//export stop
func stop() *C.char {
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()

	_, err := hcore.Stop()
	return emptyOrErrorC(err)
}

//export restart
func restart(configPath *C.char, disableMemoryLimit bool) *C.char {
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()

	_, err := hcore.Restart(&hcore.StartRequest{
		ConfigPath:             C.GoString(configPath),
		EnableOldCommandServer: true,
		DisableMemoryLimit:     bool(disableMemoryLimit),
	})
	return emptyOrErrorC(err)
}

//export GetServerPublicKey
func GetServerPublicKey() *C.char {
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()

	publicKey := hcore.GetGrpcServerPublicKey()
	return C.CString(string(publicKey)) // Return as C string, caller must free
}

//export AddGrpcClientPublicKey
func AddGrpcClientPublicKey(clientPublicKey *C.char) *C.char {
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()

	// Convert C string to Go byte slice
	clientKey := C.GoBytes(unsafe.Pointer(clientPublicKey), C.int(len(C.GoString(clientPublicKey))))
	err := hcore.AddGrpcClientPublicKey(clientKey)
	return emptyOrErrorC(err)
}

//export closeGrpc
func closeGrpc(mode C.int) {
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()

	hcore.Close(hcore.SetupMode(mode))
}
