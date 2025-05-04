package main

import (
	"os"
	"os/signal"
	"syscall"

	hcore "github.com/hiddify/hiddify-core/v2/hcore"
)

func main() {
	// defer C.free(unsafe.Pointer(port))
	hcore.StartGrpcServer("127.0.0.1:50051", "hello")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}
