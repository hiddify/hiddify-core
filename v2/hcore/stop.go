package hcore

import (
	"context"
	"fmt"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
)

func (s *CoreService) Stop(ctx context.Context, empty *hcommon.Empty) (*CoreInfoResponse, error) {
	return Stop()
}

func Stop() (coreResponse *CoreInfoResponse, err error) {
	defer config.DeferPanicToError("stop", func(recovered_err error) {
		coreResponse, err = errorWrapper(MessageType_UNEXPECTED_ERROR, recovered_err)
	})

	if static.CoreState != CoreStates_STARTED {
		return errorWrapper(MessageType_INSTANCE_NOT_STARTED, fmt.Errorf("instance not started"))
	}
	if static.Box == nil {
		return errorWrapper(MessageType_INSTANCE_NOT_FOUND, fmt.Errorf("instance not found"))
	}
	SetCoreStatus(CoreStates_STOPPING, MessageType_EMPTY, "")

	err = static.Box.Close()
	if err != nil {
		return errorWrapper(MessageType_UNEXPECTED_ERROR, err)
	}
	static.Box = nil
	return SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, ""), nil
}
