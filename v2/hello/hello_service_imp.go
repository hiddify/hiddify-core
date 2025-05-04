package hello

import (
	"context"
	"log"
	"time"
)

type HelloService struct {
	UnimplementedHelloServer
}

func (s *HelloService) SayHello(ctx context.Context, in *HelloRequest) (*HelloResponse, error) {
	return &HelloResponse{Message: "Hello, " + in.Name}, nil
}

func (s *HelloService) SayHelloStream(stream Hello_SayHelloStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			log.Printf("stream.Recv() failed: %v", err)
			break
		}
		log.Printf("Received: %v", req.Name)
		time.Sleep(1 * time.Second)
		err = stream.Send(&HelloResponse{Message: "Hello, " + req.Name})
		if err != nil {
			log.Printf("stream.Send() failed: %v", err)
			break
		}
		err = stream.Send(&HelloResponse{Message: "Hello again, " + req.Name})
		if err != nil {
			log.Printf("stream.Send() failed: %v", err)
			break
		}
	}
	return nil
}
