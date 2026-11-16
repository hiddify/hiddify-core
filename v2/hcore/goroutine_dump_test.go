package hcore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The dump of a slow stop is the goroutine profile, not runtime.Stack(all) (the profile's debug=2
// form), whose printing unwinder throws on a goroutine it cannot unwind (go1.27.3).
func TestSlowStopDumpIsTheGoroutineProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goroutines.log")
	require.NoError(t, dumpGoroutinesToFile(path))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(content), "goroutine profile: total "), "%.200s", content)
}

// Setup runs again and again in one process (Android, on every connect); the snapshot loop is
// started once, not once per Setup.
func TestSetupStartsOneGoroutineSnapshotLoop(t *testing.T) {
	dir := t.TempDir()
	for range 3 {
		require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	}
	require.LessOrEqual(t, goroutinesIn(goroutineDump(), "hcore.writeGoroutineSnapshots"), 1)
}

// goroutinesIn counts the goroutines of a goroutine profile (debug=1) whose stack has function:
// after the header line, each stack is a block that starts with "<count> @".
func goroutinesIn(profile, function string) int {
	_, stacks, _ := strings.Cut(profile, "\n")
	total := 0
	for _, block := range strings.Split(stacks, "\n\n") {
		var count int
		if _, err := fmt.Sscanf(block, "%d @", &count); err == nil && strings.Contains(block, function) {
			total += count
		}
	}
	return total
}
