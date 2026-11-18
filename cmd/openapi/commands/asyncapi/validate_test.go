package asyncapi

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testdata = "../../../../asyncapi/testdata/"

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(testdata + name)
	require.NoError(t, err, "fixture should load")
	return string(data)
}

func TestValidateAsyncAPI_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		file             string
		stdin            string
		opts             validateOptions
		expectedStderr   []string
		unexpectedStderr []string
	}{
		{
			name:             "valid document",
			file:             testdata + "validate/orders.asyncapi.yaml",
			opts:             validateOptions{resolve: true},
			expectedStderr:   []string{"Validating AsyncAPI document: " + testdata + "validate/orders.asyncapi.yaml", "✅ AsyncAPI document is valid - 0 errors\n"},
			unexpectedStderr: []string{"Not checked"},
		},
		{
			name: "references to other documents aren't followed by default",
			file: testdata + "validate/orders.asyncapi.yaml",
			expectedStderr: []string{
				"✅ AsyncAPI document is valid - 0 errors, 2 warnings:",
				"1. [10:15] warning validation-invalid-reference reference `./messages.yaml#/components/messages/OrderPlaced` is to another document and external references are disabled",
				"Not checked:\n- 2 references to other documents were not followed, use --resolve to validate them",
			},
		},
		{
			name:           "a broken reference to another document isn't reported without --resolve",
			file:           testdata + "validate/broken-ref.asyncapi.yaml",
			expectedStderr: []string{"✅ AsyncAPI document is valid - 0 errors, 1 warnings:", "1 references to other documents were not followed"},
		},
		{
			name:             "stdin resolved against --base-uri",
			file:             "-",
			stdin:            readTestdata(t, "validate/orders.asyncapi.yaml"),
			opts:             validateOptions{resolve: true, baseURI: testdata + "validate/orders.asyncapi.yaml"},
			expectedStderr:   []string{"Validating AsyncAPI document from stdin", "✅ AsyncAPI document is valid - 0 errors\n"},
			unexpectedStderr: []string{"Note:"},
		},
		{
			name:           "formats that aren't validated are reported as not checked",
			file:           testdata + "streetlights-3.0.yaml",
			expectedStderr: []string{"✅ AsyncAPI document is valid - 0 errors, 1 hints:", "Not checked:\n- 1 schemas or bindings are in formats that are preserved but not validated"},
		},
		{
			name:           "hints don't fail with --strict",
			file:           testdata + "streetlights-3.0.yaml",
			opts:           validateOptions{strict: true},
			expectedStderr: []string{"✅ AsyncAPI document is valid - 0 errors, 1 hints:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			err := validateAsyncAPI(t.Context(), tt.file, tt.opts, strings.NewReader(tt.stdin), &stderr)
			require.NoError(t, err, "validation should succeed")
			for _, expected := range tt.expectedStderr {
				assert.Contains(t, stderr.String(), expected, "stderr should contain expected output")
			}
			for _, unexpected := range tt.unexpectedStderr {
				assert.NotContains(t, stderr.String(), unexpected, "stderr should not contain unexpected output")
			}
		})
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestValidateAsyncAPI_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		file           string
		stdin          string
		opts           validateOptions
		expectedErr    string
		expectedStderr []string
	}{
		{
			name:        "invalid document",
			file:        testdata + "validate/invalid.asyncapi.yaml",
			expectedErr: "asyncapi document validation failed",
			expectedStderr: []string{
				"❌ AsyncAPI document is invalid - 3 errors:",
				"error validation-required-field `info.version` is required",
				"error validation-invalid-reference channel reference `#/channels/missing` target not found",
			},
		},
		{
			name:           "broken reference to another document with --resolve",
			file:           testdata + "validate/broken-ref.asyncapi.yaml",
			opts:           validateOptions{resolve: true},
			expectedErr:    "asyncapi document validation failed",
			expectedStderr: []string{"❌ AsyncAPI document is invalid - 1 errors:", "failed to resolve message reference `./messages.yaml#/components/messages/Missing`"},
		},
		{
			name:        "unsupported version",
			file:        testdata + "validate/v2.asyncapi.yaml",
			expectedErr: "asyncapi document version is not supported",
			expectedStderr: []string{
				"❌ AsyncAPI document is not supported:",
				"asyncapi version `2.6.0` is not supported",
				"The rest of the document was not checked.",
			},
		},
		{
			name:           "warnings fail with --strict",
			file:           testdata + "validate/orders.asyncapi.yaml",
			opts:           validateOptions{strict: true},
			expectedErr:    "asyncapi document has 2 warnings and --strict is set",
			expectedStderr: []string{"✅ AsyncAPI document is valid - 0 errors, 2 warnings:"},
		},
		{
			name:           "stdin without a location can't resolve relative references",
			file:           "-",
			stdin:          readTestdata(t, "validate/orders.asyncapi.yaml"),
			opts:           validateOptions{resolve: true},
			expectedErr:    "asyncapi document validation failed",
			expectedStderr: []string{"Note: the document has no location", "Use --base-uri to set one.", "relative reference `./messages.yaml#/components/messages/OrderPlaced` can't be resolved"},
		},
		{
			name:        "--base-uri without --resolve",
			file:        testdata + "validate/orders.asyncapi.yaml",
			opts:        validateOptions{baseURI: "https://example.com/asyncapi.yaml"},
			expectedErr: "--base-uri is only used to resolve references to other documents, use it with --resolve",
		},
		{
			name:        "missing file",
			file:        "testdata/missing.asyncapi.yaml",
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
			err := validateAsyncAPI(t.Context(), tt.file, tt.opts, strings.NewReader(tt.stdin), &stderr)
			require.Error(t, err, "validation should fail")
			assert.Contains(t, err.Error(), tt.expectedErr, "error should match")
			for _, expected := range tt.expectedStderr {
				assert.Contains(t, stderr.String(), expected, "stderr should contain expected output")
			}
		})
	}
}

func TestValidateAsyncAPI_StdinReadError(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := validateAsyncAPI(t.Context(), "-", validateOptions{}, errReader{}, &stderr)
	require.Error(t, err, "validation should fail")
	assert.Contains(t, err.Error(), "failed to read stdin", "error should match")
}

func TestSummarize_NonValidationErrors_Success(t *testing.T) {
	t.Parallel()

	s := summarize([]error{errors.New("plain error"), &validation.Error{Severity: validation.SeverityHint, UnderlyingError: errors.New("hint")}})
	assert.Equal(t, 1, s.errors, "plain errors should be counted as errors")
	assert.Equal(t, 1, s.hints, "hints should be counted")
}
