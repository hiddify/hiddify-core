package hcore

import (
	"context"

	common "github.com/hiddify/hiddify-core/v2/common"
	"github.com/sagernet/sing-box/experimental/libbox"
)

func (s *CoreService) GetSystemProxyStatus(ctx context.Context, empty *common.Empty) (*SystemProxyStatus, error) {
	return GetSystemProxyStatus(ctx, empty)
}

func GetSystemProxyStatus(ctx context.Context, empty *common.Empty) (*SystemProxyStatus, error) {
	status, err := libbox.NewStandaloneCommandClient().GetSystemProxyStatus()
	if err != nil {
		return nil, err
	}

	return &SystemProxyStatus{
		Available: status.Available,
		Enabled:   status.Enabled,
	}, nil
}

func (s *CoreService) SetSystemProxyEnabled(ctx context.Context, in *SetSystemProxyEnabledRequest) (*common.Response, error) {
	return SetSystemProxyEnabled(ctx, in)
}

func SetSystemProxyEnabled(ctx context.Context, in *SetSystemProxyEnabledRequest) (*common.Response, error) {
	err := libbox.NewStandaloneCommandClient().SetSystemProxyEnabled(in.IsEnabled)
	if err != nil {
		return &common.Response{
			Code:    common.ResponseCode_FAILED,
			Message: err.Error(),
		}, err
	}

	return &common.Response{
		Code:    common.ResponseCode_OK,
		Message: "",
	}, nil
}
