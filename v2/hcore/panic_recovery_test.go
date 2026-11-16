package hcore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// A panic while reading a profile is that profile's error: before, Parse and GenerateConfig
// returned no error (the app took the profile for valid) and stopped the running core.
func TestPanicInParseIsAnErrorAndLeavesTheCoreRunning(t *testing.T) {
	SetCoreStatus(CoreStates_STARTED, MessageType_EMPTY, "")
	t.Cleanup(func() { SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, "") })

	response, err := Parse(context.Background(), nil) // nil request: a panic inside Parse
	require.Error(t, err)
	require.NotNil(t, response)
	require.NotZero(t, response.ResponseCode)

	_, err = GenerateConfig(context.Background(), nil)
	require.Error(t, err)

	require.Equal(t, CoreStates_STARTED, currentCoreState())
}

// A panic in Setup is returned: before, Setup returned nil and the caller took the core for set up.
func TestPanicInSetupIsAnError(t *testing.T) {
	require.Error(t, Setup(nil, nil))
}
