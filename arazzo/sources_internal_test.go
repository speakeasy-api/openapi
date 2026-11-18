package arazzo

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/pointer"
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

func TestFindAsyncTarget_UnknownOperationSource_Success(t *testing.T) {
	t.Parallel()

	// validateStepSources only looks for an operationId's AsyncAPI target once the source is loaded, so call it
	// directly for a step whose source description can't be determined.
	s := &Step{OperationID: pointer.From(expression.Expression("onEvent"))}
	target, errs := (&Arazzo{}).findAsyncTarget(&SourceDocuments{}, s)
	assert.Nil(t, target, "no target should be found")
	assert.Empty(t, errs, "nothing should be reported")
}
