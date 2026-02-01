package hcore

import (
	"context"
	"time"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	C "github.com/sagernet/sing-box/constant"
)

func (s *CoreService) Close(ctx context.Context, closeReq *CloseRequest) (*hcommon.Empty, error) {
	if closeReq == nil {
		return nil, nil
	}
	mode := closeReq.Mode
	if grpcServer[mode] == nil {
		Log(LogLevel_WARNING, LogType_CORE, "grpcServer already stoped")
		return nil, nil
	}

	CloseGrpcServer(mode)
	return &hcommon.Empty{}, nil
}

func Pause() {
	if box := static.Instance(); box != nil {
		if manager := box.PauseManager(); manager != nil {
			manager.DevicePause()
			if C.IsIos {
				if static.endPauseTimer == nil {
					static.endPauseTimer = time.AfterFunc(time.Minute, manager.DeviceWake)
				} else {
					static.endPauseTimer.Reset(time.Minute)
				}
			}
		}
	}
}

func Wake() {
	if box := static.Instance(); box != nil {
		if manager := box.PauseManager(); manager != nil {
			if !C.IsIos {
				manager.DeviceWake()
			}
		}
	}

}
