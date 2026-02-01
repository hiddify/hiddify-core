package hcore

/*
#include "stdint.h"
*/

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"

	"net"
	"os"
	"strconv"
	"strings"
	sync "sync"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/db"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/hello"
	hutils "github.com/hiddify/hiddify-core/v2/hutils"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	E "github.com/sagernet/sing/common/exceptions"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
)

type CoreService struct {
	UnimplementedCoreServer
}

func Setup(params *SetupRequest, platformInterface libbox.PlatformInterface) error {
	defer config.DeferPanicToError("setup", func(err error) {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		<-time.After(5 * time.Second)
	})
	mu.Lock()
	defer mu.Unlock()
	if grpcServer[params.Mode] != nil {
		Log(LogLevel_WARNING, LogType_CORE, "grpcServer already started")
		return nil
	}
	static.BaseContext = libbox.BaseContext(platformInterface)
	static.debug = params.Debug
	static.globalPlatformInterface = platformInterface
	tcpConn := true // runtime.GOOS == "windows" // TODO add TVOS
	libbox.Setup(
		&libbox.SetupOptions{
			BasePath:    params.BasePath,
			WorkingPath: params.WorkingDir,
			TempPath:    params.TempDir,
			// IsTVOS:          !tcpConn,
			FixAndroidStack: params.FixAndroidStack,
			LogMaxLines:     100,
			Debug:           params.Debug,
		})

	hutils.RedirectStderr(fmt.Sprint(params.WorkingDir, "/data/stderr", params.Mode, ".log"))

	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("libbox.Setup success %s %s %s %v", params.BasePath, params.WorkingDir, params.TempDir, tcpConn))

	sWorkingPath = params.WorkingDir
	os.Chdir(sWorkingPath)
	sTempPath = params.TempDir
	sUserID = os.Getuid()
	sGroupID = os.Getgid()

	var defaultWriter io.Writer
	if !params.Debug {
		defaultWriter = io.Discard
	}
	factory, err := log.New(
		log.Options{
			DefaultWriter: defaultWriter,
			BaseTime:      time.Now(),
			Observable:    true,
			// Options: option.LogOptions{
			// 	Disabled: false,
			// 	Level:    "trace",
			// 	Output:   "stdout",
			// },
		})
	static.CoreLogFactory = factory

	if err != nil {
		return E.Cause(err, "create logger")
	}

	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("StartGrpcServerByMode %s %d\n", params.Listen, params.Mode))
	switch params.Mode {
	case SetupMode_OLD:
		statusPropagationPort = int64(params.FlutterStatusPort)
	// case SetupMode_GRPC_BACKGROUND_INSECURE:
	default:
		_, err := StartGrpcServerByMode(params.Listen, params.Mode)
		if err != nil {
			return err
		}
	}
	settings := db.GetTable[hcommon.AppSettings]()
	val, err := settings.Get("HiddifySettingsJson")
	Log(LogLevel_DEBUG, LogType_CORE, "HiddifySettingsJson", val, err)
	if val == nil || err != nil {
		// if params.Mode == SetupMode_GRPC_BACKGROUND_INSECURE {
		_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: ""}, false)
		if err != nil {
			Log(LogLevel_ERROR, LogType_CORE, E.Cause(err, "ChangeHiddifySettings").Error())
		}
	} else {
		// settings := db.GetTable[hcommon.AppSettings]()
		_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: val.Value.(string)}, false)
		if err != nil {
			Log(LogLevel_ERROR, LogType_CORE, E.Cause(err, "ChangeHiddifySettings").Error())
		}

	}
	return InitHiddifyService()
}

func StartGrpcServer(listenAddressG string, service string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		log.Error("failed to listen: %v", err)
		return nil, err
	}
	s := grpc.NewServer()
	if service == "core" {
		// Setup("./tmp/", "./tmp", "./tmp", 11111, false)
		RegisterCoreServer(s, &CoreService{})
		// pb.RegisterExtensionHostServiceServer(s, &extension.ExtensionHostService{})
	} else if service == "hello" {
		// RegisterHelloServer(s, &hello.HelloService{})
	} else if service == "tunnel" {
		// RegisterTunnelServiceServer(s, &TunnelService{})
	}
	log.Info("Server listening on %s", listenAddressG)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Error("failed to serve: %v", err)
		}
		log.Info("Server stopped")
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
	certpair   *hutils.CertificatePair
	grpcServer map[SetupMode]*grpc.Server = make(map[SetupMode]*grpc.Server)
	caCertPool                            = x509.NewCertPool()
	mu                                    = sync.Mutex{}
)

// StartGrpcServerByMode starts a gRPC server on the specified address with mTLS.
func StartGrpcServerByMode(listenAddressG string, mode SetupMode) (*grpc.Server, error) {
	// Validate the listen address
	if !strings.Contains(listenAddressG, ":") {
		return nil, fmt.Errorf("invalid listen address (no port): %s", listenAddressG)
	}
	// Convert the port from string to uint16
	portStr := strings.Split(listenAddressG, ":")[1]
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to convert port %s to uint16: %v", portStr, err)
	}
	if hutils.IsPortInUse(uint16(port)) {
		return nil, fmt.Errorf("port %s is already in use", portStr)
	}
	// Fetch the server private key and public key from the database
	if _, exists := grpcServer[mode]; exists {
		Log(LogLevel_WARNING, LogType_CORE, "grpcServer already started")
		return grpcServer[mode], nil
	}

	if mode == SetupMode_GRPC_BACKGROUND_INSECURE || mode == SetupMode_GRPC_NORMAL_INSECURE {
		grpcServer[mode] = grpc.NewServer()
	} else {
		table := db.GetTable[hcommon.AppSettings]()
		Log(LogLevel_DEBUG, LogType_CORE, table)
		grpcServerPrivateKey, err := table.Get("grpc_server_private_key")
		grpcServerPublicKey, err2 := table.Get("grpc_server_public_key")
		if err != nil || err2 != nil {
			Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("failed to get grpc_server_private_key and grpc_server_public_key from database: %v %v\n", err, err2))
			certpair, err = hutils.GenerateCertificatePair()
			if err != nil {
				Log(LogLevel_ERROR, LogType_CORE, fmt.Sprintf("failed to generate certificate pair: %v", err))

				return nil, err
			}
			table.UpdateInsert(
				&hcommon.AppSettings{Id: "grpc_server_public_key", Value: certpair.Certificate},
				&hcommon.AppSettings{Id: "grpc_server_private_key", Value: certpair.PrivateKey},
			)
		} else {
			certpair = &hutils.CertificatePair{
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
	log.Info("Server listening on %s", listenAddressG)

	// Run the server in a goroutine
	go func() {
		defer config.DeferPanicToError("grpcsetup", func(err error) {
			Log(LogLevel_FATAL, LogType_CORE, err.Error())
			<-time.After(5 * time.Second)
		})
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

func CloseGrpcServer(mode SetupMode) {
	mu.Lock()
	defer mu.Unlock()
	if server, ok := grpcServer[mode]; ok && server != nil {
		server.Stop()
		delete(grpcServer, mode)
	}
}
