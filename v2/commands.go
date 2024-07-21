package v2

import (
	"context"
	"time"

	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
)

var systemInfoObserver = NewObserver[pb.SystemInfo](10)
var outboundsInfoObserver = NewObserver[pb.OutboundGroupList](10)
var mainOutboundsInfoObserver = NewObserver[pb.OutboundGroupList](10)

var (
	statusClient        *libbox.CommandClient
	groupClient         *libbox.CommandClient
	groupInfoOnlyClient *libbox.CommandClient
)

func (s *CoreService) GetSystemInfo(stream pb.Core_GetSystemInfoServer) error {
	if statusClient == nil {
		statusClient = libbox.NewCommandClient(
			&CommandClientHandler{},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandStatus,
				StatusInterval: 1000000000, //1000ms debounce
			},
		)

		defer func() {
			statusClient.Disconnect()
			statusClient = nil
		}()
		statusClient.Connect()
	}

	sub, _, _ := systemInfoObserver.Subscribe()
	stopch := make(chan int)
	go func() {
		stream.Recv()
		close(stopch)
	}()
	for {
		select {
		case <-stream.Context().Done():
			break
		case <-stopch:
			break
		case info := <-sub:
			stream.Send(&info)
		case <-time.After(1000 * time.Millisecond):
		}
	}
}

func (s *CoreService) OutboundsInfo(stream pb.Core_OutboundsInfoServer) error {
	if groupClient == nil {
		groupClient = libbox.NewCommandClient(
			&CommandClientHandler{},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroup,
				StatusInterval: 500000000, //500ms debounce
			},
		)

		defer func() {
			groupClient.Disconnect()
			groupClient = nil
		}()
		groupClient.Connect()
	}

	sub, _, _ := outboundsInfoObserver.Subscribe()
	stopch := make(chan int)
	go func() {
		stream.Recv()
		close(stopch)
	}()
	for {
		select {
		case <-stream.Context().Done():
			break
		case <-stopch:
			break
		case info := <-sub:
			stream.Send(&info)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *CoreService) MainOutboundsInfo(stream pb.Core_MainOutboundsInfoServer) error {
	if groupInfoOnlyClient == nil {
		groupInfoOnlyClient = libbox.NewCommandClient(
			&CommandClientHandler{},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroupInfoOnly,
				StatusInterval: 500000000, //500ms debounce
			},
		)

		defer func() {
			groupInfoOnlyClient.Disconnect()
			groupInfoOnlyClient = nil
		}()
		groupInfoOnlyClient.Connect()
	}

	sub, _, _ := mainOutboundsInfoObserver.Subscribe()
	stopch := make(chan int)
	go func() {
		stream.Recv()
		close(stopch)
	}()
	for {
		select {
		case <-stream.Context().Done():
			break
		case <-stopch:
			break
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
