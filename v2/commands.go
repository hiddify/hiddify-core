package v2

import (
	"context"
	"time"

	pb "github.com/hiddify/libcore/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing/common/observable"
)

var systemInfoObserver = observable.Observer[pb.SystemInfo]{}
var outboundsInfoObserver = observable.Observer[pb.OutboundGroupList]{}
var mainOutboundsInfoObserver = observable.Observer[pb.OutboundGroupList]{}

var (
	statusClient        *libbox.CommandClient
	groupClient         *libbox.CommandClient
	groupInfoOnlyClient *libbox.CommandClient
)

func (s *server) GetSystemInfo(stream pb.Hiddify_GetSystemInfoServer) error {
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

func (s *server) OutboundsInfo(stream pb.Hiddify_OutboundsInfoServer) error {
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

func (s *server) MainOutboundsInfo(stream pb.Hiddify_MainOutboundsInfoServer) error {
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

// Implement the SelectOutbound method
func (s *server) SelectOutbound(ctx context.Context, in *pb.SelectOutboundRequest) (*pb.Response, error) {
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

// Implement the UrlTest method
func (s *server) UrlTest(ctx context.Context, in *pb.UrlTestRequest) (*pb.Response, error) {
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
