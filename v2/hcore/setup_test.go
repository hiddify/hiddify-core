package hcore

import (
	"context"
	"testing"

	"github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/stretchr/testify/require"
)

// A Setup that succeeds answers OK: the handler read the message of a nil error and panicked.
func TestSetupHandlerAnswersASuccess(t *testing.T) {
	dir := t.TempDir()
	response, err := (&CoreService{}).Setup(context.Background(), &SetupRequest{
		BasePath:   dir,
		WorkingDir: dir,
		TempDir:    dir,
		Mode:       SetupMode_OLD,
	})
	require.NoError(t, err)
	require.Equal(t, hcommon.ResponseCode_OK, response.Code)
}
