package hcore

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/db"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	adapter "github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/conntrack"
	"github.com/sagernet/sing-box/protocol/group"

	common "github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/batch"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/memory"
	"google.golang.org/grpc"
)

func (h *HiddifyInstance) readStatus(prev *SystemInfo) *SystemInfo {
	var message SystemInfo
	message.Memory = int64(memory.Inuse())
	message.Goroutines = int32(runtime.NumGoroutine())
	message.ConnectionsOut = int32(conntrack.Count())

	if ss := h.StartedService; ss != nil {
		status := ss.ReadStatus()
		message.DownlinkTotal = status.DownlinkTotal
		message.UplinkTotal = status.UplinkTotal
		message.ConnectionsIn = status.ConnectionsIn
		message.ConnectionsOut = status.ConnectionsOut

		if prev != nil {
			message.Uplink = message.UplinkTotal - prev.UplinkTotal
			message.Downlink = message.DownlinkTotal - prev.DownlinkTotal
		}
		if box := h.Box(); box != nil {
			if currentOutBound, ok := box.Outbound().Outbound(config.OutboundSelectTag); ok {
				if selectOutBound, ok := currentOutBound.(*group.Selector); ok {
					message.CurrentOutbound = TrimTagName(selectOutBound.Now())
				}
			}
			if message.CurrentOutbound == config.OutboundURLTestTag {
				if currentOutBound, ok := box.Outbound().Outbound(config.OutboundURLTestTag); ok {
					if urltest, ok := currentOutBound.(*group.URLTest); ok {
						message.CurrentOutbound = fmt.Sprint(message.CurrentOutbound, "→", TrimTagName(urltest.Now()))
					}
				}
			}
		}

		if prev == nil || prev.CurrentProfile == "" || message.UplinkTotal < 1000000 {
			settings := db.GetTable[hcommon.AppSettings]()
			lastName, err := settings.Get("lastStartRequestName")
			if err == nil {
				message.CurrentProfile = lastName.Value.(string)
			}
		} else {
			message.CurrentProfile = prev.CurrentProfile
		}
	}

	return &message
}

func (s *CoreService) GetSystemInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[SystemInfo]) error {
	// return fmt.Errorf("not implemented yet")
	ticker := time.NewTicker(time.Duration(1 * time.Second))
	h := static
	if ctx := h.Context(); ctx != nil {
		current_status := h.readStatus(nil)
		for {
			select {
			case <-stream.Context().Done():
				return nil
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				current_status = h.readStatus(current_status)
				stream.Send(current_status)
			}
		}
	}
	return nil
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
	return static.SelectOutbound(in)
}

func (h *HiddifyInstance) SelectOutbound(in *SelectOutboundRequest) (*hcommon.Response, error) {
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
	if box := h.Box(); box != nil {
		outboundGroup, isLoaded := box.Outbound().Outbound(in.GroupTag)
		if !isLoaded {
			return &hcommon.Response{
				Code:    hcommon.ResponseCode_FAILED,
				Message: E.New("selector not found: ", in.GroupTag).Error(),
			}, E.New("selector not found: ", in.GroupTag)
		}
		selector, isSelector := outboundGroup.(*group.Selector)
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
		Log(LogLevel_DEBUG, LogType_CORE, "Trying to ping outbound: ", in.OutboundTag)
		go func() {
			for _, detour := range box.Outbound().Outbounds() {
				if urlTest, ok := detour.(*group.URLTest); ok {
					if urlTest.ForceRecheckOutbound(in.OutboundTag) == nil {
						break
					}
				}
			}
		}()
		if urltesHistory := h.UrlTestHistory(); urltesHistory != nil {
			urltesHistory.Observer().Emit(2)
		}
	}
	return &hcommon.Response{
		Code:    hcommon.ResponseCode_OK,
		Message: "",
	}, nil
}

func (s *CoreService) UrlTest(ctx context.Context, in *UrlTestRequest) (*hcommon.Response, error) {
	return static.UrlTest(in)
}

func (h *HiddifyInstance) UrlTest(in *UrlTestRequest) (*hcommon.Response, error) {
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
	box := h.Box()
	if box == nil {
		return nil, E.New("service not ready")
	}

	router := box.Outbound()
	abstractOutboundGroup, isLoaded := router.Outbound(groupTag)
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

	if urlTest, isURLTest := abstractOutboundGroup.(*group.URLTest); isURLTest {
		go func() {
			for _, p := range router.Outbounds() {
				if p.Tag() == groupTag {
					continue
				}
				if group, isGroup := p.(adapter.OutboundGroup); isGroup {
					urlTest.ForceRecheckOutbound(group.Now())
				}
			}
			urlTest.CheckOutbounds()
		}()
	} else {
		historyStorage := h.UrlTestHistory()
		outbounds := common.Filter(common.Map(outboundGroup.All(), func(it string) adapter.Outbound {
			itOutbound, _ := router.Outbound(it)
			return itOutbound
		}), func(it adapter.Outbound) bool {
			if it == nil {
				return false
			}
			_, isGroup := it.(adapter.OutboundGroup)
			return !isGroup
		})
		b, _ := batch.New(h.Context(), batch.WithConcurrencyNum[any](10))
		for _, detour := range outbounds {
			outboundToTest := detour
			outboundTag := outboundToTest.Tag()
			b.Go(outboundTag, func() (any, error) {
				instance := box

				group.CheckOutbound(instance.Logger(), h.Context(), historyStorage, router, "", outboundToTest, nil)
				return nil, nil
			})
		}
	}

	return &hcommon.Response{
		Code:    hcommon.ResponseCode_OK,
		Message: "",
	}, nil
}
