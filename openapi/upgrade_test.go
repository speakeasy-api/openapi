package openapi_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUpgrade_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		inputFile     string
		expectedFile  string
		options       []openapi.Option[openapi.UpgradeOptions]
		description   string
		targetVersion string
	}{
		{
			name:         "upgrade_3_0_0_yaml",
			inputFile:    "testdata/upgrade/3_0_0.yaml",
			expectedFile: "testdata/upgrade/expected_3_0_0_upgraded.yaml",
			description:  "3.0.0 should upgrade without options",
		},
		{
			name:         "upgrade_3_0_2_json",
			inputFile:    "testdata/upgrade/3_0_2.json",
			expectedFile: "testdata/upgrade/expected_3_0_2_upgraded.json",
			options:      []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeTargetVersion("3.1.2")},
			description:  "3.0.2 should upgrade without options",
		},
		{
			name:         "upgrade_3_0_3_yaml",
			inputFile:    "testdata/upgrade/3_0_3.yaml",
			expectedFile: "testdata/upgrade/expected_3_0_3_upgraded.yaml",
			description:  "3.0.3 should upgrade without options",
		},
		{
			name:         "upgrade_3_1_0_yaml_with_option",
			inputFile:    "testdata/upgrade/3_1_0.yaml",
			expectedFile: "testdata/upgrade/expected_3_1_0_upgraded.yaml",
			options:      []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSameMinorVersion(), openapi.WithUpgradeTargetVersion("3.1.2")},
			description:  "3.1.0 should upgrade with WithUpgradeSameMinorVersion option",
		},
		{
			name:         "upgrade_nullable_schema",
			inputFile:    "testdata/upgrade/minimal_nullable.json",
			expectedFile: "testdata/upgrade/expected_minimal_nullable_upgraded.json",
			options:      nil,
			description:  "nullable schema should upgrade to oneOf without panic",
		},
		{
			name:          "upgrade_3_1_0_with_custom_methods",
			inputFile:     "testdata/upgrade/3_1_0_with_custom_methods.yaml",
			expectedFile:  "testdata/upgrade/expected_3_1_0_with_custom_methods_upgraded.yaml",
			options:       []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeTargetVersion("3.2.0")},
			description:   "3.1.0 with custom HTTP methods should migrate to additionalOperations",
			targetVersion: "3.2.0",
		},
		{
			name:          "upgrade_3_1_0_to_3_2_0_yaml",
			inputFile:     "testdata/upgrade/3_1_0.yaml",
			expectedFile:  "testdata/upgrade/expected_3_2_0_upgraded.yaml",
			options:       []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeTargetVersion("3.2.0")},
			description:   "3.1.0 should upgrade to 3.2.0 with schema transformations",
			targetVersion: "3.2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			// Read and unmarshal original document
			originalFile, err := os.Open(tt.inputFile)
			require.NoError(t, err, "failed to open input file")
			defer originalFile.Close()

			originalDoc, validationErrs, err := openapi.Unmarshal(ctx, originalFile, openapi.WithSkipValidation())
			require.NoError(t, err, "failed to unmarshal original document")
			require.Empty(t, validationErrs, "original document should not have validation errors")

			// Perform upgrade with options
			upgraded, err := openapi.Upgrade(ctx, originalDoc, tt.options...)
			require.NoError(t, err, "upgrade should not fail: %s", tt.description)
			assert.True(t, upgraded, "upgrade should have been performed")

			// Marshal the upgraded document
			var actualBuf bytes.Buffer
			err = openapi.Marshal(ctx, originalDoc, &actualBuf)
			require.NoError(t, err, "failed to marshal upgraded document")
			actualOutput := actualBuf.String()

			// Read expected output
			expectedFile, err := os.Open(tt.expectedFile)
			require.NoError(t, err, "failed to open expected file")
			defer expectedFile.Close()

			expectedBytes, err := io.ReadAll(expectedFile)
			require.NoError(t, err, "failed to read expected file")
			expectedOutput := string(expectedBytes)

			// Compare actual vs expected output
			assert.Equal(t, expectedOutput, actualOutput, "upgraded output should match expected")
		})
	}
}

func TestUpgrade_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		options  []openapi.Option[openapi.UpgradeOptions]
		wantErrs string
	}{
		{
			name:     "2_0_0_with_upgrade_same_minor_no_upgrade",
			version:  "2.0.0",
			options:  []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSameMinorVersion()},
			wantErrs: "cannot upgrade OpenAPI document version from 2.0.0 to 3.2.0: only OpenAPI 3.x.x is supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

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
			_, err := openapi.Upgrade(ctx, doc, tt.options...)
			require.Error(t, err, "upgrade should fail")
			assert.Contains(t, err.Error(), tt.wantErrs)
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
			name:            "already_3_2_0_no_options",
			version:         "3.2.0",
			options:         nil,
			shouldUpgrade:   false,
			expectedVersion: "3.2.0",
		},
		{
			name:            "3_1_0_with_upgrade_same_minor",
			version:         "3.1.0",
			options:         []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSameMinorVersion()},
			shouldUpgrade:   true,
			expectedVersion: openapi.Version,
		},
		{
			name:            "current_version_with_upgrade_same_minor_no_upgrade",
			version:         openapi.Version,
			options:         []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSameMinorVersion()},
			shouldUpgrade:   false,
			expectedVersion: openapi.Version,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

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
			upgraded, err := openapi.Upgrade(ctx, doc, tt.options...)
			require.NoError(t, err, "upgrade should not fail")
			require.Equal(t, tt.shouldUpgrade, upgraded)

			// Check expected version
			assert.Equal(t, tt.expectedVersion, doc.OpenAPI, "version should match expected for %s", tt.name)
		})
	}
}

func TestUpgrade_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

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
	upgraded, err := openapi.Upgrade(ctx, doc1)
	require.NoError(t, err, "upgrade should not fail")
	assert.Equal(t, openapi.Version, doc1.OpenAPI, "upgraded version should be 3.2.0")
	assert.True(t, upgraded, "upgrade should have been performed")

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
	assert.Equal(t, openapi.Version, doc2.OpenAPI, "second doc version should be 3.2.0")

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

func TestUpgradeAdditionalOperations(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Create a document with non-standard HTTP methods
	doc := &openapi.OpenAPI{
		OpenAPI: "3.1.0",
		Info: openapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi.NewPaths(),
	}

	// Add a path with both standard and non-standard methods
	pathItem := openapi.NewPathItem()

	// Standard method
	pathItem.Set(openapi.HTTPMethodGet, &openapi.Operation{
		Summary:   &[]string{"Get operation"}[0],
		Responses: openapi.NewResponses(),
	})

	// Non-standard methods
	pathItem.Set(openapi.HTTPMethod("copy"), &openapi.Operation{
		Summary:   &[]string{"Copy operation"}[0],
		Responses: openapi.NewResponses(),
	})

	pathItem.Set(openapi.HTTPMethod("purge"), &openapi.Operation{
		Summary:   &[]string{"Purge operation"}[0],
		Responses: openapi.NewResponses(),
	})

	doc.Paths.Set("/test", &openapi.ReferencedPathItem{Object: pathItem})

	// Verify initial state
	assert.Equal(t, 3, pathItem.Len(), "should have 3 operations initially")
	assert.Nil(t, pathItem.AdditionalOperations, "additionalOperations should be nil initially")
	assert.NotNil(t, pathItem.GetOperation(openapi.HTTPMethod("copy")), "copy operation should exist in main map")
	assert.NotNil(t, pathItem.GetOperation(openapi.HTTPMethod("purge")), "purge operation should exist in main map")

	// Perform upgrade to 3.2.0
	upgraded, err := openapi.Upgrade(ctx, doc, openapi.WithUpgradeTargetVersion("3.2.0"))
	require.NoError(t, err, "upgrade should not fail")
	assert.True(t, upgraded, "upgrade should have been performed")
	assert.Equal(t, "3.2.0", doc.OpenAPI, "version should be 3.2.0")

	// Verify migration results
	assert.Equal(t, 1, pathItem.Len(), "should have only 1 operation in main map after migration")
	assert.NotNil(t, pathItem.AdditionalOperations, "additionalOperations should be initialized")
	assert.Equal(t, 2, pathItem.AdditionalOperations.Len(), "should have 2 operations in additionalOperations")

	// Verify standard method remains in main map
	assert.NotNil(t, pathItem.GetOperation(openapi.HTTPMethodGet), "get operation should remain in main map")

	// Verify non-standard methods are moved to additionalOperations
	assert.Nil(t, pathItem.GetOperation(openapi.HTTPMethod("copy")), "copy operation should be removed from main map")
	assert.Nil(t, pathItem.GetOperation(openapi.HTTPMethod("purge")), "purge operation should be removed from main map")

	copyOp, exists := pathItem.AdditionalOperations.Get("copy")
	assert.True(t, exists, "copy operation should exist in additionalOperations")
	assert.NotNil(t, copyOp, "copy operation should not be nil")
	assert.Equal(t, "Copy operation", *copyOp.Summary, "copy operation summary should be preserved")

	purgeOp, exists := pathItem.AdditionalOperations.Get("purge")
	assert.True(t, exists, "purge operation should exist in additionalOperations")
	assert.NotNil(t, purgeOp, "purge operation should not be nil")
	assert.Equal(t, "Purge operation", *purgeOp.Summary, "purge operation summary should be preserved")
}

func TestUpgradeTagGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupDoc    func() *openapi.OpenAPI
		validate    func(t *testing.T, doc *openapi.OpenAPI)
		wantErr     bool
		errContains string
	}{
		{
			name: "basic_x_tagGroups_migration",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")

				// Add existing child tags
				doc.Tags = []*openapi.Tag{
					{Name: "books"},
					{Name: "cds"},
					{Name: "giftcards"},
				}

				// Add x-tagGroups extension
				doc.Extensions = createXTagGroupsExtension([]map[string]interface{}{
					{
						"name": "Products",
						"tags": []interface{}{"books", "cds", "giftcards"},
					},
				})

				return doc
			},
			validate: func(t *testing.T, doc *openapi.OpenAPI) {
				t.Helper()
				// Should have 4 tags total (3 existing + 1 new parent)
				assert.Len(t, doc.Tags, 4, "should have 4 tags after migration")

				// Find parent tag
				var parentTag *openapi.Tag
				for _, tag := range doc.Tags {
					if tag.Name == "Products" {
						parentTag = tag
						break
					}
				}
				require.NotNil(t, parentTag, "parent tag should exist")
				assert.Equal(t, "Products", *parentTag.Summary, "parent summary should be set")
				assert.Equal(t, "nav", *parentTag.Kind, "parent kind should be nav")

				// Verify child tag parent assignments
				childNames := []string{"books", "cds", "giftcards"}
				for _, childName := range childNames {
					var childTag *openapi.Tag
					for _, tag := range doc.Tags {
						if tag.Name == childName {
							childTag = tag
							break
						}
					}
					require.NotNil(t, childTag, "child tag %s should exist", childName)
					require.NotNil(t, childTag.Parent, "child tag %s should have parent", childName)
					assert.Equal(t, "Products", *childTag.Parent, "child tag %s should have correct parent", childName)
				}

				// x-tagGroups extension should be removed
				_, exists := doc.Extensions.Get("x-tagGroups")
				assert.False(t, exists, "x-tagGroups extension should be removed")
			},
		},
		{
			name: "existing_parent_tag",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")

				// Add existing parent tag with different kind
				existingKind := "category"
				doc.Tags = []*openapi.Tag{
					{Name: "Products", Kind: &existingKind},
					{Name: "books"},
				}

				doc.Extensions = createXTagGroupsExtension([]map[string]interface{}{
					{
						"name": "Products",
						"tags": []interface{}{"books"},
					},
				})

				return doc
			},
			validate: func(t *testing.T, doc *openapi.OpenAPI) {
				t.Helper()
				// Find parent tag
				var parentTag *openapi.Tag
				for _, tag := range doc.Tags {
					if tag.Name == "Products" {
						parentTag = tag
						break
					}
				}
				require.NotNil(t, parentTag, "parent tag should exist")
				// Kind should remain unchanged when parent already exists
				assert.Equal(t, "category", *parentTag.Kind, "existing parent kind should be preserved")

				// Child should have parent set
				var childTag *openapi.Tag
				for _, tag := range doc.Tags {
					if tag.Name == "books" {
						childTag = tag
						break
					}
				}
				require.NotNil(t, childTag, "child tag should exist")
				require.NotNil(t, childTag.Parent, "child tag should have parent")
				assert.Equal(t, "Products", *childTag.Parent, "child should have correct parent")
			},
		},
		{
			name: "missing_child_tags_created",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")

				// No existing tags
				doc.Tags = []*openapi.Tag{}

				doc.Extensions = createXTagGroupsExtension([]map[string]interface{}{
					{
						"name": "Products",
						"tags": []interface{}{"books", "electronics"},
					},
				})

				return doc
			},
			validate: func(t *testing.T, doc *openapi.OpenAPI) {
				t.Helper()
				// Should have 3 tags (1 parent + 2 children)
				assert.Len(t, doc.Tags, 3, "should have 3 tags after migration")

				// All tags should exist
				tagNames := []string{"Products", "books", "electronics"}
				for _, name := range tagNames {
					found := false
					for _, tag := range doc.Tags {
						if tag.Name == name {
							found = true
							break
						}
					}
					assert.True(t, found, "tag %s should exist", name)
				}
			},
		},
		{
			name: "multiple_groups",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")
				doc.Tags = []*openapi.Tag{}

				doc.Extensions = createXTagGroupsExtension([]map[string]interface{}{
					{
						"name": "Products",
						"tags": []interface{}{"books", "cds"},
					},
					{
						"name": "Support",
						"tags": []interface{}{"help", "contact"},
					},
				})

				return doc
			},
			validate: func(t *testing.T, doc *openapi.OpenAPI) {
				t.Helper()
				// Should have 6 tags (2 parents + 4 children)
				assert.Len(t, doc.Tags, 6, "should have 6 tags after migration")

				// Verify both parent tags exist
				parentNames := []string{"Products", "Support"}
				for _, parentName := range parentNames {
					var parentTag *openapi.Tag
					for _, tag := range doc.Tags {
						if tag.Name == parentName {
							parentTag = tag
							break
						}
					}
					require.NotNil(t, parentTag, "parent tag %s should exist", parentName)
					assert.Equal(t, "nav", *parentTag.Kind, "parent %s should have nav kind", parentName)
				}

				// Verify child relationships
				childParentMap := map[string]string{
					"books":   "Products",
					"cds":     "Products",
					"help":    "Support",
					"contact": "Support",
				}

				for childName, expectedParent := range childParentMap {
					var childTag *openapi.Tag
					for _, tag := range doc.Tags {
						if tag.Name == childName {
							childTag = tag
							break
						}
					}
					require.NotNil(t, childTag, "child tag %s should exist", childName)
					require.NotNil(t, childTag.Parent, "child tag %s should have parent", childName)
					assert.Equal(t, expectedParent, *childTag.Parent, "child %s should have correct parent", childName)
				}
			},
		},
		{
			name: "no_x_tagGroups_extension",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")
				doc.Tags = []*openapi.Tag{
					{Name: "existing"},
				}
				// No x-tagGroups extension
				return doc
			},
			validate: func(t *testing.T, doc *openapi.OpenAPI) {
				t.Helper()
				// Should remain unchanged
				assert.Len(t, doc.Tags, 1, "should have 1 tag")
				assert.Equal(t, "existing", doc.Tags[0].Name, "existing tag should remain")
				assert.Nil(t, doc.Tags[0].Parent, "existing tag should have no parent")
			},
		},
		{
			name: "empty_x_tagGroups",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")
				doc.Extensions = createXTagGroupsExtension([]map[string]interface{}{})
				return doc
			},
			validate: func(t *testing.T, doc *openapi.OpenAPI) {
				t.Helper()
				// Should remove empty extension
				_, exists := doc.Extensions.Get("x-tagGroups")
				assert.False(t, exists, "empty x-tagGroups should be removed")
			},
		},
		{
			name: "conflicting_parent_assignment",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")

				// Add tag with existing parent
				existingParent := "ExistingParent"
				doc.Tags = []*openapi.Tag{
					{Name: "ExistingParent"},
					{Name: "books", Parent: &existingParent},
				}

				// Try to assign different parent
				doc.Extensions = createXTagGroupsExtension([]map[string]interface{}{
					{
						"name": "Products",
						"tags": []interface{}{"books"},
					},
				})

				return doc
			},
			wantErr:     true,
			errContains: "already has parent",
		},
		{
			name: "self_referencing_prevention",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")

				doc.Extensions = createXTagGroupsExtension([]map[string]interface{}{
					{
						"name": "SelfRef",
						"tags": []interface{}{"SelfRef"},
					},
				})

				return doc
			},
			wantErr:     true,
			errContains: "cannot be its own parent",
		},
		{
			name: "invalid_x_tagGroups_format",
			setupDoc: func() *openapi.OpenAPI {
				doc := createTestDocWithVersion("3.1.0")

				// Create malformed extension
				doc.Extensions = extensions.New()
				// This will create an invalid structure that can't be parsed as []TagGroup
				doc.Extensions.Set("x-tagGroups", createYAMLNode("invalid string"))

				return doc
			},
			wantErr:     true,
			errContains: "failed to parse x-tagGroups extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc := tt.setupDoc()

			// Perform upgrade to 3.2.0
			_, err := openapi.Upgrade(ctx, doc, openapi.WithUpgradeTargetVersion("3.2.0"))

			if tt.wantErr {
				require.Error(t, err, "should have error")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains, "error should contain expected text")
				}
				return
			}

			require.NoError(t, err, "upgrade should not fail")
			if tt.validate != nil {
				tt.validate(t, doc)
			}
		})
	}
}

// Helper functions for test setup

//nolint:unparam // version parameter kept for flexibility even though currently only used with "3.1.0"
func createTestDocWithVersion(version string) *openapi.OpenAPI {
	return &openapi.OpenAPI{
		OpenAPI: version,
		Info: openapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi.NewPaths(),
	}
}

func createXTagGroupsExtension(groups []map[string]interface{}) *extensions.Extensions {
	exts := extensions.New()
	exts.Set("x-tagGroups", createYAMLNode(groups))
	return exts
}

func createYAMLNode(value interface{}) *yaml.Node {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		panic(err)
	}
	return &node
}
