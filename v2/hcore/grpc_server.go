package hcore

/*
#include "stdint.h"
*/

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"

	"github.com/hiddify/hiddify-core/v2/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/hiddify/hiddify-core/v2/common"
	"github.com/hiddify/hiddify-core/v2/common/utils"

	"github.com/hiddify/hiddify-core/v2/db"
)

type CoreService struct {
	UnimplementedCoreServer
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
		RegisterCoreServer(s, &CoreService{})
		// pb.RegisterExtensionHostServiceServer(s, &extension.ExtensionHostService{})
	} else if service == "hello" {
		// RegisterHelloServer(s, &hello.HelloService{})
	} else if service == "tunnel" {
		// RegisterTunnelServiceServer(s, &TunnelService{})
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

var (
	certpair   *utils.CertificatePair
	grpcServer map[SetupMode]*grpc.Server = make(map[SetupMode]*grpc.Server)
	caCertPool                            = x509.NewCertPool()
)

// StartGrpcServerByMode starts a gRPC server on the specified address with mTLS.
func StartGrpcServerByMode(listenAddressG string, mode SetupMode) (*grpc.Server, error) {
	// Fetch the server private key and public key from the database
	if grpcServer[mode] != nil {
		Log(LogLevel_WARNING, LogType_CORE, "grpcServer already started")
		return grpcServer[mode], nil
	}
	table := db.GetTable[common.AppSettings]()
	grpcServerPrivateKey, err := table.Get("grpc_server_private_key")
	grpcServerPublicKey, err2 := table.Get("grpc_server_public_key")
	if err != nil || err2 != nil {
		Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("failed to get grpc_server_private_key and grpc_server_public_key from database: %v %v\n", err, err2))
		certpair, err = utils.GenerateCertificatePair()
		if err != nil {
			Log(LogLevel_ERROR, LogType_CORE, fmt.Sprintf("failed to generate certificate pair: %v", err))

			return nil, err
		}
		table.UpdateInsert(
			&common.AppSettings{Id: "grpc_server_public_key", Value: certpair.Certificate},
			&common.AppSettings{Id: "grpc_server_private_key", Value: certpair.PrivateKey},
		)
	} else {
		certpair = &utils.CertificatePair{
			Certificate: grpcServerPublicKey.Value.([]byte),
			PrivateKey:  grpcServerPrivateKey.Value.([]byte),
		}
	}

	// Load server certificate and private key
	serverCert, err := tls.X509KeyPair(certpair.Certificate, certpair.PrivateKey)
	if err != nil {
		Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("failed to load server certificate and key: %v\n", err))

		return nil, err
	}

	// Create TLS credentials for the gRPC server
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // Enforce mutual TLS (mTLS)
		ClientCAs:    caCertPool,                     // Client CAs to verify client certificates
	}

	// Create a new gRPC server with TLS credentials
	creds := credentials.NewTLS(tlsConfig)
	if mode == GRPC_BACKGROUND_INSECURE || mode == GRPC_NORMAL_INSECURE {
		grpcServer[mode] = grpc.NewServer()
	} else {
		grpcServer[mode] = grpc.NewServer(grpc.Creds(creds))
	}
	// Register your gRPC service here
	RegisterCoreServer(grpcServer[mode], &CoreService{})
	hello.RegisterHelloServer(grpcServer[mode], &hello.HelloService{})

	// Listen on the provided address
	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		Log(LogLevel_ERROR, LogType_CORE, fmt.Sprintf("failed to listen on %s: %v\n", listenAddressG, err))
		return nil, err
	}
	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("grpcServer started on %s\n", listenAddressG))
	log.Printf("Server listening on %s", listenAddressG)

	// Run the server in a goroutine
	go func() {
		if err := grpcServer[mode].Serve(lis); err != nil {
			Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("failed to serve: %v\n", err))
		}
		Log(LogLevel_DEBUG, LogType_CORE, "Server stopped")
	}()

	return grpcServer[mode], nil
}

// GetGrpcServerPublicKey returns the gRPC server's public key.
func GetGrpcServerPublicKey() []byte {
	return certpair.Certificate
}

// AddGrpcClientPublicKey adds a client's public key to the CA pool for verification.
func AddGrpcClientPublicKey(clientPublicKey []byte) error {
	block, _ := pem.Decode(clientPublicKey)
	if block == nil || block.Type != "PUBLIC KEY" {
		return fmt.Errorf("failed to decode client public key")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse client public key: %v", err)
		}
		cert = &x509.Certificate{
			PublicKey: pubKey,
		}
	}
	caCertPool.AddCert(cert)

	return nil
}

func CloseGrpcServer() {
	for mode := range grpcServer {
		grpcServer[mode].Stop()
		grpcServer[mode] = nil
	}
}
