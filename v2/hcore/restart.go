package hcore

import (
	"context"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
)

func (s *CoreService) Restart(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return Restart(withRequestContext(static.BaseContext, ctx), in)
}

func Restart(ctx context.Context, in *StartRequest) (coreResponse *CoreInfoResponse, err error) {
	defer config.DeferPanicToError("startmobile", func(recovered_err error) {
		coreResponse, err = errorWrapper(MessageType_UNEXPECTED_ERROR, recovered_err)
	})
	log.Debug("[Service] Restarting")
	// if static.CoreState != CoreStates_STARTED {
	// 	return errorWrapper(MessageType_INSTANCE_NOT_STARTED, fmt.Errorf("instance not started"))
	// }
	// if static.Box == nil {
	// 	return errorWrapper(MessageType_INSTANCE_NOT_FOUND, fmt.Errorf("instance not found"))
	// }

	if err := contextErr(ctx); err != nil {
		return &CoreInfoResponse{
			CoreState:   static.CoreState,
			MessageType: MessageType_EMPTY,
			Message:     "restart cancelled",
		}, err
	}

	resp, err := Stop()
	if err != nil {
		return resp, err
	}

	if C.IsAndroid && static.HiddifyOptions.EnableTun {
		sleepContext(ctx, time.Second)
	}
	if err := contextErr(ctx); err != nil {
		// stopped already: stay stopped instead of starting again
		return SetCoreStatus(CoreStates_STOPPED, MessageType_INSTANCE_NOT_STARTED, "restart cancelled"), err
	}
	return StartService(ctx, in)
}
