package hcore

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// A server that could not listen is not kept: Setup took it for started and answered OK from then
// on, with nothing listening.
func TestServerThatCannotListenIsNotKept(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	grpcServer = make(map[SetupMode]*grpc.Server)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// a free port, on an address that is not this host's: the listen fails
	_, err = StartGrpcServerByMode(fmt.Sprintf("192.0.2.1:%d", port), SetupMode_GRPC_NORMAL_INSECURE)
	require.Error(t, err)
	require.Nil(t, grpcServer[SetupMode_GRPC_NORMAL_INSECURE])
}
