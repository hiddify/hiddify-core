package config

import (
	context "context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/sagernet/sing-box/experimental/libbox"
	"google.golang.org/grpc"
)

type server struct {
	UnimplementedCoreServiceServer
}

func String(s string) *string {
	return &s
}

func (s *server) ParseConfig(ctx context.Context, in *ParseConfigRequest) (resp *ParseConfigResponse, err error) {
	defer DeferPanicToError("ParseConfig", func(recovered_err error) {
		resp = &ParseConfigResponse{Error: String(fmt.Sprintf("%v", recovered_err))}
		err = nil
	})
	ctx = libbox.BaseContext(nil)
	config, err := ParseConfig(ctx, &ReadOptions{Path: in.Path}, in.Debug, nil, false)
	if err != nil {
		return &ParseConfigResponse{Error: String(err.Error())}, nil
	}
	configStr, err := config.MarshalJSONContext(ctx)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(in.Path, []byte(configStr), 0o644)
	if err != nil {
		return nil, err
	}
	return &ParseConfigResponse{Error: String("")}, nil
}

func (s *server) GenerateFullConfig(ctx context.Context, in *GenerateConfigRequest) (resp *GenerateConfigResponse, err error) {
	defer DeferPanicToError("GenerateFullConfig", func(recovered_err error) {
		resp = &GenerateConfigResponse{Error: String(fmt.Sprintf("%v", recovered_err))}
		err = nil
	})
	ctx = libbox.BaseContext(nil)
	config, err := BuildConfigJson(ctx, DefaultHiddifyOptions(), &ReadOptions{Path: in.Path})
	if err != nil {
		return nil, err
	}
	return &GenerateConfigResponse{
		Config: string(config),
	}, nil
}

func StartGRPCServer(port uint16) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	RegisterCoreServiceServer(s, &server{})

	log.Println("Server started on :", port)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return nil
}
