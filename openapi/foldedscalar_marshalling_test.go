package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The trailing table row is indented one space further than the rows above it, which
// makes yaml.v3 emit an extra line break before it on every encode. Marshalling has to
// stay a fixed point regardless of whether an overlay was involved.
const foldedScalarDocument = `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
  description: >-
    ### Widgets

    | Name | Kind |
    | ---- | ---- |
     | acme | ` + "`petstore`" + ` |
paths: {}
`

func TestMarshal_FoldedScalar_SurvivesRepeatedRoundTrips(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(foldedScalarDocument))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	want := doc.Info.GetDescription()
	require.Contains(t, want, "| acme |")

	current := foldedScalarDocument
	for i := range 3 {
		doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(current))
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, openapi.Marshal(ctx, doc, &buf))

		current = buf.String()
		assert.Equal(t, want, doc.Info.GetDescription(), "value changed after %d round trips", i+1)
	}
}
