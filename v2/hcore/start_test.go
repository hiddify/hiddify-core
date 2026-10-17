package hcore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildConfig_StartServicePanic(t *testing.T) {
	ctx := context.Background()

	in := &StartRequest{
		ConfigContent: "{}",
	}

	_, err := StartService(ctx, in)
	require.Error(t, err)
	require.Equal(t, static.CoreState, CoreStates_STOPPED)
}

func TestStartService_CancelledRequestContext(t *testing.T) {
	request, cancel := context.WithCancel(context.Background())
	cancel()
	ctx := withRequestContext(context.Background(), request)

	response, err := StartService(ctx, &StartRequest{ConfigContent: "{}"})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, CoreStates_STOPPED, response.CoreState)
	require.Equal(t, CoreStates_STOPPED, static.CoreState)
	require.Nil(t, static.StartedService)

	response, err = Restart(ctx, &StartRequest{ConfigContent: "{}"})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, CoreStates_STOPPED, static.CoreState)

	_, err = HotReload(ctx, &StartRequest{ConfigContent: "{}"})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, CoreStates_STOPPED, static.CoreState)
}
