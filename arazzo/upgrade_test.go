package arazzo_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const upgradeTestDoc10 = `arazzo: 1.0.1
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
          - condition: $statusCode == 200
          - context: $response.body
            condition: $[?@.status == 'available']
            type: jsonpath
          - context: $response.body
            condition: //pet/status
            type: xpath
          - context: $response.body
            condition: ^available$
            type: regex
          - context: $response.body
            condition: $.status
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
        onSuccess:
          - name: done
            type: end
            criteria:
              - context: $response.body
                condition: $.done
                type: jsonpath
        onFailure:
          - name: stop
            type: end
            criteria:
              - context: $response.body
                condition: //error
                type: xpath
    successActions:
      - name: workflowDone
        type: end
        criteria:
          - context: $response.body
            condition: $.ok
            type: jsonpath
    failureActions:
      - name: workflowFailed
        type: end
        criteria:
          - context: $response.body
            condition: $.failed
            type: jsonpath
components:
  successActions:
    finish:
      name: finish
      type: end
      criteria:
        - context: $response.body
          condition: //finished
          type: xpath
  failureActions:
    retry:
      name: retry
      type: retry
      criteria:
        - context: $response.body
          condition: $.retry
          type: jsonpath
`

const upgradeTestDoc11 = `arazzo: 1.1.0
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
          - condition: $statusCode == 200
          - context: $response.body
            condition: $[?@.status == 'available']
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
          - context: $response.body
            condition: //pet/status
            type:
              type: xpath
              version: xpath-30
          - context: $response.body
            condition: ^available$
            type: regex
          - context: $response.body
            condition: $.status
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
        onSuccess:
          - name: done
            type: end
            criteria:
              - context: $response.body
                condition: $.done
                type:
                  type: jsonpath
                  version: draft-goessner-dispatch-jsonpath-00
        onFailure:
          - name: stop
            type: end
            criteria:
              - context: $response.body
                condition: //error
                type:
                  type: xpath
                  version: xpath-30
    successActions:
      - name: workflowDone
        type: end
        criteria:
          - context: $response.body
            condition: $.ok
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
    failureActions:
      - name: workflowFailed
        type: end
        criteria:
          - context: $response.body
            condition: $.failed
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
components:
  successActions:
    finish:
      name: finish
      type: end
      criteria:
        - context: $response.body
          condition: //finished
          type:
            type: xpath
            version: xpath-30
  failureActions:
    retry:
      name: retry
      type: retry
      criteria:
        - context: $response.body
          condition: $.retry
          type:
            type: jsonpath
            version: draft-goessner-dispatch-jsonpath-00
`

func TestUpgrade_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		input            string
		opts             []arazzo.Option[arazzo.UpgradeOptions]
		expectedUpgraded bool
		expected         string
	}{
		{
			name:             "upgrades 1.0.x to 1.1.0 and pins criteria versions",
			input:            upgradeTestDoc10,
			expectedUpgraded: true,
			expected:         upgradeTestDoc11,
		},
		{
			name:             "no change when already at the target version",
			input:            upgradeTestDoc11,
			expectedUpgraded: false,
			expected:         upgradeTestDoc11,
		},
		{
			name:             "patch upgrade within 1.0 only changes the version",
			input:            "arazzo: 1.0.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			opts:             []arazzo.Option[arazzo.UpgradeOptions]{arazzo.WithUpgradeTargetVersion("1.0.1"), arazzo.WithUpgradeSameMinorVersion()},
			expectedUpgraded: true,
			expected:         "arazzo: 1.0.1\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
		},
		{
			name:             "patch upgrade skipped without same minor version option",
			input:            "arazzo: 1.0.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			opts:             []arazzo.Option[arazzo.UpgradeOptions]{arazzo.WithUpgradeTargetVersion("1.0.1")},
			expectedUpgraded: false,
			expected:         "arazzo: 1.0.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
		},
		{
			name:             "prerelease suffix is accepted",
			input:            "arazzo: 1.0.1-rc1\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			expectedUpgraded: true,
			expected:         "arazzo: 1.1.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
		},
		{
			name:             "prerelease target version suffix is preserved",
			input:            "arazzo: 1.0.1\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			opts:             []arazzo.Option[arazzo.UpgradeOptions]{arazzo.WithUpgradeTargetVersion("1.1.0-rc1")},
			expectedUpgraded: true,
			expected:         "arazzo: 1.1.0-rc1\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(tt.input), arazzo.WithSkipValidation())
			require.NoError(t, err, "unmarshal should succeed")

			upgraded, err := arazzo.Upgrade(ctx, doc, tt.opts...)
			require.NoError(t, err, "upgrade should succeed")
			assert.Equal(t, tt.expectedUpgraded, upgraded, "upgraded should match")

			out := bytes.NewBuffer(nil)
			require.NoError(t, arazzo.Marshal(ctx, doc, out), "marshal should succeed")
			assert.Equal(t, tt.expected, out.String(), "upgraded document should match")
		})
	}
}

func TestUpgrade_NilDocument_Success(t *testing.T) {
	t.Parallel()

	upgraded, err := arazzo.Upgrade(t.Context(), nil)
	require.NoError(t, err, "upgrade should succeed")
	assert.False(t, upgraded, "nil document should not be upgraded")
}

func TestUpgrade_UnknownCriterionType_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `arazzo: 1.0.1
info:
  title: Upgrade
  version: 1.0.0
workflows:
  - workflowId: workflow1
    steps:
      - stepId: step1
        operationId: getPet
        successCriteria:
          - context: $response.body
            condition: $.ok
            type: unknown
`

	doc, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(input), arazzo.WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")

	upgraded, err := arazzo.Upgrade(ctx, doc)
	require.NoError(t, err, "upgrade should succeed")
	assert.True(t, upgraded, "document should be upgraded")

	out := bytes.NewBuffer(nil)
	require.NoError(t, arazzo.Marshal(ctx, doc, out), "marshal should succeed")
	assert.Contains(t, out.String(), "type: unknown", "unknown criterion types should be left unchanged")
}

func TestUpgrade_UpgradedDocumentIsValid_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(upgradeTestDoc10), arazzo.WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")

	_, err = arazzo.Upgrade(ctx, doc)
	require.NoError(t, err, "upgrade should succeed")

	out := bytes.NewBuffer(nil)
	require.NoError(t, arazzo.Marshal(ctx, doc, out), "marshal should succeed")

	_, validationErrs, err := arazzo.Unmarshal(ctx, out)
	require.NoError(t, err, "unmarshal of upgraded document should succeed")
	assert.Empty(t, validationErrs, "upgraded document should be valid")
}

func TestUpgrade_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		opts          []arazzo.Option[arazzo.UpgradeOptions]
		expectedError string
	}{
		{
			name:          "downgrade is rejected",
			input:         "arazzo: 1.1.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			opts:          []arazzo.Option[arazzo.UpgradeOptions]{arazzo.WithUpgradeTargetVersion("1.0.1")},
			expectedError: "cannot downgrade Arazzo document version from 1.1.0 to 1.0.1",
		},
		{
			name:          "unsupported document version is rejected",
			input:         "arazzo: 2.0.0\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			expectedError: "cannot upgrade Arazzo document version 2.0.0: only Arazzo versions `1.0.x`, `1.1.x` are supported",
		},
		{
			name:          "unsupported target version is rejected",
			input:         "arazzo: 1.0.1\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			opts:          []arazzo.Option[arazzo.UpgradeOptions]{arazzo.WithUpgradeTargetVersion("1.2.0")},
			expectedError: "cannot upgrade Arazzo document to version 1.2.0: only Arazzo versions `1.0.x`, `1.1.x` are supported",
		},
		{
			name:          "invalid document version is rejected",
			input:         "arazzo: one\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			expectedError: "invalid Arazzo document version `one`",
		},
		{
			name:          "invalid target version is rejected",
			input:         "arazzo: 1.0.1\ninfo:\n  title: Upgrade\n  version: 1.0.0\n",
			opts:          []arazzo.Option[arazzo.UpgradeOptions]{arazzo.WithUpgradeTargetVersion("one")},
			expectedError: "invalid target Arazzo version `one`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(tt.input), arazzo.WithSkipValidation())
			require.NoError(t, err, "unmarshal should succeed")

			upgraded, err := arazzo.Upgrade(ctx, doc, tt.opts...)
			require.Error(t, err, "upgrade should fail")
			assert.Contains(t, err.Error(), tt.expectedError, "error should match")
			assert.False(t, upgraded, "document should not be upgraded")
		})
	}
}
