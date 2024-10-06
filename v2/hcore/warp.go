package hcore

import (
	"context"

	"github.com/hiddify/hiddify-core/config"
)

func (s *CoreService) GenerateWarpConfig(ctx context.Context, in *GenerateWarpConfigRequest) (*WarpGenerationResponse, error) {
	return GenerateWarpConfig(in)
}

func GenerateWarpConfig(in *GenerateWarpConfigRequest) (*WarpGenerationResponse, error) {
	identity, log, wg, err := config.GenerateWarpInfo(in.LicenseKey, in.AccountId, in.AccessToken)
	if err != nil {
		return nil, err
	}
	return &WarpGenerationResponse{
		Account: &WarpAccount{
			AccountId:   identity.ID,
			AccessToken: identity.Token,
		},
		Config: &WarpWireguardConfig{
			PrivateKey:       wg.PrivateKey,
			LocalAddressIpv4: wg.LocalAddressIPv4,
			LocalAddressIpv6: wg.LocalAddressIPv6,
			PeerPublicKey:    wg.PeerPublicKey,
			ClientId:         wg.ClientID,
		},
		Log: log,
	}, nil
}
