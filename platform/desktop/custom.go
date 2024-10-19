package main

/*
#include "stdint.h"
*/
import "C"

import (
	hcore "github.com/hiddify/hiddify-core/v2/hcore"

	"github.com/sagernet/sing-box/log"
)

//export setup
func setup(baseDir *C.char, workingDir *C.char, tempDir *C.char, mode C.int, listen *C.char, secret *C.char, statusPort C.longlong, debug bool) (CErr *C.char) {
	// err := hcore.Setup(C.GoString(baseDir), C.GoString(workingDir), C.GoString(tempDir), int64(statusPort), debug)
	err := hcore.Setup(hcore.SetupParameters{
		BasePath:          C.GoString(baseDir),
		WorkingDir:        C.GoString(workingDir),
		TempDir:           C.GoString(tempDir),
		FlutterStatusPort: int64(statusPort),
		Debug:             debug,
		Mode:              hcore.SetupMode(mode),
		Listen:            C.GoString(listen),
		Secret:            C.GoString(secret),
	})
	return emptyOrErrorC(err)
}

//export start
func start(configPath *C.char, disableMemoryLimit bool) (CErr *C.char) {
	_, err := hcore.Start(&hcore.StartRequest{
		ConfigPath:             C.GoString(configPath),
		EnableOldCommandServer: true,
		DisableMemoryLimit:     disableMemoryLimit,
	})
	return emptyOrErrorC(err)
}

//export stop
func stop() (CErr *C.char) {
	_, err := hcore.Stop()
	return emptyOrErrorC(err)
}

//export restart
func restart(configPath *C.char, disableMemoryLimit bool) (CErr *C.char) {
	_, err := hcore.Restart(&hcore.StartRequest{
		ConfigPath:             C.GoString(configPath),
		EnableOldCommandServer: true,
		DisableMemoryLimit:     disableMemoryLimit,
	})
	return emptyOrErrorC(err)
}

func emptyOrErrorC(err error) *C.char {
	if err == nil {
		return C.CString("")
	}
	log.Error(err.Error())
	return C.CString(err.Error())
}

func main() {}

//export GetServerPublicKey
func GetServerPublicKey() []byte {
	return hcore.GetGrpcServerPublicKey()
}

//export AddGrpcClientPublicKey
func AddGrpcClientPublicKey(clientPublicKey []byte) error {
	return hcore.AddGrpcClientPublicKey(clientPublicKey)
}

//export close
func close(mode C.int) {
	hcore.Close(hcore.SetupMode(int32(mode)))
}
