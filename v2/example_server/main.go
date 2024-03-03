package main

import (
	"os"
	"os/signal"
	"syscall"

	v2 "github.com/hiddify/libcore/v2"
)

func main() {

	// defer C.free(unsafe.Pointer(port))
	v2.StartGrpcServerGo("127.0.0.1:50051")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}
