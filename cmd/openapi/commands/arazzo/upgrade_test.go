package arazzo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeArazzo_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expected       string
		expectedStderr []string
	}{
		{
			name: "upgrades 1.0 document",
			input: `arazzo: 1.0.1
info:
  title: Upgrade
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        successCriteria:
          - context: $response.body
            condition: $[?@.status == 'available']
            type: jsonpath
`,
			expected: `arazzo: 1.1.0
info:
  title: Upgrade
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        successCriteria:
          - context: $response.body
            condition: $[?@.status == 'available']
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
`,
			expectedStderr: []string{"Successfully upgraded from 1.0.1 to 1.1.0"},
		},
		{
			name: "reports expressions that need manual changes",
			input: `arazzo: 1.0.1
info:
  title: Upgrade
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        outputs:
          location: $response.headers.Location
`,
			expected: `arazzo: 1.1.0
info:
  title: Upgrade
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        outputs:
          location: $response.headers.Location
`,
			expectedStderr: []string{
				"Successfully upgraded from 1.0.1 to 1.1.0",
				"Found 1 validation errors in the upgraded document that need to be fixed manually",
				"$response.headers.Location",
			},
		},
		{
			name:           "no upgrade needed",
			input:          "arazzo: 1.1.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			expected:       "arazzo: 1.1.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			expectedStderr: []string{"No upgrade needed - document is already at version 1.1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := bytes.NewBuffer(nil)
			stderr := bytes.NewBuffer(nil)

			err := upgradeArazzo(t.Context(), bytes.NewBufferString(tt.input), out, stderr)
			require.NoError(t, err, "upgrade should succeed")
			assert.Equal(t, tt.expected, out.String(), "upgraded document should match")
			for _, expected := range tt.expectedStderr {
				assert.Contains(t, stderr.String(), expected, "stderr should contain expected output")
			}
		})
	}
}

func TestUpgradeArazzo_Error(t *testing.T) {
	t.Parallel()

	err := upgradeArazzo(t.Context(), bytes.NewBufferString("arazzo: 2.0.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n"), bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	require.Error(t, err, "upgrade should fail")
	assert.Contains(t, err.Error(), "only Arazzo versions `1.0.x`, `1.1.x` are supported", "error should match")
}
