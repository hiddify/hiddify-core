package adminbgservice

import (
	context "context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/hcore/adminbg"
	hutils "github.com/hiddify/hiddify-core/v2/hutils"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const adminBgServicePort uint16 = 18020

var (
	adminBgServiceAddress = fmt.Sprintf("127.0.0.1:%d", adminBgServicePort)
	adminBgServiceRunning = false
)

func isSupportedOS() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "linux"
}

func ActivateAdminBgService(opt *AdminBgStartRequest) error {
	adminBgServiceRunning = true
	return startTunnelRequestWithFailover(opt, true)
}

func DeactivateAdminBgServiceForce() error {
	return stopTunnelRequest()
}

func DeactivateAdminBgService() error {
	if adminBgServiceRunning {
		err := stopTunnelRequest()
		if err != nil {
			adminBgServiceRunning = false
		}
		return err
	} else {
		return stopTunnelRequest()
	}
}

func startTunnelRequestWithFailover(opt *AdminBgStartRequest, installService bool) error {
	res, err := startTunnelRequest(opt, installService)
	fmt.Printf("Start Tunnel Result: %v\n", res)
	if err != nil {
		fmt.Printf("Start Tunnel Failed! Stopping hcore... err=%v\n", err)
		// StopAndAlert(pb.MessageType.MessageType_UNEXPECTED_ERROR, "Start Tunnel Failed! Stopping...")
	}
	return err
}

func startTunnelRequest(opt *AdminBgStartRequest, installService bool) (bool, error) {
	if !hutils.IsPortInUse(adminBgServicePort) {
		if installService {
			return runTunnelService(opt)
		}
		return false, fmt.Errorf("service is not running")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, ctx, err := dialTunnel(ctx)
	if err != nil {
		log.Printf("did not connect: %v", err)
		return false, err
	}
	defer conn.Close()
	c := NewHiddifyAdminBGClient(conn)
	_, _ = c.Stop(ctx, &hcommon.Empty{})
	res, err := c.Start(ctx, opt)
	if err != nil {
		log.Printf("could not greet: %+v %+v", res, err)

		if installService {
			ExitAdminBgService()
			return runTunnelService(opt)
		}
		return false, err
	}

	return true, nil
}

func stopTunnelRequest() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	conn, ctx, err := dialTunnel(ctx)
	if err != nil {
		log.Printf("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	c := NewHiddifyAdminBGClient(conn)
	res, err := c.Stop(ctx, &hcommon.Empty{})
	if err != nil {
		log.Printf("did not Stopped: %v %v", res, err)
		_, _ = c.Stop(ctx, &hcommon.Empty{})
		return err
	}
	return nil
}

func ExitAdminBgService() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	conn, ctx, err := dialTunnel(ctx)
	if err != nil {
		log.Printf("did not connect: %v", err)
		return false, err
	}
	defer conn.Close()
	res, err := NewHiddifyAdminBGClient(conn).Exit(ctx, &hcommon.Empty{})
	if err != nil {
		log.Printf("did not exit: %v %v", res, err)
		return false, err
	}
	return true, nil
}

// dialTunnel attaches the control secret when the service was installed with one (macOS).
func dialTunnel(ctx context.Context) (*grpc.ClientConn, context.Context, error) {
	conn, err := grpc.Dial(adminBgServiceAddress, grpc.WithInsecure())
	if err != nil {
		return nil, ctx, err
	}
	if secret, err := LoadControlSecret(); err == nil {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(adminbg.SecretHeader, secret))
	}
	return conn, ctx, nil
}

func runTunnelService(opt *AdminBgStartRequest) (bool, error) {
	executablePath := getTunnelServicePath()
	fmt.Printf("Executable path is %s", executablePath)
	out, err := hutils.ExecuteCmd(executablePath, false, "tunnel", "install")
	fmt.Println("Shell command executed:", out, err)
	if err != nil {
		out, err = hutils.ExecuteCmd(executablePath, true, "tunnel", "run")
		fmt.Println("Shell command executed without flag:", out, err)
	}
	if err == nil {
		<-time.After(1 * time.Second) // wait until service loaded completely
	}
	return startTunnelRequest(opt, false)
}

func getTunnelServicePath() string {
	var fullPath string
	exePath, _ := os.Executable()
	binFolder := filepath.Dir(exePath)
	switch runtime.GOOS {
	case "windows":
		fullPath = "HiddifyCli.exe"
	case "darwin":
		fallthrough
	default:
		fullPath = "HiddifyCli"
	}

	abspath, _ := filepath.Abs(filepath.Join(binFolder, fullPath))
	return abspath
}
