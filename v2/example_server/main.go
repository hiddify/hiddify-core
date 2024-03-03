package main

import (
	"os"
	"os/signal"
	"syscall"

	v2 "github.com/hiddify/libcore/v2"
)

func main() {
	v2.StartGrpcServer()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}
