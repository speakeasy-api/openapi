package arazzo_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// The fixtures in testdata/spec/v1.1 are the official Arazzo 1.1 JSON Schema test fixtures from
// https://github.com/OAI/Arazzo-Specification/tree/22be335abb8b70f33ac328290adf52edeef8739e/tests/schema.
//
// The JSON Schema can't express every rule in the specification text, so a small number of fixtures are
// "characterization" fixtures that pass the schema but violate the specification. Where the specification
// text and the schema disagree this library follows the specification text, those fixtures are listed below
// with the reason they are expected to produce validation errors.
var specPassFixtureDeviations = map[string]string{
	// Characterization fixtures: documented upstream as passing the schema while violating the specification.
	"characterization-async-invalid-channelPath-string.arazzo.yaml":     "channelPath must be a `{$sourceDescriptions.<name>.url}#/json/pointer` expression",
	"characterization-asyncapi-operationId-without-action.arazzo.yaml":  "action is REQUIRED for AsyncAPI steps",
	"characterization-openapi-invalid-operationPath-string.arazzo.yaml": "operationPath must be a `{$sourceDescriptions.<name>.url}#/json/pointer` expression",
	"characterization-openapi-operationId-with-action.arazzo.yaml":      "action only applies to AsyncAPI steps",
	"characterization-openapi-receive-with-correlationId.arazzo.yaml":   "action and correlationId only apply to AsyncAPI steps",
	"characterization-workflow-step-with-requestBody.arazzo.yaml":       "requestBody only applies to steps targeting an operation",

	// Fixtures that are only checked against the JSON Schema upstream, which doesn't validate runtime expressions,
	// JSONPath syntax or references, and that violate the specification text.
	"async-with-channel-path.arazzo.yaml":            "channelPath must be a `{$sourceDescriptions.<name>.url}#/json/pointer` expression, not a bare name",
	"openapi-operation-path.arazzo.yaml":             "`$sourceDescriptions.userAPI#/...` doesn't match the source-reference ABNF, operationPath must use `{$sourceDescriptions.<name>.url}#/...`",
	"openapi-success-criteria.arazzo.yaml":           "sourceDescription names are case-sensitive, `userAPI` doesn't match `UserAPI`",
	"workflow-success-criteria.arazzo.yaml":          "sourceDescription names are case-sensitive, `userAPI` doesn't match `UserAPI`",
	"payload-replacement-using-selector.arazzo.yaml": "`$steps.previous.outputs` doesn't match the steps-reference ABNF which requires an output name",
	"async-depends-on.arazzo.yaml":                   "`$message.payload.userId` doesn't match the payload-reference ABNF which only allows a JSON Pointer after `#`",
	"async-depends-on-cross-workflow.arazzo.yaml":    "`$message.payload.userId` is invalid and `$workflows.<id>.steps...` doesn't match the workflows-reference ABNF",
	"openapi-and-asyncapi.arazzo.yaml":               "`$message.payload.orderId` doesn't match the payload-reference ABNF which only allows a JSON Pointer after `#`",
	"bnpl-example.yaml":                              "`$response.body.<name>` and `$response.headers.<name>` don't match the source ABNF",
	"asyncapi-success-criteria.arazzo.yaml":          "`$.status == 'accepted'` isn't a valid RFC 9535 JSONPath query, the default for `jsonpath` criteria",
	"oauth-example.yaml":                             "`$.access_token != null` isn't a valid RFC 9535 JSONPath query, the default for `jsonpath` criteria",
}

func TestArazzo_SpecFixtures_Pass_Success(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/spec/v1.1/pass/*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, files, "should find spec pass fixtures")

	for _, file := range files {
		name := filepath.Base(file)
		if _, ok := specPassFixtureDeviations[name]; ok {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f, err := os.Open(file)
			require.NoError(t, err)
			defer f.Close()

			_, validationErrs, err := arazzo.Unmarshal(t.Context(), f)
			require.NoError(t, err, "unmarshal should succeed")
			assert.Empty(t, validationErrs, "spec pass fixture should be valid")
		})
	}
}

func TestArazzo_SpecFixtures_RoundTrip_Success(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/spec/v1.1/pass/*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, files, "should find spec pass fixtures")

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			data, err := os.ReadFile(file)
			require.NoError(t, err)

			a, _, err := arazzo.Unmarshal(ctx, bytes.NewReader(data))
			require.NoError(t, err, "unmarshal should succeed")

			outBuf := bytes.NewBuffer([]byte{})
			require.NoError(t, arazzo.Marshal(ctx, a, outBuf), "marshal should succeed")

			// The fixtures use a variety of YAML formatting styles, so compare the decoded data rather than the
			// bytes to ensure no fields are lost or changed.
			var expected, actual any
			require.NoError(t, yaml.Unmarshal(data, &expected))
			require.NoError(t, yaml.Unmarshal(outBuf.Bytes(), &actual))
			assert.Equal(t, expected, actual, "round tripped document should match the original")
		})
	}
}

func TestArazzo_SpecFixtures_PassDeviations_Error(t *testing.T) {
	t.Parallel()

	for name, reason := range specPassFixtureDeviations {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f, err := os.Open(filepath.Join("testdata/spec/v1.1/pass", name))
			require.NoError(t, err)
			defer f.Close()

			_, validationErrs, err := arazzo.Unmarshal(t.Context(), f)
			require.NoError(t, err, "unmarshal should succeed")
			assert.NotEmpty(t, validationErrs, "fixture should be invalid per the specification text: %s", reason)
		})
	}
}

func TestArazzo_SpecFixtures_Fail_Error(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/spec/v1.1/fail/*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, files, "should find spec fail fixtures")

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()

			f, err := os.Open(file)
			require.NoError(t, err)
			defer f.Close()

			_, validationErrs, err := arazzo.Unmarshal(t.Context(), f)
			require.NoError(t, err, "unmarshal should succeed, invalid documents should be reported as validation errors")
			assert.NotEmpty(t, validationErrs, "spec fail fixture should produce validation errors")
		})
	}
}
