package arazzo

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validateAsyncAPISourceInput = `arazzo: 1.1.0
info:
  title: Events
  version: 1.0.0
sourceDescriptions:
  - name: events
    url: ./events.yaml
    type: asyncapi
workflows:
  - workflowId: publish
    steps:
      - stepId: send
        operationId: $sourceDescriptions.events.publishPet
        action: send
`

func TestValidateArazzo_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		file           string
		stdin          string
		skipSources    bool
		expectedStderr []string
	}{
		{
			name:           "valid against its sources",
			file:           "../../../../arazzo/testdata/sources/valid.arazzo.yaml",
			expectedStderr: []string{"Validating Arazzo document: ../../../../arazzo/testdata/sources/valid.arazzo.yaml", "✅ Arazzo document is valid - 0 errors\n"},
		},
		{
			name:           "source checks skipped",
			file:           "../../../../arazzo/testdata/sources/invalid.arazzo.yaml",
			skipSources:    true,
			expectedStderr: []string{"✅ Arazzo document is valid - 0 errors\n"},
		},
		{
			name:           "hints don't fail validation",
			file:           "-",
			stdin:          validateAsyncAPISourceInput,
			expectedStderr: []string{"Validating Arazzo document from stdin", "✅ Arazzo document is valid - 0 errors, 1 warnings or hints:", "1. [7:10] hint validation-invalid-reference sourceDescription `events` is an AsyncAPI document"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			err := validateArazzo(t.Context(), tt.file, tt.skipSources, strings.NewReader(tt.stdin), &stderr)
			require.NoError(t, err, "validation should succeed")
			for _, expected := range tt.expectedStderr {
				assert.Contains(t, stderr.String(), expected, "stderr should contain expected output")
			}
		})
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestValidateArazzo_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		file           string
		stdin          string
		expectedErr    string
		expectedStderr []string
	}{
		{
			name:           "invalid against its sources",
			file:           "../../../../arazzo/testdata/sources/invalid.arazzo.yaml",
			expectedErr:    "arazzo document validation failed",
			expectedStderr: []string{"❌ Arazzo document is invalid - 17 errors, 2 warnings or hints:", "error validation-operation-not-found step.operationId `missingOperation` not found in sourceDescription `petstore`"},
		},
		{
			name:        "missing file",
			file:        "testdata/missing.arazzo.yaml",
			expectedErr: "failed to open file",
		},
		{
			name:        "unparseable document",
			file:        "-",
			stdin:       "[",
			expectedErr: "failed to unmarshal file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			err := validateArazzo(t.Context(), tt.file, false, strings.NewReader(tt.stdin), &stderr)
			require.Error(t, err, "validation should fail")
			assert.Contains(t, err.Error(), tt.expectedErr, "error should match")
			for _, expected := range tt.expectedStderr {
				assert.Contains(t, stderr.String(), expected, "stderr should contain expected output")
			}
		})
	}
}

func TestValidateArazzo_StdinReadError(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := validateArazzo(t.Context(), "-", false, errReader{}, &stderr)
	require.Error(t, err, "validation should fail")
	assert.Contains(t, err.Error(), "failed to read stdin", "error should match")
}

func TestSeverityOf_Success(t *testing.T) {
	t.Parallel()

	assert.Equal(t, validation.SeverityError, severityOf(errors.New("plain error")), "plain errors should be treated as errors")
	assert.Equal(t, validation.SeverityWarning, severityOf(&validation.Error{Severity: validation.SeverityWarning, UnderlyingError: errors.New("warn")}), "validation errors should keep their severity")
}
