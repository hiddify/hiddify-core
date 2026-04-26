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
