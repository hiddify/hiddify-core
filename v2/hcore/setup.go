package hcore

import (
	"context"

	"github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/service_manager"
)

var (
	sWorkingPath          string
	sTempPath             string
	sUserID               int
	sGroupID              int
	statusPropagationPort int64
)

func InitHiddifyService() error {
	return service_manager.StartServices()
}

func (s *CoreService) Setup(ctx context.Context, req *SetupRequest) (*hcommon.Response, error) {
	if grpcServer[req.Mode] != nil {
		return &hcommon.Response{Code: hcommon.ResponseCode_OK, Message: ""}, nil
	}
	err := Setup(req, nil)
	code := hcommon.ResponseCode_OK
	if err != nil {
		code = hcommon.ResponseCode_FAILED
	}
	return &hcommon.Response{Code: code, Message: err.Error()}, err
}
