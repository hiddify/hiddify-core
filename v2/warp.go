package v2

import (
	"context"

	"github.com/hiddify/libcore/config"
	pb "github.com/hiddify/libcore/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
)

func (s *server) GenerateWarpConfig(ctx context.Context, in *pb.GenerateWarpConfigRequest) (*pb.WarpGenerationResponse, error) {
	account, log, wg, err := config.GenerateWarpInfo(in.LicenseKey, in.AccountId, in.AccessToken)
	if err != nil {
		return nil, err
	}
	return &pb.WarpGenerationResponse{
		Account: &pb.WarpAccount{
			AccountId:   account.AccountID,
			AccessToken: account.AccessToken,
		},
		Config: &pb.WarpWireguardConfig{
			PrivateKey:       wg.PrivateKey,
			LocalAddressIpv4: wg.LocalAddressIPv4,
			LocalAddressIpv6: wg.LocalAddressIPv6,
			PeerPublicKey:    wg.PeerPublicKey,
		},
		Log: log,
	}, nil
}

// Implement the GetSystemProxyStatus method
func (s *server) GetSystemProxyStatus(ctx context.Context, empty *pb.Empty) (*pb.SystemProxyStatus, error) {
	status, err := libbox.NewStandaloneCommandClient().GetSystemProxyStatus()

	if err != nil {
		return nil, err
	}

	return &pb.SystemProxyStatus{
		Available: status.Available,
		Enabled:   status.Enabled,
	}, nil
}

// Implement the SetSystemProxyEnabled method
func (s *server) SetSystemProxyEnabled(ctx context.Context, in *pb.SetSystemProxyEnabledRequest) (*pb.Response, error) {
	err := libbox.NewStandaloneCommandClient().SetSystemProxyEnabled(in.IsEnabled)

	if err != nil {
		return nil, err
	}

	if err != nil {
		return &pb.Response{
			ResponseCode: pb.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}

	return &pb.Response{
		ResponseCode: pb.ResponseCode_OK,
		Message:      "",
	}, nil

}
