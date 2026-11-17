package arazzo_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceDescription_Marshal_EmptyType_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		doc      string
		expected string
	}{
		{
			name: "cleared type is removed",
			doc: `arazzo: 1.1.0
info:
  title: Sources
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows: []
`,
			expected: `arazzo: 1.1.0
info:
  title: Sources
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
workflows: []
`,
		},
		{
			name: "missing type stays missing",
			doc: `arazzo: 1.1.0
info:
  title: Sources
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
workflows: []
`,
			expected: `arazzo: 1.1.0
info:
  title: Sources
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
workflows: []
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			a, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(tt.doc), arazzo.WithSkipValidation())
			require.NoError(t, err, "unmarshal should succeed")
			a.SourceDescriptions[0].Type = ""

			var buf bytes.Buffer
			require.NoError(t, arazzo.Marshal(ctx, a, &buf), "marshal should succeed")
			assert.Equal(t, tt.expected, buf.String(), "an empty optional type should not be written")
		})
	}
}

func TestSourceDescription_Marshal_NewSourceWithoutType_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	a, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(`arazzo: 1.1.0
info:
  title: Sources
  version: 1.0.0
sourceDescriptions: []
workflows: []
`), arazzo.WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")
	a.SourceDescriptions = append(a.SourceDescriptions, &arazzo.SourceDescription{Name: "api", URL: "https://example.com/openapi.yaml"})

	var buf bytes.Buffer
	require.NoError(t, arazzo.Marshal(ctx, a, &buf), "marshal should succeed")
	assert.NotContains(t, buf.String(), "type:", "an unset optional type should not be written")
}
