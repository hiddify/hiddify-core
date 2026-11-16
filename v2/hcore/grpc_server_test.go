package hcore

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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

// A store that cannot be read is an error, not a missing certificate: a new one would replace the
// stored one, which a client of another process has pinned.
func TestUnreadableStoreDoesNotReplaceTheServerCertificate(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	certpair = nil
	t.Cleanup(func() { certpair = nil })

	// the store's folder is a file: it can not be opened
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "data", "AppSettings.db")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data", "AppSettings.db"), nil, 0o644))

	require.Error(t, loadServerCertificate())
	require.Nil(t, certpair)
}

// With no certificate stored, one is made and stored.
func TestMissingServerCertificateIsMade(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	certpair = nil
	t.Cleanup(func() { certpair = nil })

	require.NoError(t, loadServerCertificate())
	require.NotNil(t, certpair)
	made := certpair.Certificate

	certpair = nil
	require.NoError(t, loadServerCertificate())
	require.Equal(t, made, certpair.Certificate, "the stored one is read back")
}
