package config

import (
	context "context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/sagernet/sing-box/option"
	"google.golang.org/grpc"
)

type server struct {
	UnimplementedCoreServiceServer
}

func String(s string) *string {
	return &s
}

func (s *server) ParseConfig(ctx context.Context, in *ParseConfigRequest) (*ParseConfigResponse, error) {
	config, err := ParseConfig(in.TempPath, in.Debug)
	if err != nil {
		return &ParseConfigResponse{Error: String(err.Error())}, nil
	}
	err = os.WriteFile(in.Path, config, 0o644)
	if err != nil {
		return nil, err
	}
	return &ParseConfigResponse{Error: String("")}, nil
}

func (s *server) GenerateFullConfig(ctx context.Context, in *GenerateConfigRequest) (*GenerateConfigResponse, error) {
	os.Chdir(filepath.Dir(in.Path))
	content, err := os.ReadFile(in.Path)
	if err != nil {
		return nil, err
	}
	var options option.Options
	err = options.UnmarshalJSON(content)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	config, err := BuildConfigJson(*DefaultHiddifyOptions(), options)
	if err != nil {
		return nil, err
	}
	return &GenerateConfigResponse{
		Config: config,
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
