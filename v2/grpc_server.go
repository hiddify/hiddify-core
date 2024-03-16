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

type HelloService struct {
	pb.UnimplementedHelloServer
}
type CoreService struct {
	pb.UnimplementedCoreServer
}

type TunnelService struct {
	pb.UnimplementedTunnelServiceServer
}

func StartGrpcServer(listenAddressG string, service string) error {

	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return err
	}
	s := grpc.NewServer()
	if service == "core" {
		pb.RegisterCoreServer(s, &CoreService{})
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
	}()
	return nil
}
func StartCoreGrpcServer(listenAddressG string) error {
	return StartGrpcServer(listenAddressG, "core")
}

func StartHelloGrpcServer(listenAddressG string) error {
	return StartGrpcServer(listenAddressG, "hello")
}

func StartTunnelGrpcServer(listenAddressG string) error {
	return StartGrpcServer(listenAddressG, "tunnel")
}
