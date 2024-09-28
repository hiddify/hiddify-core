package v2

import (
	"context"
	"time"

	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"google.golang.org/grpc"
)

var (
	systemInfoObserver        = NewObserver[pb.SystemInfo](10)
	outboundsInfoObserver     = NewObserver[pb.OutboundGroupList](10)
	mainOutboundsInfoObserver = NewObserver[pb.OutboundGroupList](10)
)

var (
	statusClient        *libbox.CommandClient
	groupClient         *libbox.CommandClient
	groupInfoOnlyClient *libbox.CommandClient
)

func (s *CoreService) GetSystemInfo(req *pb.Empty, stream grpc.ServerStreamingServer[pb.SystemInfo]) error {
	if statusClient == nil {
		statusClient = libbox.NewCommandClient(
			&CommandClientHandler{},
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
			stream.Send(&info)
		case <-time.After(1000 * time.Millisecond):
		}
	}
}

func (s *CoreService) OutboundsInfo(req *pb.Empty, stream grpc.ServerStreamingServer[pb.OutboundGroupList]) error {
	if groupClient == nil {
		groupClient = libbox.NewCommandClient(
			&CommandClientHandler{},
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
			stream.Send(&info)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *CoreService) MainOutboundsInfo(req *pb.Empty, stream grpc.ServerStreamingServer[pb.OutboundGroupList]) error {
	if groupInfoOnlyClient == nil {
		groupInfoOnlyClient = libbox.NewCommandClient(
			&CommandClientHandler{},
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
			stream.Send(&info)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *CoreService) SelectOutbound(ctx context.Context, in *pb.SelectOutboundRequest) (*pb.Response, error) {
	return SelectOutbound(in)
}

func SelectOutbound(in *pb.SelectOutboundRequest) (*pb.Response, error) {
	err := libbox.NewStandaloneCommandClient().SelectOutbound(in.GroupTag, in.OutboundTag)
	if err != nil {
		return &pb.Response{
			ResponseCode: pb.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}

	return &pb.Response{
		ResponseCode: pb.ResponseCode_OK,
		Message:      "",
	}, nil
}

func (s *CoreService) UrlTest(ctx context.Context, in *pb.UrlTestRequest) (*pb.Response, error) {
	return UrlTest(in)
}

func UrlTest(in *pb.UrlTestRequest) (*pb.Response, error) {
	err := libbox.NewStandaloneCommandClient().URLTest(in.GroupTag)
	if err != nil {
		return &pb.Response{
			ResponseCode: pb.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}

	return &pb.Response{
		ResponseCode: pb.ResponseCode_OK,
		Message:      "",
	}, nil
}
