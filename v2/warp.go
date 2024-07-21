package v2

import (
	"context"

	"github.com/hiddify/hiddify-core/config"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
)

func (s *CoreService) GenerateWarpConfig(ctx context.Context, in *pb.GenerateWarpConfigRequest) (*pb.WarpGenerationResponse, error) {
	return GenerateWarpConfig(in)
}
func GenerateWarpConfig(in *pb.GenerateWarpConfigRequest) (*pb.WarpGenerationResponse, error) {
	identity, log, wg, err := config.GenerateWarpInfo(in.LicenseKey, in.AccountId, in.AccessToken)
	if err != nil {
		return nil, err
	}
	return &pb.WarpGenerationResponse{
		Account: &pb.WarpAccount{
			AccountId:   identity.ID,
			AccessToken: identity.Token,
		},
		Config: &pb.WarpWireguardConfig{
			PrivateKey:       wg.PrivateKey,
			LocalAddressIpv4: wg.LocalAddressIPv4,
			LocalAddressIpv6: wg.LocalAddressIPv6,
			PeerPublicKey:    wg.PeerPublicKey,
			ClientId:         wg.ClientID,
		},
		Log: log,
	}, nil
}
