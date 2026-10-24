package tunnelservice

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	"google.golang.org/grpc"
)

var logger service.Logger

type hiddifyNext struct {
	tunnelService *TunnelService
}

var port int = 18020

func (m *hiddifyNext) StartTunnelGrpcServer(listenAddressG string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return nil, err
	}
	s := grpc.NewServer()
	m.tunnelService = &TunnelService{}
	RegisterTunnelServiceServer(s, m.tunnelService)

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

func (m *hiddifyNext) Start(s service.Service) error {
	_, err := m.StartTunnelGrpcServer(fmt.Sprintf("127.0.0.1:%d", port))
	return err
}

func (m *hiddifyNext) Stop(s service.Service) error {
	_, err := m.tunnelService.Stop(nil, nil)
	if err != nil {
		return nil
	}
	// Stop should not block. Return with a few seconds.
	// <-time.After(time.Second * 1)
	return nil
}

func getCurrentExecutableDirectory() string {
	executablePath, err := os.Executable()
	if err != nil {
		return ""
	}

	// Extract the directory (folder) containing the executable
	executableDirectory := filepath.Dir(executablePath)

	return executableDirectory
}

// serviceDataDirectory is where the service keeps its files, such as the box's cache: next to the app
// when it runs from a macOS app bundle, whose contents must not change, else beside the executable.
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
	// The service starts in this directory, so it must exist from installation on.
	dataDirectory := serviceDataDirectory()
	if err := os.MkdirAll(dataDirectory, 0o755); err != nil {
		return 3, fmt.Sprintf("Error: %v", err)
	}
	svcConfig := &service.Config{
		Name:             "HiddifyTunnelService",
		DisplayName:      "Hiddify Tunnel Service",
		Arguments:        []string{"tunnel", "run"},
		Description:      "This is a bridge for tunnel",
		WorkingDirectory: dataDirectory,
		Option: map[string]interface{}{
			"RunAtLoad": true,
		},
	}

	prg := &hiddifyNext{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		// log.Printf("Error: %v", err)
		return 1, fmt.Sprintf("Error: %v", err)
	}

	if len(goArg) > 0 && goArg != "run" {
		return control(s, goArg)
	}

	// A service installed before it had a working directory starts in /, read-only on macOS.
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
		if dolog {
			fmt.Printf("Tunnel Service Uninstalled Successfully.\n")
		}
		err = s.Uninstall()
	case "start":
		if status == service.StatusRunning {
			if dolog {
				fmt.Printf("Tunnel Service Already Running.\n")
			}
			return 0, "Tunnel Service Already Running."
		} else if status == service.StatusUnknown {
			s.Uninstall()
			s.Install()
			status, serr = s.Status()
			if dolog {
				fmt.Printf("Check status again: %+v %+v!", status, serr)
			}
		}
		if status != service.StatusRunning {
			err = s.Start()
		}
	case "install":
		s.Uninstall()
		err = s.Install()
		status, serr = s.Status()
		if dolog {
			fmt.Printf("Check Status Again: %+v %+v", status, serr)
		}
		if status != service.StatusRunning {
			err = s.Start()
		}
	case "stop":
		if status == service.StatusStopped {
			if dolog {
				fmt.Printf("Tunnel Service Already Stopped.\n")
			}
			return 0, "Tunnel Service Already Stopped."
		}
		err = s.Stop()
	default:
		err = service.Control(s, goArg)
	}
	if err == nil {
		out := fmt.Sprintf("Tunnel Service %sed Successfully.", goArg)
		if dolog {
			fmt.Printf("%s", out)
		}
		return 0, out
	} else {
		out := fmt.Sprintf("Error: %v", err)
		if dolog {
			log.Printf("%s", out)
		}
		return 2, out
	}
}
