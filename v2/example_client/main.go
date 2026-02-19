package main

import (
	"context"
	"log"
	"time"

	"github.com/hiddify/hiddify-core/v2/hello"
	"google.golang.org/grpc"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := hello.NewHelloClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// SayHello
	r, err := c.SayHello(ctx, &hello.HelloRequest{Name: defaultName})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)

	// SayHelloStream
	stream, err := c.SayHelloStream(ctx)
	if err != nil {
		log.Fatalf("could not stream: %v", err)
	}

	names := []string{"Alice", "Bob", "Charlie"}

	for _, name := range names {
		err := stream.Send(&hello.HelloRequest{Name: name})
		if err != nil {
			log.Fatalf("could not send: %v", err)
		}
		r, err := stream.Recv()
		if err != nil {
			log.Fatalf("could not receive: %v", err)
		}
		log.Printf("Received1: %s", r.Message)
		r2, err2 := stream.Recv()
		if err2 != nil {
			log.Fatalf("could not receive2: %v", err2)
		}
		log.Printf("Received: %s", r2.Message)
		time.Sleep(1 * time.Second)
	}
}
