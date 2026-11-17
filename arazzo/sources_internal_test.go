package arazzo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchSource_InvalidRequest_Error(t *testing.T) {
	t.Parallel()

	// A nil context is the only way to make building the request fail, as only parseable URLs are fetched.
	var ctx context.Context
	_, err := fetchSource(ctx, "https://example.com/openapi.yaml", LoadSourcesOptions{})
	require.Error(t, err, "building the request should fail")
	assert.Contains(t, err.Error(), "nil Context", "error should match")
}
