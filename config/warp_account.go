package config

import "encoding/json"

type WarpAccount struct {
	AccountID   string `json:"account-id"`
	AccessToken string `json:"access-token"`
}

func GenerateWarpAccount(licenseKey string, accountId string, accessToken string) (string, error) {
	data, _, _, err := GenerateWarpInfo(licenseKey, accountId, accessToken)
	if err != nil {
		return "", err
	}
	warpAccount := WarpAccount{
		AccountID:   data.AccountID,
		AccessToken: data.AccessToken,
	}
	accountJson, err := json.Marshal(warpAccount)
	if err != nil {
		return "", err
	}
	return string(accountJson), nil
}
