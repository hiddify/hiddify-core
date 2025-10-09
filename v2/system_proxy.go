package v2

import (
	"context"
	"os"

	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
)

func (s *CoreService) GetSystemProxyStatus(ctx context.Context, empty *pb.Empty) (*pb.SystemProxyStatus, error) {
	return GetSystemProxyStatus(ctx, empty)
}
func GetSystemProxyStatus(ctx context.Context, empty *pb.Empty) (*pb.SystemProxyStatus, error) {
	status, err := libbox.NewStandaloneCommandClient().GetSystemProxyStatus()

	if err != nil {
		return nil, err
	}

	return &pb.SystemProxyStatus{
		Available: status.Available,
		Enabled:   status.Enabled,
	}, nil
}

func (s *CoreService) SetSystemProxyEnabled(ctx context.Context, in *pb.SetSystemProxyEnabledRequest) (*pb.Response, error) {
	return SetSystemProxyEnabled(ctx, in)
}
func SetSystemProxyEnabled(ctx context.Context, in *pb.SetSystemProxyEnabledRequest) (*pb.Response, error) {
	err := libbox.NewStandaloneCommandClient().SetSystemProxyEnabled(in.IsEnabled)

	if err != nil {
		return &pb.Response{
			ResponseCode: pb.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}

	// Also clear common proxy environment variables if disabling
	if !in.IsEnabled {
		keys := []string{
			"http_proxy", "https_proxy", "ftp_proxy", "all_proxy",
			"HTTP_PROXY", "HTTPS_PROXY", "FTP_PROXY", "ALL_PROXY",
			"no_proxy", "NO_PROXY",
		}
		for _, k := range keys {
			_ = os.Unsetenv(k)
		}
	}

	return &pb.Response{
		ResponseCode: pb.ResponseCode_OK,
		Message:      "",
	}, nil

}
