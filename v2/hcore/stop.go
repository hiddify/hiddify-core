package hcore

import (
	"context"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing/common"
)

func (s *CoreService) Stop(ctx context.Context, empty *hcommon.Empty) (*CoreInfoResponse, error) {
	return Stop()
}

func Stop() (coreResponse *CoreInfoResponse, err error) {
	defer config.DeferPanicToError("stop", func(recovered_err error) {
		coreResponse, err = errorWrapper(MessageType_UNEXPECTED_ERROR, recovered_err)
	})

	// if static.CoreState != CoreStates_STARTED {
	// 	return errorWrapper(MessageType_INSTANCE_NOT_STARTED, fmt.Errorf("instance not started"))
	// }
	// if static.Box == nil {
	// 	return errorWrapper(MessageType_INSTANCE_NOT_FOUND, fmt.Errorf("instance not found"))
	// }
	static.lock.Lock()
	defer static.lock.Unlock()

	SetCoreStatus(CoreStates_STOPPING, MessageType_EMPTY, "")
	if static.Box == nil {
		return SetCoreStatus(CoreStates_STOPPED, MessageType_ALREADY_STOPPED, ""), nil
	}
	err = common.Close(static.Box)
	static.Box = nil
	if err != nil {
		return errorWrapper(MessageType_UNEXPECTED_ERROR, err)
	}

	return SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, ""), nil
}
