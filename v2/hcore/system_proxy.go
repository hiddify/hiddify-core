package hcore

import (
	"context"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/experimental/libbox"
)

func (s *CoreService) GetSystemProxyStatus(ctx context.Context, empty *hcommon.Empty) (*SystemProxyStatus, error) {
	return GetSystemProxyStatus(ctx, empty)
}

func GetSystemProxyStatus(ctx context.Context, empty *hcommon.Empty) (*SystemProxyStatus, error) {
	status, err := libbox.NewStandaloneCommandClient().GetSystemProxyStatus()
	if err != nil {
		return nil, err
	}

	return &SystemProxyStatus{
		Available: status.Available,
		Enabled:   status.Enabled,
	}, nil
}

func (s *CoreService) SetSystemProxyEnabled(ctx context.Context, in *SetSystemProxyEnabledRequest) (*hcommon.Response, error) {
	return SetSystemProxyEnabled(ctx, in)
}

func SetSystemProxyEnabled(ctx context.Context, in *SetSystemProxyEnabledRequest) (*hcommon.Response, error) {
	err := libbox.NewStandaloneCommandClient().SetSystemProxyEnabled(in.IsEnabled)
	if err != nil {
		return &hcommon.Response{
			Code:    hcommon.ResponseCode_FAILED,
			Message: err.Error(),
		}, err
	}

	return &hcommon.Response{
		Code:    hcommon.ResponseCode_OK,
		Message: "",
	}, nil
}
