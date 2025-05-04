package config

import (
	"encoding/json"
)

type WarpAccount struct {
	AccountID   string `json:"account-id,omitempty"`
	AccessToken string `json:"access-token,omitempty"`
}

type WarpWireguardConfig struct {
	PrivateKey       string `json:"private-key,omitempty"`
	LocalAddressIPv4 string `json:"local-address-ipv4,omitempty"`
	LocalAddressIPv6 string `json:"local-address-ipv6,omitempty"`
	PeerPublicKey    string `json:"peer-public-key,omitempty"`
	ClientID         string `json:"client-id,omitempty"`
}

type WarpGenerationResponse struct {
	WarpAccount
	Log    string              `json:"log"`
	Config WarpWireguardConfig `json:"config"`
}

func GenerateWarpAccount(licenseKey string, accountId string, accessToken string) (string, error) {
	identity, log, wg, err := GenerateWarpInfo(licenseKey, accountId, accessToken)
	if err != nil {
		return "", err
	}

	warpAccount := WarpAccount{
		AccountID:   identity.ID,
		AccessToken: identity.Token,
	}
	warpConfig := WarpWireguardConfig{
		PrivateKey:       wg.PrivateKey,
		LocalAddressIPv4: wg.LocalAddressIPv4,
		LocalAddressIPv6: wg.LocalAddressIPv6,
		PeerPublicKey:    wg.PeerPublicKey,
		ClientID:         wg.ClientID,
	}
	response := WarpGenerationResponse{warpAccount, log, warpConfig}

	responseJson, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	return string(responseJson), nil
}
