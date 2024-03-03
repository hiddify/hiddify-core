package v2

import (
	"context"
	"log"
	"time"

	pb "github.com/hiddify/libcore/hiddifyrpc"
)

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Message: "Hello, " + in.Name}, nil
}
func (s *server) SayHelloStream(stream pb.Hiddify_SayHelloStreamServer) error {

	for {
		req, err := stream.Recv()
		if err != nil {
			log.Printf("stream.Recv() failed: %v", err)
			break
		}
		log.Printf("Received: %v", req.Name)
		time.Sleep(1 * time.Second)
		err = stream.Send(&pb.HelloResponse{Message: "Hello, " + req.Name})
		if err != nil {
			log.Printf("stream.Send() failed: %v", err)
			break
		}
		err = stream.Send(&pb.HelloResponse{Message: "Hello again, " + req.Name})
		if err != nil {
			log.Printf("stream.Send() failed: %v", err)
			break
		}
	}
	return nil
}
