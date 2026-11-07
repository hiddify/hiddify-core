package adminbgservice

import (
	"fmt"

	"github.com/hiddify/hiddify-core/v2/hcore/adminbg"
	"google.golang.org/grpc"
)

// foregroundSecret is the control secret of a service run in the foreground (RunForeground): it
// has no secret file, the app that starts it passes the secret.
var foregroundSecret string

// ForegroundService is the VPN service run in the foreground by RunForeground.
type ForegroundService struct {
	server *grpc.Server
	next   *hiddifyNext
}

// RunForeground serves the VPN service's API on listen in this process, authenticated by secret,
// instead of as a system service: the AppImage's VPN, started as root through pkexec.
func RunForeground(listen string, secret string) (*ForegroundService, error) {
	if len(secret) < adminbg.MinSecretLength {
		return nil, fmt.Errorf("the control secret is too short")
	}
	foregroundSecret = secret
	next := &hiddifyNext{secret: secret}
	server, err := next.StartAdminBgGrpcServer(listen)
	if err != nil {
		return nil, err
	}
	return &ForegroundService{server: server, next: next}, nil
}

// Close stops the engine, restoring what it changed (TUN, routes), and the API.
func (f *ForegroundService) Close() {
	if f.next.adminBg != nil {
		f.next.adminBg.mutex.Lock()
		_ = f.next.adminBg.closeRunning()
		f.next.adminBg.mutex.Unlock()
	}
	f.server.Stop()
}
