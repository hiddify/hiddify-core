package hcore

import (
	"context"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
)

func (s *CoreService) Pause(ctx context.Context, pauseReq *PauseRequest) (*hcommon.Empty, error) {
	if pauseReq == nil {
		return nil, nil
	}
	mode := (*pauseReq).Mode
	if grpcServer[mode] == nil {
		Log(LogLevel_WARNING, LogType_CORE, "grpcServer already stoped")
		return nil, nil
	}

	CloseGrpcServer(mode)
	return &hcommon.Empty{}, nil
}
