package hcore

import (
	"context"
	"time"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	adapter "github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/experimental/libbox"
	outbound "github.com/sagernet/sing-box/outbound"
	common "github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/batch"
	E "github.com/sagernet/sing/common/exceptions"
	"google.golang.org/grpc"
)

var (
	systemInfoObserver        = NewObserver[*SystemInfo](1)
	outboundsInfoObserver     = NewObserver[*OutboundGroupList](1)
	mainOutboundsInfoObserver = NewObserver[*OutboundGroupList](1)
)

var (
	statusClient        *libbox.CommandClient
	groupClient         *libbox.CommandClient
	groupInfoOnlyClient *libbox.CommandClient
)

func (s *CoreService) GetSystemInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[SystemInfo]) error {
	if statusClient == nil {
		statusClient = libbox.NewCommandClient(
			&CommandClientHandler{
				command: libbox.CommandStatus,
				// port:   s.port,
			},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandStatus,
				StatusInterval: 1000000000, // 1000ms debounce
			},
		)

		defer func() {
			statusClient.Disconnect()
			statusClient = nil
		}()
		statusClient.Connect()
	}

	sub, done, _ := systemInfoObserver.Subscribe()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-done:
			return nil
		case info := <-sub:
			stream.Send(info)
			// case <-time.After(1000 * time.Millisecond):
		}
	}
}

// func (s *CoreService) OutboundsInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
// 	if groupClient == nil {
// 		groupClient = libbox.NewCommandClient(
// 			&CommandClientHandler{
// 				command: libbox.CommandGroup,
// 				// port:   s.port,
// 			},
// 			&libbox.CommandClientOptions{
// 				Command:        libbox.CommandGroup,
// 				StatusInterval: 500000000, // 500ms debounce
// 			},
// 		)

// 		defer func() {
// 			groupClient.Disconnect()
// 			groupClient = nil
// 		}()

// 		groupClient.Connect()
// 	}

// 	sub, done, _ := outboundsInfoObserver.Subscribe()

// 	for {
// 		select {
// 		case <-stream.Context().Done():
// 			return nil
// 		case <-done:
// 			return nil
// 		case info := <-sub:
// 			stream.Send(info)
// 			// case <-time.After(500 * time.Millisecond):
// 		}
// 	}
// }

// func (s *CoreService) MainOutboundsInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
// 	if groupInfoOnlyClient == nil {
// 		groupInfoOnlyClient = libbox.NewCommandClient(
// 			&CommandClientHandler{
// 				command: libbox.CommandGroupInfoOnly,
// 				// port:   s.port,
// 			},
// 			&libbox.CommandClientOptions{
// 				Command:        libbox.CommandGroupInfoOnly,
// 				StatusInterval: 500000000, // 500ms debounce
// 			},
// 		)

// 		defer func() {
// 			groupInfoOnlyClient.Disconnect()
// 			groupInfoOnlyClient = nil
// 		}()
// 		groupInfoOnlyClient.Connect()
// 	}

// 	sub, stopch, _ := mainOutboundsInfoObserver.Subscribe()

// 	for {
// 		select {
// 		case <-stream.Context().Done():
// 			return nil
// 		case <-stopch:
// 			return nil
// 		case info := <-sub:
// 			stream.Send(info)
// 			// case <-time.After(500 * time.Millisecond):
// 		}
// 	}
// }

func (s *CoreService) SelectOutbound(ctx context.Context, in *SelectOutboundRequest) (*hcommon.Response, error) {
	return SelectOutbound(in)
}

func SelectOutbound(in *SelectOutboundRequest) (*hcommon.Response, error) {
	// err := libbox.NewStandaloneCommandClient().SelectOutbound(in.GroupTag, in.OutboundTag)
	// if err != nil {
	// 	return &hcommon.Response{
	// 		Code:    hcommon.ResponseCode_FAILED,
	// 		Message: err.Error(),
	// 	}, err
	// }

	// return &hcommon.Response{
	// 	Code:    hcommon.ResponseCode_OK,
	// 	Message: "",
	// }, nil
	Log(LogLevel_DEBUG, LogType_CORE, "select outbound: ", in.GroupTag, " -> ", in.OutboundTag)
	outboundGroup, isLoaded := Box.GetInstance().Router().Outbound(in.GroupTag)
	if !isLoaded {
		return &hcommon.Response{
			Code:    hcommon.ResponseCode_FAILED,
			Message: E.New("selector not found: ", in.GroupTag).Error(),
		}, E.New("selector not found: ", in.GroupTag)
	}
	selector, isSelector := outboundGroup.(*outbound.Selector)
	if !isSelector {
		return &hcommon.Response{
			Code:    hcommon.ResponseCode_FAILED,
			Message: E.New("outbound is not a selector: ", in.GroupTag).Error(),
		}, E.New("outbound is not a selector: ", in.GroupTag)
	}
	if !selector.SelectOutbound(in.OutboundTag) {
		return &hcommon.Response{
			Code:    hcommon.ResponseCode_FAILED,
			Message: E.New("outbound not found in selector:: ", in.GroupTag).Error(),
		}, E.New("outbound not found in selector: ", in.GroupTag)
	}

	Box.UrlTestHistory().Observer().Emit(2)
	return &hcommon.Response{
		Code:    hcommon.ResponseCode_OK,
		Message: "",
	}, nil
}

func (s *CoreService) UrlTest(ctx context.Context, in *UrlTestRequest) (*hcommon.Response, error) {
	return UrlTest(in)
}

func UrlTest(in *UrlTestRequest) (*hcommon.Response, error) {
	// err := libbox.NewStandaloneCommandClient().URLTest(in.GroupTag)
	// if err != nil {
	// 	return &hcommon.Response{
	// 		Code:    hcommon.ResponseCode_FAILED,
	// 		Message: err.Error(),
	// 	}, err
	// }

	// return &hcommon.Response{
	// 	Code:    hcommon.ResponseCode_OK,
	// 	Message: "",
	// }, nil

	groupTag := in.GroupTag

	if Box == nil {
		return nil, E.New("service not ready")
	}
	abstractOutboundGroup, isLoaded := Box.GetInstance().Router().Outbound(groupTag)
	if !isLoaded {
		return &hcommon.Response{
			Code:    hcommon.ResponseCode_FAILED,
			Message: E.New("outbound group not found: ", in.GroupTag).Error(),
		}, E.New("outbound group not found: ", groupTag)
	}
	outboundGroup, isOutboundGroup := abstractOutboundGroup.(adapter.OutboundGroup)
	if !isOutboundGroup {
		return &hcommon.Response{
			Code:    hcommon.ResponseCode_FAILED,
			Message: E.New("outbound is not a group: ", in.GroupTag).Error(),
		}, E.New("outbound is not a group: ", groupTag)
	}

	urlTest, isURLTest := abstractOutboundGroup.(*outbound.URLTest)
	if isURLTest {
		go urlTest.CheckOutbounds()
	} else {
		historyStorage := Box.UrlTestHistory()
		outbounds := common.Filter(common.Map(outboundGroup.All(), func(it string) adapter.Outbound {
			itOutbound, _ := Box.GetInstance().Router().Outbound(it)
			return itOutbound
		}), func(it adapter.Outbound) bool {
			if it == nil {
				return false
			}
			_, isGroup := it.(adapter.OutboundGroup)
			return !isGroup
		})
		b, _ := batch.New(Box.Context(), batch.WithConcurrencyNum[any](10))
		for _, detour := range outbounds {
			outboundToTest := detour
			outboundTag := outboundToTest.Tag()
			b.Go(outboundTag, func() (any, error) {
				t, err := urltest.URLTest(Box.Context(), "", outboundToTest)
				if err != nil {
					// historyStorage.DeleteURLTestHistory(outboundTag)
					t = 65535
				} else {
				}
				historyStorage.StoreURLTestHistory(outboundTag, &urltest.History{
					Time:  time.Now(),
					Delay: t,
				})

				return nil, nil
			})
		}
	}

	return &hcommon.Response{
		Code:    hcommon.ResponseCode_OK,
		Message: "",
	}, nil
}
