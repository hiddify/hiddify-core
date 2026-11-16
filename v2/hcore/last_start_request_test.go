package hcore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A start without a request (Android's tile, shortcut and always-on) runs the last request again,
// with the memory limit setting it had: it was dropped, so those starts always had the limit on.
func TestLastStartRequestKeepsTheMemoryLimitSetting(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))

	require.NoError(t, saveLastStartRequest(&StartRequest{ConfigPath: "configs/a.json", ConfigName: "a", DisableMemoryLimit: true}))
	request, err := loadLastStartRequestIfNeeded(&StartRequest{})
	require.NoError(t, err)
	require.Equal(t, "configs/a.json", request.ConfigPath)
	require.True(t, request.DisableMemoryLimit)
}
