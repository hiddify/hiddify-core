package v2

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
)

var logger service.Logger

type hiddifyNext struct{}

var port int = 18020

func (m *hiddifyNext) Start(s service.Service) error {
	_, err := StartTunnelGrpcServer(fmt.Sprintf("127.0.0.1:%d", port))
	return err
}

func (m *hiddifyNext) Stop(s service.Service) error {
	_, err := Stop()
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

func StartTunnelService(goArg string) (int, string) {
	svcConfig := &service.Config{
		Name:        "HiddifyTunnelService",
		DisplayName: "Hiddify Tunnel Service",
		Arguments:   []string{"tunnel", "run"},
		Description: "This is a bridge for tunnel",
		Option: map[string]interface{}{
			"RunAtLoad":        true,
			"WorkingDirectory": getCurrentExecutableDirectory(),
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
			fmt.Printf(out)
		}
		return 0, out
	} else {
		out := fmt.Sprintf("Error: %v", err)
		if dolog {
			log.Printf(out)
		}
		return 2, out
	}
}
