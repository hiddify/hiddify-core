package tunnelservice

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/hiddify/hiddify-core/v2/hcore/tunnel"
	"github.com/kardianos/service"
	"google.golang.org/grpc"
)

var logger service.Logger

type hiddifyNext struct {
	tunnelService *TunnelService
	secret        string
}

var port int = 18020

func allowTunnelMethod(method string) error {
	switch method {
	case TunnelService_Start_FullMethodName, TunnelService_Stop_FullMethodName,
		TunnelService_Status_FullMethodName, TunnelService_Exit_FullMethodName:
		return nil
	}
	return tunnel.AllowCoreMethod(method)
}

func (m *hiddifyNext) StartTunnelGrpcServer(listenAddressG string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return nil, err
	}
	var serverOptions []grpc.ServerOption
	if m.secret != "" {
		serverOptions = tunnel.ServerOptions(m.secret, allowTunnelMethod)
	}
	s := grpc.NewServer(serverOptions...)
	m.tunnelService = &TunnelService{}
	RegisterTunnelServiceServer(s, m.tunnelService)
	if m.secret != "" {
		tunnel.RegisterControl(s)
	}

	log.Printf("Server listening on %s", listenAddressG)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
		}
		log.Printf("Server stopped")
	}()
	return s, nil
}

func (m *hiddifyNext) Start(s service.Service) error {
	if secret, err := LoadControlSecret(); err == nil {
		m.secret = secret
	} else if os.IsNotExist(err) {
		log.Printf("tunnel control secret missing; gRPC is unauthenticated until reinstall")
	} else {
		return err
	}
	_, err := m.StartTunnelGrpcServer(fmt.Sprintf("127.0.0.1:%d", port))
	return err
}

func (m *hiddifyNext) Stop(s service.Service) error {
	if m.tunnelService == nil {
		return nil
	}
	_, err := m.tunnelService.Stop(nil, nil)
	if err != nil {
		return nil
	}
	return nil
}

func getCurrentExecutableDirectory() string {
	executablePath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(executablePath)
}

func serviceDataDirectory() string {
	return dataDirectoryFor(getCurrentExecutableDirectory())
}

func dataDirectoryFor(directory string) string {
	if filepath.Base(directory) == "MacOS" && filepath.Base(filepath.Dir(directory)) == "Contents" &&
		filepath.Ext(filepath.Dir(filepath.Dir(directory))) == ".app" {
		return filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(directory))), "data")
	}
	return directory
}

func StartTunnelService(goArg string) (int, string) {
	dataDirectory := serviceDataDirectory()
	if err := os.MkdirAll(dataDirectory, 0o755); err != nil {
		return 3, fmt.Sprintf("Error: %v", err)
	}
	if goArg == "install" {
		owner := os.Getenv("HIDDIFY_TUNNEL_OWNER")
		if _, err := EnsureControlSecret(owner); err != nil {
			return 3, fmt.Sprintf("Error: %v", err)
		}
	}
	svcConfig := &service.Config{
		Name:             "HiddifyTunnelService",
		DisplayName:      "Hiddify Tunnel Service",
		Arguments:        []string{"tunnel", "run"},
		Description:      "Runs Hiddify's VPN engine with administrator rights",
		WorkingDirectory: dataDirectory,
		Option: map[string]interface{}{
			"RunAtLoad": true,
		},
	}

	prg := &hiddifyNext{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return 1, fmt.Sprintf("Error: %v", err)
	}

	if len(goArg) > 0 && goArg != "run" {
		return control(s, goArg)
	}

	if err := os.Chdir(dataDirectory); err != nil {
		return 3, fmt.Sprintf("Error: %v", err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
		return 3, fmt.Sprintf("Error: %v", err)
	}
	return 0, ""
}

func control(s service.Service, goArg string) (int, string) {
	dolog := false
	var err error
	status, serr := s.Status()
	if dolog {
		fmt.Printf("Current Status: %+v %+v!\n", status, serr)
	}
	switch goArg {
	case "uninstall":
		if status == service.StatusRunning {
			s.Stop()
		}
		err = s.Uninstall()
	case "start":
		if status == service.StatusRunning {
			return 0, "Tunnel Service Already Running."
		} else if status == service.StatusUnknown {
			s.Uninstall()
			s.Install()
			status, serr = s.Status()
			_ = serr
		}
		if status != service.StatusRunning {
			err = s.Start()
		}
	case "install":
		s.Uninstall()
		err = s.Install()
		status, serr = s.Status()
		_ = serr
		if status != service.StatusRunning {
			err = s.Start()
		}
	case "stop":
		if status == service.StatusStopped {
			return 0, "Tunnel Service Already Stopped."
		}
		err = s.Stop()
	default:
		err = service.Control(s, goArg)
	}
	if err == nil {
		return 0, fmt.Sprintf("Tunnel Service %sed Successfully.", goArg)
	}
	return 2, fmt.Sprintf("Error: %v", err)
}
