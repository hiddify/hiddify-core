package hcore

import (
	"os"
	"path/filepath"
	"testing"
)

// The app (its core is the library in the app on Windows and Linux) asks to disable the system
// proxy once the proxy-mode engine's process ended. A crashed engine left its marker in the shared
// working dir: the disable undoes it. A core running an engine itself keeps its own marker.
func TestStoppedEngineLeftoversAreCleanedOnDisable(t *testing.T) {
	previousPath, previousState := sWorkingPath, static.CoreState
	t.Cleanup(func() {
		sWorkingPath = previousPath
		setCoreState(previousState)
	})
	sWorkingPath = t.TempDir()
	marker := filepath.Join(sWorkingPath, "data", "system-proxy.on")
	writeEngineMarker := func() {
		if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
			t.Fatal(err)
		}
		// an address restoreSystemProxy refuses at once, so the test never touches this machine's
		// real system proxy
		if err := os.WriteFile(marker, []byte("not-an-address"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeEngineMarker()
	setCoreState(CoreStates_STARTED)
	cleanupStoppedEngine()
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("a running engine's marker was removed: %v", err)
	}

	setCoreState(CoreStates_STOPPED)
	cleanupStoppedEngine()
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("the stopped engine's marker is still there: %v", err)
	}
}
