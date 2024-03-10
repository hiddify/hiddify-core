package v2

/*
#include "stdint.h"
*/

import "C"
import (
	"log"
	"net"

	pb "github.com/hiddify/libcore/hiddifyrpc"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedHiddifyServer
}

//export StartGrpcServer
func StartGrpcServer(listenAddress *C.char) (CErr *C.char) {
	//Example Listen Address: "127.0.0.1:50051"
	err := StartGrpcServerGo(C.GoString(listenAddress))
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

func StartGrpcServerGo(listenAddressG string) error {
	//Example Listen Address: "127.0.0.1:50051"
	// defer C.free(unsafe.Pointer(CErr))          // free the C string when it's no longer needed
	// defer C.free(unsafe.Pointer(listenAddress)) // free the C string when it's no longer needed

	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return err
	}
	s := grpc.NewServer()
	pb.RegisterHiddifyServer(s, &server{})
	log.Printf("Server listening on %s", listenAddressG)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
		}
	}()
	return nil
}
