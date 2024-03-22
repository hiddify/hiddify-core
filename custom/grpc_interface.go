package main

import "C"
import v2 "github.com/hiddify/hiddify-core/v2"

//export StartCoreGrpcServer
func StartCoreGrpcServer(listenAddress *C.char) (CErr *C.char) {
	err := v2.StartCoreGrpcServer(C.GoString(listenAddress))
	return emptyOrErrorC(err)
}
