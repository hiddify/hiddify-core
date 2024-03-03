package v2

import "C"
import (
	"log"
	"net"

	pb "github.com/hiddify/libcore/hiddifyrpc"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

type server struct {
	pb.UnimplementedHiddifyServer
}

//export StartGrpcServer
func StartGrpcServer() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Printf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterHiddifyServer(s, &server{})
	log.Printf("Server listening on %s", port)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
		}
	}()
}
