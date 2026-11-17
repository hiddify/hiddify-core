package hcore

import (
	"os"
	"path/filepath"
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
	require.Equal(t, filepath.Join(dir, "configs/a.json"), request.ConfigPath) // relative to the working directory
	require.True(t, request.DisableMemoryLimit)
}

// The working directory moved (iOS, out of Library/Caches): the last request's config is found
// where it moved. It was saved as an absolute path, and a start without a request failed.
func TestLastStartRequestFollowsAMovedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Support", "Working")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "configs"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	config := filepath.Join(dir, "configs", "a.json")
	require.NoError(t, os.WriteFile(config, []byte("{}"), 0o644))

	// saved now: relative, so it follows the directory
	require.NoError(t, saveLastStartRequest(&StartRequest{ConfigPath: config, ConfigName: "a"}))
	request, err := loadLastStartRequestIfNeeded(&StartRequest{})
	require.NoError(t, err)
	require.Equal(t, config, request.ConfigPath)

	// saved before, in the old place
	old := filepath.Join(root, "Caches", "Working", "configs", "a.json")
	require.Equal(t, config, inWorkingDir(old))
}
