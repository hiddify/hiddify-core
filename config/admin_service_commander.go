package config

import (
	context "context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	pb "github.com/hiddify/libcore/hiddifyrpc"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
	grpc "google.golang.org/grpc"
)

const (
	serviceURL    = "http://localhost:18020"
	startEndpoint = "/start"
	stopEndpoint  = "/stop"
)

var tunnelServiceRunning = false

func isSupportedOS() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "linux"
}
func ActivateTunnelService(opt ConfigOptions) (bool, error) {
	tunnelServiceRunning = true
	// if !isSupportedOS() {
	// 	return false, E.New("Unsupported OS: " + runtime.GOOS)
	// }

	go startTunnelRequestWithFailover(opt, true)
	return true, nil
}

func DeactivateTunnelService() (bool, error) {
	// if !isSupportedOS() {
	// 	return true, nil
	// }
	if tunnelServiceRunning {
		stopTunnelRequest()
	}
	tunnelServiceRunning = false

	return true, nil
}

func startTunnelRequestWithFailover(opt ConfigOptions, installService bool) {
	res, err := startTunnelRequest(opt, installService)
	fmt.Printf("Start Tunnel Result: %v\n", res)
	if err != nil {

		fmt.Printf("Start Tunnel Failed! Stopping core... err=%v\n", err)
		// StopAndAlert(pb.MessageType.MessageType_UNEXPECTED_ERROR, "Start Tunnel Failed! Stopping...")

	}
}

func startTunnelRequest(opt ConfigOptions, installService bool) (bool, error) {
	conn, err := grpc.Dial("127.0.0.1:18020", grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewTunnelServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	_, err = c.Start(ctx, &pb.TunnelStartRequest{
		Ipv6:                   opt.IPv6Mode == option.DomainStrategy(dns.DomainStrategyUseIPv4),
		ServerPort:             int32(opt.InboundOptions.MixedPort),
		StrictRoute:            opt.InboundOptions.StrictRoute,
		EndpointIndependentNat: true,
		Stack:                  opt.InboundOptions.TUNStack,
	})
	if err != nil {
		log.Printf("could not greet: %v", err)

		if installService {
			return runTunnelService(opt)
		}
		return false, err
	}

	return true, nil
}

func stopTunnelRequest() (bool, error) {
	conn, err := grpc.Dial("127.0.0.1:18020", grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewTunnelServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	_, err = c.Stop(ctx, &pb.Empty{})
	if err != nil {
		return false, err
	}

	return true, nil
}

func runTunnelService(opt ConfigOptions) (bool, error) {
	executablePath := getTunnelServicePath()
	fmt.Printf("Executable path is %s", executablePath)
	out, err := ExecuteCmd(executablePath, false, "tunnel", "install")
	fmt.Println("Shell command executed:", out, err)
	if err != nil {
		out, err = ExecuteCmd(executablePath, true, "tunnel", "run")
		fmt.Println("Shell command executed without flag:", out, err)
	}
	if err == nil {
		<-time.After(1 * time.Second) //wait until service loaded completely
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
