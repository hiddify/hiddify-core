package hcore

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func traceSettings(until time.Time) *ChangeHiddifySettingsRequest {
	return &ChangeHiddifySettingsRequest{HiddifySettingsJson: fmt.Sprintf(
		`{"log-level":"trace","log-level-until":%q,"log-level-after":"warn"}`, until.UTC().Format(time.RFC3339Nano))}
}

// A trace session ends in the core at its end, without the app, and settings read back after it
// (a start from the database) are not on trace at all.
func TestLogLevelEndsInTheCore(t *testing.T) {
	t.Cleanup(func() { ChangeHiddifySettings(&ChangeHiddifySettingsRequest{}, false) })

	_, err := ChangeHiddifySettings(traceSettings(time.Now().Add(150*time.Millisecond)), false)
	require.NoError(t, err)
	require.Equal(t, LogLevel_TRACE, LogLevel(static.logLevel.Load()))
	require.True(t, static.debug)

	require.Eventually(t, func() bool { return LogLevel(static.logLevel.Load()) == LogLevel_WARNING }, 2*time.Second, 10*time.Millisecond)
	logLevelEnd.Lock()
	require.Equal(t, "warn", static.HiddifyOptions.LogLevel)
	require.Empty(t, static.HiddifyOptions.LogLevelUntil)
	logLevelEnd.Unlock()
	require.Equal(t, static.setupDebug, static.debug)

	_, err = ChangeHiddifySettings(traceSettings(time.Now().Add(-time.Minute)), false)
	require.NoError(t, err)
	require.Equal(t, LogLevel_WARNING, LogLevel(static.logLevel.Load()))
	require.Equal(t, "warn", static.HiddifyOptions.LogLevel)
}

// Settings set during a session replace its end: the old timer does not touch them.
func TestLogLevelEndOfReplacedSettings(t *testing.T) {
	t.Cleanup(func() { ChangeHiddifySettings(&ChangeHiddifySettingsRequest{}, false) })

	_, err := ChangeHiddifySettings(traceSettings(time.Now().Add(100*time.Millisecond)), false)
	require.NoError(t, err)
	_, err = ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: `{"log-level":"trace"}`}, false)
	require.NoError(t, err)
	time.Sleep(300 * time.Millisecond)
	require.Equal(t, LogLevel_TRACE, LogLevel(static.logLevel.Load()))
	require.Equal(t, "trace", static.HiddifyOptions.LogLevel)
}
