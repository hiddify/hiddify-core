package v2

import (
	"time"

	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"google.golang.org/grpc"
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
				StatusInterval: 1_000_000_000, // 1000ms debounce
			},
		)
		defer func() {
			statusClient.Disconnect()
			statusClient = nil
		}()
		statusClient.Connect()
	}

	sub, done, _ := systemInfoObserver.Subscribe()
	defer systemInfoObserver.UnSubscribe(sub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-done:
			return nil
		case info := <-sub:
			if err := stream.Send(info); err != nil {
				return err
			}
		case <-time.After(1 * time.Second):
		}
	}
}

func (s *CoreService) OutboundsInfo(req *pb.Empty, stream grpc.ServerStreamingServer[pb.OutboundGroupList]) error {
	if groupClient == nil {
		groupClient = libbox.NewCommandClient(
			&CommandClientHandler{},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroup,
				StatusInterval: 500_000_000, // 500ms debounce
			},
		)
		defer func() {
			groupClient.Disconnect()
			groupClient = nil
		}()
		groupClient.Connect()
	}

	sub, done, _ := outboundsInfoObserver.Subscribe()
	defer outboundsInfoObserver.UnSubscribe(sub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-done:
			return nil
		case info := <-sub:
			if err := stream.Send(info); err != nil {
				return err
			}
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *CoreService) MainOutboundsInfo(req *pb.Empty, stream grpc.ServerStreamingServer[pb.OutboundGroupList]) error {
	if groupInfoOnlyClient == nil {
		groupInfoOnlyClient = libbox.NewCommandClient(
			&CommandClientHandler{},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroup,
				StatusInterval: 500_000_000, // 500ms debounce
			},
		)
		defer func() {
			groupInfoOnlyClient.Disconnect()
			groupInfoOnlyClient = nil
		}()
		groupInfoOnlyClient.Connect()
	}

	sub, stopch, _ := mainOutboundsInfoObserver.Subscribe()
	defer mainOutboundsInfoObserver.UnSubscribe(sub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-stopch:
			return nil
		case info := <-sub:
			if err := stream.Send(info); err != nil {
				return err
			}
		case <-time.After(500 * time.Millisecond):
		}
	}
}
