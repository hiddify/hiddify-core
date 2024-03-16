package v2

import (
	"context"

	"github.com/hiddify/libcore/config"
	pb "github.com/hiddify/libcore/hiddifyrpc"
)

func (s *CoreService) GenerateWarpConfig(ctx context.Context, in *pb.GenerateWarpConfigRequest) (*pb.WarpGenerationResponse, error) {
	return GenerateWarpConfig(in)
}
func GenerateWarpConfig(in *pb.GenerateWarpConfigRequest) (*pb.WarpGenerationResponse, error) {
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
