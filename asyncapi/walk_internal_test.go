package asyncapi

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMatchFunc_UnregisteredType_Success(t *testing.T) {
	t.Parallel()

	target := "not a model"
	var visited any
	mf := getMatchFunc(&target)

	require.NoError(t, mf(Matcher{Any: func(v any) error {
		visited = v
		return nil
	}}), "visiting should succeed")
	assert.Equal(t, &target, visited, "Any should be called with an unregistered type")
	assert.NoError(t, mf(Matcher{}), "a matcher without Any should do nothing")
}

func TestGetMatchFunc_UnregisteredType_Error(t *testing.T) {
	t.Parallel()

	target := "not a model"
	stop := errors.New("stop")

	err := getMatchFunc(&target)(Matcher{Any: func(any) error { return stop }})
	assert.ErrorIs(t, err, stop, "the error from Any should be returned")
}
