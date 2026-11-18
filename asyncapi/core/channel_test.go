package core_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannel_SyncChanges_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  *string
		expected string
	}{
		{
			name:     "unknown address keeps authored null",
			address:  nil,
			expected: "channels:\n  c:\n    address: null\n    title: C\n",
		},
		{
			name:     "set address replaces authored null",
			address:  pointer.From("orders"),
			expected: "channels:\n  c:\n    address: orders\n    title: C\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := "asyncapi: 3.1.0\ninfo:\n  title: T\n  version: '1'\nchannels:\n  c:\n    address: null\n    title: C\n"
			a, _, err := asyncapi.Unmarshal(t.Context(), strings.NewReader(doc))
			require.NoError(t, err, "unmarshal should succeed")

			a.GetChannels().GetOrZero("c").GetObject().Address = tt.address

			var buf bytes.Buffer
			require.NoError(t, asyncapi.Marshal(t.Context(), a, &buf), "marshal should succeed")
			assert.Contains(t, buf.String(), tt.expected, "address should be synced")
		})
	}
}

func TestChannel_SyncChanges_Error(t *testing.T) {
	t.Parallel()

	var channel core.Channel
	_, err := channel.SyncChanges(t.Context(), "channel", nil)
	require.Error(t, err, "syncing a non-struct model should fail")
	assert.Equal(t, "SyncValue expected target to be string, got struct", err.Error(), "sync error should match")
}
