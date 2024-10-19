package main

import "C"
import hcore "github.com/hiddify/hiddify-core/v2/hcore"

//export StartCoreGrpcServer
func StartCoreGrpcServer(listenAddress *C.char) (CErr *C.char) {
	_, err := hcore.StartCoreGrpcServer(C.GoString(listenAddress))
	return emptyOrErrorC(err)
}
