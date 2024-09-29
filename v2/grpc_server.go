package v2

/*
#include "stdint.h"
*/

import (
	"log"
	"net"

	"github.com/hiddify/hiddify-core/extension"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"

	"google.golang.org/grpc"
)

type HelloService struct {
	pb.UnimplementedHelloServer
}
type CoreService struct {
	pb.UnimplementedCoreServer
}

type TunnelService struct {
	pb.UnimplementedTunnelServiceServer
}

func StartGrpcServer(listenAddressG string, service string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return nil, err
	}
	s := grpc.NewServer()
	if service == "core" {

		// Setup("./tmp/", "./tmp", "./tmp", 11111, false)

		useFlutterBridge = false
		pb.RegisterCoreServer(s, &CoreService{})
		pb.RegisterExtensionHostServiceServer(s, &extension.ExtensionHostService{})
	} else if service == "hello" {
		pb.RegisterHelloServer(s, &HelloService{})
	} else if service == "tunnel" {
		pb.RegisterTunnelServiceServer(s, &TunnelService{})
	}
	log.Printf("Server listening on %s", listenAddressG)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
		}
		log.Printf("Server stopped")
		// cancel()
	}()
	return s, nil
}

func StartCoreGrpcServer(listenAddressG string) (*grpc.Server, error) {
	return StartGrpcServer(listenAddressG, "core")
}

func StartHelloGrpcServer(listenAddressG string) (*grpc.Server, error) {
	return StartGrpcServer(listenAddressG, "hello")
}

func StartTunnelGrpcServer(listenAddressG string) (*grpc.Server, error) {
	return StartGrpcServer(listenAddressG, "tunnel")
}
