package openapi_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgrade_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputFile    string
		expectedFile string
		options      []openapi.Option[openapi.UpgradeOptions]
		description  string
	}{
		{
			name:         "upgrade_3_0_0_yaml",
			inputFile:    "testdata/upgrade/3_0_0.yaml",
			expectedFile: "testdata/upgrade/expected_3_0_0_upgraded.yaml",
			options:      nil,
			description:  "3.0.0 should upgrade without options",
		},
		{
			name:         "upgrade_3_0_2_json",
			inputFile:    "testdata/upgrade/3_0_2.json",
			expectedFile: "testdata/upgrade/expected_3_0_2_upgraded.json",
			options:      nil,
			description:  "3.0.2 should upgrade without options",
		},
		{
			name:         "upgrade_3_0_3_yaml",
			inputFile:    "testdata/upgrade/3_0_3.yaml",
			expectedFile: "testdata/upgrade/expected_3_0_3_upgraded.yaml",
			options:      nil,
			description:  "3.0.3 should upgrade without options",
		},
		{
			name:         "upgrade_3_1_0_yaml_with_option",
			inputFile:    "testdata/upgrade/3_1_0.yaml",
			expectedFile: "testdata/upgrade/expected_3_1_0_upgraded.yaml",
			options:      []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSamePatchVersion()},
			description:  "3.1.0 should upgrade with WithUpgradeSamePatchVersion option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Read and unmarshal original document
			originalFile, err := os.Open(tt.inputFile)
			require.NoError(t, err, "failed to open input file")
			defer originalFile.Close() // nolint:errcheck

			originalDoc, validationErrs, err := openapi.Unmarshal(ctx, originalFile, openapi.WithSkipValidation())
			require.NoError(t, err, "failed to unmarshal original document")
			require.Empty(t, validationErrs, "original document should not have validation errors")

			// Perform upgrade with options
			err = openapi.Upgrade(ctx, originalDoc, tt.options...)
			require.NoError(t, err, "upgrade should not fail: %s", tt.description)

			// Marshal the upgraded document
			var actualBuf bytes.Buffer
			err = openapi.Marshal(ctx, originalDoc, &actualBuf)
			require.NoError(t, err, "failed to marshal upgraded document")
			actualOutput := actualBuf.String()

			// Read expected output
			expectedFile, err := os.Open(tt.expectedFile)
			require.NoError(t, err, "failed to open expected file")
			defer expectedFile.Close() // nolint:errcheck

			expectedBytes, err := io.ReadAll(expectedFile)
			require.NoError(t, err, "failed to read expected file")
			expectedOutput := string(expectedBytes)

			// Compare actual vs expected output
			assert.Equal(t, expectedOutput, actualOutput, "upgraded output should match expected")
		})
	}
}

func TestUpgrade_NoUpgradeNeeded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		version         string
		options         []openapi.Option[openapi.UpgradeOptions]
		shouldUpgrade   bool
		expectedVersion string
	}{
		{
			name:            "already_3_1_0_no_options",
			version:         "3.1.0",
			options:         nil,
			shouldUpgrade:   false,
			expectedVersion: "3.1.0",
		},
		{
			name:            "already_3_1_1_no_options",
			version:         "3.1.1",
			options:         nil,
			shouldUpgrade:   false,
			expectedVersion: "3.1.1",
		},
		{
			name:            "not_3_0_x_no_options",
			version:         "2.0.0",
			options:         nil,
			shouldUpgrade:   false,
			expectedVersion: "2.0.0",
		},
		{
			name:            "3_1_0_with_upgrade_same_patch",
			version:         "3.1.0",
			options:         []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSamePatchVersion()},
			shouldUpgrade:   true,
			expectedVersion: openapi.Version,
		},
		{
			name:            "3_1_1_with_upgrade_same_patch_no_upgrade",
			version:         "3.1.1",
			options:         []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSamePatchVersion()},
			shouldUpgrade:   false,
			expectedVersion: "3.1.1",
		},
		{
			name:            "2_0_0_with_upgrade_same_patch_no_upgrade",
			version:         "2.0.0",
			options:         []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSamePatchVersion()},
			shouldUpgrade:   false,
			expectedVersion: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Create a simple document with the specified version
			doc := &openapi.OpenAPI{
				OpenAPI: tt.version,
				Info: openapi.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: openapi.NewPaths(),
			}

			// Perform upgrade with options
			err := openapi.Upgrade(ctx, doc, tt.options...)
			require.NoError(t, err, "upgrade should not fail")

			// Check expected version
			assert.Equal(t, tt.expectedVersion, doc.OpenAPI, "version should match expected for %s", tt.name)
		})
	}
}

func TestUpgrade_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test with a comprehensive document that exercises all upgrade paths
	yamlDoc := `
openapi: 3.0.1
info:
  title: Round Trip Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
components:
  schemas:
    TestSchema:
      type: object
      nullable: true
      properties:
        simpleExample:
          type: string
          example: "test value"
        exclusiveMinMax:
          type: number
          minimum: 0
          exclusiveMinimum: true
          maximum: 100
          exclusiveMaximum: false
        nullableAnyOf:
          anyOf:
            - type: string
            - type: integer
          nullable: true
        nullableOneOf:
          oneOf:
            - type: string
            - type: boolean
          nullable: true
        simpleNullable:
          type: string
          nullable: true
`

	// First unmarshal
	doc1, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yamlDoc), openapi.WithSkipValidation())
	require.NoError(t, err, "first unmarshal should not fail")
	require.Empty(t, validationErrs, "first unmarshal should not have validation errors")
	assert.Equal(t, "3.0.1", doc1.OpenAPI, "original version should be 3.0.1")

	// Upgrade (no options needed for 3.0.x documents)
	err = openapi.Upgrade(ctx, doc1)
	require.NoError(t, err, "upgrade should not fail")
	assert.Equal(t, openapi.Version, doc1.OpenAPI, "upgraded version should be 3.1.1")

	// Marshal back
	var buf1 bytes.Buffer
	err = openapi.Marshal(ctx, doc1, &buf1)
	require.NoError(t, err, "first marshal should not fail")

	// Store the marshalled content for reuse
	marshalledContent := buf1.String()
	require.NotEmpty(t, marshalledContent, "first marshal should produce content")

	// Unmarshal again using a new reader
	doc2, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(marshalledContent))
	require.NoError(t, err, "second unmarshal should not fail")
	require.Empty(t, validationErrs, "second unmarshal should not have validation errors")
	assert.Equal(t, openapi.Version, doc2.OpenAPI, "second doc version should be 3.1.1")

	// Marshal again
	var buf2 bytes.Buffer
	err = openapi.Marshal(ctx, doc2, &buf2)
	require.NoError(t, err, "second marshal should not fail")

	// The two marshalled outputs should be identical (idempotent)
	secondMarshalledContent := buf2.String()
	if !assert.Equal(t, marshalledContent, secondMarshalledContent, "marshalled outputs should be identical") {
		t.Logf("First marshal output:\n%s", marshalledContent)
		t.Logf("Second marshal output:\n%s", secondMarshalledContent)
	}

	// Verify specific upgrades were applied
	require.NotNil(t, doc2.Components, "components should exist")
	require.NotNil(t, doc2.Components.Schemas, "schemas should exist")

	testSchema, exists := doc2.Components.Schemas.Get("TestSchema")
	require.True(t, exists, "TestSchema should exist")
	require.True(t, testSchema.IsLeft(), "TestSchema should be a schema object")

	schema := testSchema.GetLeft()

	// Check nullable conversion
	schemaTypes := schema.GetType()
	assert.Contains(t, schemaTypes, oas3.SchemaTypeObject, "should have object type")
	assert.Contains(t, schemaTypes, oas3.SchemaTypeNull, "should have null type")

	// Check example -> examples conversion
	simpleExampleProp, exists := schema.GetProperties().Get("simpleExample")
	require.True(t, exists, "simpleExample property should exist")
	require.True(t, simpleExampleProp.IsLeft(), "simpleExample should be a schema object")

	simpleExample := simpleExampleProp.GetLeft()
	assert.Nil(t, simpleExample.Example, "example should be nil")
	assert.NotEmpty(t, simpleExample.Examples, "examples should not be empty")
}
