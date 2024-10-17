package hcore

import (
	"context"

	common "github.com/hiddify/hiddify-core/v2/common"
	"github.com/sagernet/sing-box/experimental/libbox"
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

func (s *CoreService) GetSystemInfo(req *common.Empty, stream grpc.ServerStreamingServer[SystemInfo]) error {
	if statusClient == nil {
		statusClient = libbox.NewCommandClient(
			&CommandClientHandler{
				logger: coreLogFactory.NewLogger("[SystemInfo Command Client]"),
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

func (s *CoreService) OutboundsInfo(req *common.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
	if groupClient == nil {
		groupClient = libbox.NewCommandClient(
			&CommandClientHandler{
				logger: coreLogFactory.NewLogger("[OutboundsInfo Command Client]"),
				// port:   s.port,
			},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroup,
				StatusInterval: 500000000, // 500ms debounce
			},
		)

		defer func() {
			groupClient.Disconnect()
			groupClient = nil
		}()

		groupClient.Connect()
	}

	sub, done, _ := outboundsInfoObserver.Subscribe()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-done:
			return nil
		case info := <-sub:
			stream.Send(info)
			// case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *CoreService) MainOutboundsInfo(req *common.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
	if groupInfoOnlyClient == nil {
		groupInfoOnlyClient = libbox.NewCommandClient(
			&CommandClientHandler{
				logger: coreLogFactory.NewLogger("[MainOutboundsInfo Command Client]"),
				// port:   s.port,
			},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroupInfoOnly,
				StatusInterval: 500000000, // 500ms debounce
			},
		)

		defer func() {
			groupInfoOnlyClient.Disconnect()
			groupInfoOnlyClient = nil
		}()
		groupInfoOnlyClient.Connect()
	}

	sub, stopch, _ := mainOutboundsInfoObserver.Subscribe()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-stopch:
			return nil
		case info := <-sub:
			stream.Send(info)
			// case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *CoreService) SelectOutbound(ctx context.Context, in *SelectOutboundRequest) (*common.Response, error) {
	return SelectOutbound(in)
}

func SelectOutbound(in *SelectOutboundRequest) (*common.Response, error) {
	err := libbox.NewStandaloneCommandClient().SelectOutbound(in.GroupTag, in.OutboundTag)
	if err != nil {
		return &common.Response{
			Code:    common.ResponseCode_FAILED,
			Message: err.Error(),
		}, err
	}

	return &common.Response{
		Code:    common.ResponseCode_OK,
		Message: "",
	}, nil
}

func (s *CoreService) UrlTest(ctx context.Context, in *UrlTestRequest) (*common.Response, error) {
	return UrlTest(in)
}

func UrlTest(in *UrlTestRequest) (*common.Response, error) {
	err := libbox.NewStandaloneCommandClient().URLTest(in.GroupTag)
	if err != nil {
		return &common.Response{
			Code:    common.ResponseCode_FAILED,
			Message: err.Error(),
		}, err
	}

	return &common.Response{
		Code:    common.ResponseCode_OK,
		Message: "",
	}, nil
}
