package hcore

import (
	"context"

	"github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/adapter"
	E "github.com/sagernet/sing/common/exceptions"
)

// StartDeepSearch starts a deep search on a proxy whose search gave up (see
// adapter.DeepSearcher): the app asks the user first, as it may take hours, battery and CPU.
func (s *CoreService) StartDeepSearch(ctx context.Context, in *DeepSearchRequest) (*hcommon.Response, error) {
	return static.StartDeepSearch(in)
}

func (h *HiddifyInstance) StartDeepSearch(in *DeepSearchRequest) (*hcommon.Response, error) {
	failed := func(err error) (*hcommon.Response, error) {
		return &hcommon.Response{Code: hcommon.ResponseCode_FAILED, Message: err.Error()}, err
	}
	box, _ := h.running()
	if box == nil {
		return failed(E.New("service not started"))
	}
	var searcher adapter.DeepSearcher
	if outbound, loaded := box.Outbound().Outbound(in.Tag); loaded {
		searcher, _ = outbound.(adapter.DeepSearcher)
	}
	if searcher == nil {
		if endpoint, loaded := box.Endpoint().Get(in.Tag); loaded {
			searcher, _ = endpoint.(adapter.DeepSearcher)
		}
	}
	if searcher == nil {
		return failed(E.New("no search to deepen: ", in.Tag))
	}
	Log(LogLevel_INFO, LogType_CORE, "deep search: ", in.Tag)
	if err := searcher.StartDeepSearch(); err != nil {
		return failed(err)
	}
	return &hcommon.Response{Code: hcommon.ResponseCode_OK}, nil
}
