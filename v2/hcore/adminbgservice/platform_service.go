package adminbgservice

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hiddify/hiddify-core/v2/hcore/adminbg"
	"github.com/kardianos/service"
	"google.golang.org/grpc"
)

var logger service.Logger

type hiddifyNext struct {
	adminBg *hiddifyAdminBG
	secret  string
}

var port int = 18020

func allowAdminBgMethod(method string) error {
	switch method {
	case HiddifyAdminBG_Start_FullMethodName, HiddifyAdminBG_Stop_FullMethodName,
		HiddifyAdminBG_Status_FullMethodName, HiddifyAdminBG_Exit_FullMethodName:
		return nil
	}
	return adminbg.AllowCoreMethod(method)
}

func (m *hiddifyNext) StartAdminBgGrpcServer(listenAddressG string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return nil, err
	}
	var serverOptions []grpc.ServerOption
	if m.secret != "" {
		serverOptions = adminbg.ServerOptions(m.secret, allowAdminBgMethod)
	}
	s := grpc.NewServer(serverOptions...)
	m.adminBg = &hiddifyAdminBG{}
	RegisterHiddifyAdminBGServer(s, m.adminBg)
	if m.secret != "" {
		adminbg.RegisterControl(s)
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
	if err := WriteRunningBinaryStamp(); err != nil {
		return fmt.Errorf("record tunnel binary stamp: %w", err)
	}
	if secret, err := LoadControlSecret(); err == nil {
		m.secret = secret
	} else if os.IsNotExist(err) {
		log.Printf("tunnel control secret missing; gRPC is unauthenticated until reinstall")
	} else {
		return err
	}
	_, err := m.StartAdminBgGrpcServer(fmt.Sprintf("127.0.0.1:%d", port))
	return err
}

func (m *hiddifyNext) Stop(s service.Service) error {
	if m.adminBg == nil {
		return nil
	}
	_, err := m.adminBg.Stop(nil, nil)
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
	// Windows/Linux: keep cache and the control secret under the OS data dir, never beside the
	// install tree in Program Files (writes there look suspicious and often fail).
	if runtime.GOOS != "darwin" {
		return controlDataDir()
	}
	return dataDirectoryFor(getCurrentExecutableDirectory())
}

func dataDirectoryFor(directory string) string {
	if filepath.Base(directory) == "MacOS" && filepath.Base(filepath.Dir(directory)) == "Contents" &&
		filepath.Ext(filepath.Dir(filepath.Dir(directory))) == ".app" {
		return filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(directory))), "data")
	}
	return directory
}

func installOwner() string {
	if owner := os.Getenv("HIDDIFY_ADMIN_BG_OWNER"); owner != "" {
		return owner
	}
	if owner := os.Getenv("HIDDIFY_TUNNEL_OWNER"); owner != "" {
		return owner
	}
	if owner := os.Getenv("USERNAME"); owner != "" {
		return owner
	}
	return os.Getenv("USER")
}

// osServiceName is the OS service id passed to kardianos/service.
// On Linux systemd, that library writes Name+".service" (do not include the suffix here).
func osServiceName() string {
	return "HiddifyAdminBG"
}

func serviceRunArguments() []string {
	if runtime.GOOS == "darwin" {
		return []string{"--tunnel", "run"}
	}
	return []string{"admin_bgservice", "run"}
}

func StartAdminBgService(goArg string) (int, string) {
	dataDirectory := serviceDataDirectory()
	if err := os.MkdirAll(dataDirectory, 0o755); err != nil {
		return 3, fmt.Sprintf("Error: %v", err)
	}
	if goArg == "install" {
		if _, err := EnsureControlSecret(installOwner()); err != nil {
			return 3, fmt.Sprintf("Error: %v", err)
		}
	}
	// Windows/Linux: HiddifyCli admin_bgservice run.
	// macOS: the copied .app must get --tunnel so AppDelegate runs parseCli instead of the Flutter UI.
	svcConfig := &service.Config{
		Name:             osServiceName(),
		DisplayName:      "Hiddify VPN",
		Arguments:        serviceRunArguments(),
		Description:      "Hiddify virtual private network",
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
