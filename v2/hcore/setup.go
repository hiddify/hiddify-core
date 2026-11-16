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
	sSetupMode            SetupMode // which core this process is: the app's own, or the background one
)

func InitHiddifyService() error {
	return service_manager.StartServices()
}

// Setup sets the core up for a gRPC client. Setup itself answers an already started server (under
// its lock), so the handler does not look at grpcServer, which it would read without that lock.
func (s *CoreService) Setup(ctx context.Context, req *SetupRequest) (*hcommon.Response, error) {
	if err := Setup(req, nil); err != nil {
		return &hcommon.Response{Code: hcommon.ResponseCode_FAILED, Message: err.Error()}, err
	}
	return &hcommon.Response{Code: hcommon.ResponseCode_OK}, nil
}
