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
	CloseGrpcServer(closeReq.Mode) // it looks at grpcServer under its lock
	return &hcommon.Empty{}, nil
}

func Pause() {
	if box := static.Instance(); box != nil {
		if manager := box.PauseManager(); manager != nil {
			manager.DevicePause()
			if C.IsIos {
				if static.endPauseTimer == nil {
					// Wake, not manager.DeviceWake: the timer is made once and reset for every
					// pause, and this manager is the first box's; after a restart it woke a
					// closed box and left the running one paused
					static.endPauseTimer = time.AfterFunc(time.Minute, Wake)
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
			// if !C.IsIos {
			manager.DeviceWake()
			// }
		}
	}

}
