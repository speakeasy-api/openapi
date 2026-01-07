package oas3_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaRegistry_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		documentBaseURI string
		expectedBase    string
	}{
		{
			name:            "empty document base",
			documentBaseURI: "",
			expectedBase:    "",
		},
		{
			name:            "absolute URL document base",
			documentBaseURI: "https://example.com/schemas/document.json",
			expectedBase:    "https://example.com/schemas/document.json",
		},
		{
			name:            "URL with trailing slash preserved",
			documentBaseURI: "https://example.com/schemas/",
			expectedBase:    "https://example.com/schemas/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := oas3.NewSchemaRegistry(tt.documentBaseURI)
			require.NotNil(t, registry, "registry should not be nil")
			assert.Equal(t, tt.expectedBase, registry.GetDocumentBaseURI(), "document base URI should match")
		})
	}
}

func TestSchemaRegistry_RegisterSchema_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		documentBase  string
		schemaID      string
		schemaAnchor  string
		parentBaseURI string
		expectedBase  string
	}{
		{
			name:          "schema with absolute $id",
			documentBase:  "https://example.com/doc.json",
			schemaID:      "https://example.com/schemas/user.json",
			schemaAnchor:  "",
			parentBaseURI: "",
			expectedBase:  "https://example.com/schemas/user.json",
		},
		{
			name:          "schema with relative $id resolved against document base",
			documentBase:  "https://example.com/schemas/doc.json",
			schemaID:      "user.json",
			schemaAnchor:  "",
			parentBaseURI: "",
			expectedBase:  "https://example.com/schemas/user.json",
		},
		{
			name:          "schema with relative $id resolved against parent base",
			documentBase:  "https://example.com/doc.json",
			schemaID:      "nested.json",
			schemaAnchor:  "",
			parentBaseURI: "https://example.com/schemas/parent.json",
			expectedBase:  "https://example.com/schemas/nested.json",
		},
		{
			name:          "schema with $anchor only",
			documentBase:  "https://example.com/doc.json",
			schemaID:      "",
			schemaAnchor:  "myAnchor",
			parentBaseURI: "",
			expectedBase:  "https://example.com/doc.json",
		},
		{
			name:          "schema with both $id and $anchor",
			documentBase:  "https://example.com/doc.json",
			schemaID:      "https://example.com/schemas/user.json",
			schemaAnchor:  "address",
			parentBaseURI: "",
			expectedBase:  "https://example.com/schemas/user.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := oas3.NewSchemaRegistry(tt.documentBase)
			schema := createTestJSONSchema(tt.schemaID, tt.schemaAnchor)

			err := registry.RegisterSchema(schema, tt.parentBaseURI)
			require.NoError(t, err, "registration should succeed")

			// Verify base URI was computed correctly
			actualBase := registry.GetBaseURI(schema)
			assert.Equal(t, tt.expectedBase, actualBase, "effective base URI should match")

			// Verify $id lookup works if $id was set
			if tt.schemaID != "" {
				found := registry.LookupByID(tt.expectedBase)
				assert.NotNil(t, found, "schema should be found by $id")
				assert.Equal(t, schema, found, "looked up schema should match registered schema")
			}

			// Verify $anchor lookup works if $anchor was set
			if tt.schemaAnchor != "" {
				found := registry.LookupByAnchor(tt.expectedBase, tt.schemaAnchor)
				assert.NotNil(t, found, "schema should be found by $anchor")
				assert.Equal(t, schema, found, "looked up schema should match registered schema")
			}
		})
	}
}

func TestSchemaRegistry_RegisterSchema_NilSchema_Success(t *testing.T) {
	t.Parallel()

	registry := oas3.NewSchemaRegistry("https://example.com/doc.json")
	err := registry.RegisterSchema(nil, "")
	require.NoError(t, err, "registering nil schema should not error")
}

func TestSchemaRegistry_LookupByID_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		documentBase string
		registeredID string
		lookupURI    string
		shouldFind   bool
	}{
		{
			name:         "exact match",
			documentBase: "https://example.com/doc.json",
			registeredID: "https://example.com/schemas/user.json",
			lookupURI:    "https://example.com/schemas/user.json",
			shouldFind:   true,
		},
		{
			name:         "URI with fragment stripped",
			documentBase: "https://example.com/doc.json",
			registeredID: "https://example.com/schemas/user.json",
			lookupURI:    "https://example.com/schemas/user.json#foo",
			shouldFind:   true,
		},
		{
			name:         "not found",
			documentBase: "https://example.com/doc.json",
			registeredID: "https://example.com/schemas/user.json",
			lookupURI:    "https://example.com/schemas/other.json",
			shouldFind:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := oas3.NewSchemaRegistry(tt.documentBase)
			schema := createTestJSONSchema(tt.registeredID, "")

			err := registry.RegisterSchema(schema, "")
			require.NoError(t, err, "registration should succeed")

			found := registry.LookupByID(tt.lookupURI)
			if tt.shouldFind {
				assert.NotNil(t, found, "schema should be found")
				assert.Equal(t, schema, found, "found schema should match")
			} else {
				assert.Nil(t, found, "schema should not be found")
			}
		})
	}
}

func TestSchemaRegistry_LookupByAnchor_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		documentBase     string
		schemaID         string
		registeredAnchor string
		lookupBaseURI    string
		lookupAnchor     string
		shouldFind       bool
	}{
		{
			name:             "anchor in document base scope",
			documentBase:     "https://example.com/doc.json",
			schemaID:         "",
			registeredAnchor: "myAnchor",
			lookupBaseURI:    "https://example.com/doc.json",
			lookupAnchor:     "myAnchor",
			shouldFind:       true,
		},
		{
			name:             "anchor in nested $id scope",
			documentBase:     "https://example.com/doc.json",
			schemaID:         "https://example.com/schemas/user.json",
			registeredAnchor: "address",
			lookupBaseURI:    "https://example.com/schemas/user.json",
			lookupAnchor:     "address",
			shouldFind:       true,
		},
		{
			name:             "anchor not found in different scope",
			documentBase:     "https://example.com/doc.json",
			schemaID:         "https://example.com/schemas/user.json",
			registeredAnchor: "address",
			lookupBaseURI:    "https://example.com/doc.json",
			lookupAnchor:     "address",
			shouldFind:       false,
		},
		{
			name:             "anchor not found - different anchor name",
			documentBase:     "https://example.com/doc.json",
			schemaID:         "",
			registeredAnchor: "myAnchor",
			lookupBaseURI:    "https://example.com/doc.json",
			lookupAnchor:     "otherAnchor",
			shouldFind:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := oas3.NewSchemaRegistry(tt.documentBase)
			schema := createTestJSONSchema(tt.schemaID, tt.registeredAnchor)

			err := registry.RegisterSchema(schema, "")
			require.NoError(t, err, "registration should succeed")

			found := registry.LookupByAnchor(tt.lookupBaseURI, tt.lookupAnchor)
			if tt.shouldFind {
				assert.NotNil(t, found, "schema should be found by anchor")
				assert.Equal(t, schema, found, "found schema should match")
			} else {
				assert.Nil(t, found, "schema should not be found")
			}
		})
	}
}

func TestSchemaRegistry_GetBaseURI_Success(t *testing.T) {
	t.Parallel()

	t.Run("registered schema returns computed base", func(t *testing.T) {
		t.Parallel()

		registry := oas3.NewSchemaRegistry("https://example.com/doc.json")
		schema := createTestJSONSchema("https://example.com/schemas/user.json", "")

		err := registry.RegisterSchema(schema, "")
		require.NoError(t, err, "registration should succeed")

		base := registry.GetBaseURI(schema)
		assert.Equal(t, "https://example.com/schemas/user.json", base, "should return schema's $id as base")
	})

	t.Run("unregistered schema returns document base", func(t *testing.T) {
		t.Parallel()

		registry := oas3.NewSchemaRegistry("https://example.com/doc.json")
		schema := createTestJSONSchema("", "")

		base := registry.GetBaseURI(schema)
		assert.Equal(t, "https://example.com/doc.json", base, "should return document base for unregistered schema")
	})
}

func TestSchemaRegistry_RegisterSchema_DuplicateID_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		documentBase   string
		firstSchemaID  string
		secondSchemaID string
		expectedError  string
	}{
		{
			name:           "duplicate absolute $id",
			documentBase:   "https://example.com/doc.json",
			firstSchemaID:  "https://example.com/schemas/user.json",
			secondSchemaID: "https://example.com/schemas/user.json",
			expectedError:  `duplicate $id detected: "https://example.com/schemas/user.json" is already registered`,
		},
		{
			name:           "duplicate relative $id resolving to same absolute",
			documentBase:   "https://example.com/schemas/doc.json",
			firstSchemaID:  "user.json",
			secondSchemaID: "https://example.com/schemas/user.json",
			expectedError:  `duplicate $id detected: "https://example.com/schemas/user.json" is already registered`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := oas3.NewSchemaRegistry(tt.documentBase)

			firstSchema := createTestJSONSchema(tt.firstSchemaID, "")
			err := registry.RegisterSchema(firstSchema, "")
			require.NoError(t, err, "first registration should succeed")

			secondSchema := createTestJSONSchema(tt.secondSchemaID, "")
			err = registry.RegisterSchema(secondSchema, "")
			require.Error(t, err, "second registration should fail")
			assert.Equal(t, tt.expectedError, err.Error(), "error message should match")
		})
	}
}

func TestSchemaRegistry_RegisterSchema_DuplicateAnchor_Error(t *testing.T) {
	t.Parallel()

	t.Run("duplicate anchor in document scope", func(t *testing.T) {
		t.Parallel()

		registry := oas3.NewSchemaRegistry("https://example.com/doc.json")

		firstSchema := createTestJSONSchema("", "myAnchor")
		err := registry.RegisterSchema(firstSchema, "")
		require.NoError(t, err, "first registration should succeed")

		secondSchema := createTestJSONSchema("", "myAnchor")
		err = registry.RegisterSchema(secondSchema, "")
		require.Error(t, err, "second registration should fail")
		assert.Equal(t, `duplicate $anchor detected: "myAnchor" in scope "https://example.com/doc.json" is already registered`, err.Error(), "error message should match")
	})

	t.Run("duplicate anchor in $id scope", func(t *testing.T) {
		t.Parallel()

		registry := oas3.NewSchemaRegistry("https://example.com/doc.json")

		// First schema has its own $id and an anchor
		firstSchema := createTestJSONSchema("https://example.com/schemas/user.json", "address")
		err := registry.RegisterSchema(firstSchema, "")
		require.NoError(t, err, "first registration should succeed")

		// Second schema has no $id but inherits the base via parentBaseURI, same anchor
		secondSchema := createTestJSONSchema("", "address")
		err = registry.RegisterSchema(secondSchema, "https://example.com/schemas/user.json")
		require.Error(t, err, "second registration should fail")
		assert.Equal(t, `duplicate $anchor detected: "address" in scope "https://example.com/schemas/user.json" is already registered`, err.Error(), "error message should match")
	})
}

func TestSchemaRegistry_RegisterSchema_SamePointer_Success(t *testing.T) {
	t.Parallel()

	t.Run("re-registering same schema with $id succeeds", func(t *testing.T) {
		t.Parallel()

		registry := oas3.NewSchemaRegistry("https://example.com/doc.json")
		schema := createTestJSONSchema("https://example.com/schemas/user.json", "")

		err := registry.RegisterSchema(schema, "")
		require.NoError(t, err, "first registration should succeed")

		// Re-registering the same schema pointer should be idempotent
		err = registry.RegisterSchema(schema, "")
		require.NoError(t, err, "second registration of same pointer should succeed")
	})

	t.Run("re-registering same schema with $anchor succeeds", func(t *testing.T) {
		t.Parallel()

		registry := oas3.NewSchemaRegistry("https://example.com/doc.json")
		schema := createTestJSONSchema("", "myAnchor")

		err := registry.RegisterSchema(schema, "")
		require.NoError(t, err, "first registration should succeed")

		// Re-registering the same schema pointer should be idempotent
		err = registry.RegisterSchema(schema, "")
		require.NoError(t, err, "second registration of same pointer should succeed")
	})

	t.Run("re-registering same schema with both $id and $anchor succeeds", func(t *testing.T) {
		t.Parallel()

		registry := oas3.NewSchemaRegistry("https://example.com/doc.json")
		schema := createTestJSONSchema("https://example.com/schemas/user.json", "address")

		err := registry.RegisterSchema(schema, "")
		require.NoError(t, err, "first registration should succeed")

		// Re-registering the same schema pointer should be idempotent
		err = registry.RegisterSchema(schema, "")
		require.NoError(t, err, "second registration of same pointer should succeed")
	})
}

func TestIsAbsoluteURI_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		uri        string
		isAbsolute bool
	}{
		{
			name:       "HTTPS URL",
			uri:        "https://example.com/path",
			isAbsolute: true,
		},
		{
			name:       "HTTP URL",
			uri:        "http://example.com/path",
			isAbsolute: true,
		},
		{
			name:       "file URI",
			uri:        "file:///path/to/file",
			isAbsolute: true,
		},
		{
			name:       "URN",
			uri:        "urn:example:animal:ferret:nose",
			isAbsolute: true,
		},
		{
			name:       "relative path",
			uri:        "path/to/schema.json",
			isAbsolute: false,
		},
		{
			name:       "relative with leading slash",
			uri:        "/path/to/schema.json",
			isAbsolute: false,
		},
		{
			name:       "fragment only",
			uri:        "#anchor",
			isAbsolute: false,
		},
		{
			name:       "empty string",
			uri:        "",
			isAbsolute: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := oas3.IsAbsoluteURI(tt.uri)
			assert.Equal(t, tt.isAbsolute, result, "IsAbsoluteURI result should match expected")
		})
	}
}

func TestIsAnchorReference_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ref      string
		isAnchor bool
	}{
		{
			name:     "plain anchor",
			ref:      "#myAnchor",
			isAnchor: true,
		},
		{
			name:     "anchor with underscore",
			ref:      "#my_anchor",
			isAnchor: true,
		},
		{
			name:     "JSON pointer",
			ref:      "#/path/to/schema",
			isAnchor: false,
		},
		{
			name:     "empty fragment",
			ref:      "#",
			isAnchor: false,
		},
		{
			name:     "no fragment",
			ref:      "schema.json",
			isAnchor: false,
		},
		{
			name:     "URL with anchor",
			ref:      "https://example.com/schema.json#anchor",
			isAnchor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := oas3.IsAnchorReference(tt.ref)
			assert.Equal(t, tt.isAnchor, result, "IsAnchorReference result should match expected")
		})
	}
}

func TestExtractAnchor_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "fragment only",
			ref:      "#myAnchor",
			expected: "myAnchor",
		},
		{
			name:     "URL with fragment",
			ref:      "https://example.com/schema.json#anchor",
			expected: "anchor",
		},
		{
			name:     "JSON pointer returns empty",
			ref:      "#/path/to/schema",
			expected: "",
		},
		{
			name:     "no fragment",
			ref:      "schema.json",
			expected: "",
		},
		{
			name:     "empty fragment",
			ref:      "#",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := oas3.ExtractAnchor(tt.ref)
			assert.Equal(t, tt.expected, result, "ExtractAnchor result should match expected")
		})
	}
}

func TestResolveURI_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     string
		ref      string
		expected string
	}{
		{
			name:     "absolute ref unchanged",
			base:     "https://example.com/base/doc.json",
			ref:      "https://other.com/schema.json",
			expected: "https://other.com/schema.json",
		},
		{
			name:     "relative path resolved",
			base:     "https://example.com/schemas/doc.json",
			ref:      "user.json",
			expected: "https://example.com/schemas/user.json",
		},
		{
			name:     "relative path with parent directory",
			base:     "https://example.com/schemas/nested/doc.json",
			ref:      "../user.json",
			expected: "https://example.com/schemas/user.json",
		},
		{
			name:     "absolute path resolved",
			base:     "https://example.com/schemas/doc.json",
			ref:      "/other/schema.json",
			expected: "https://example.com/other/schema.json",
		},
		{
			name:     "empty ref returns base",
			base:     "https://example.com/doc.json",
			ref:      "",
			expected: "https://example.com/doc.json",
		},
		{
			name:     "empty base returns normalized ref",
			base:     "",
			ref:      "user.json",
			expected: "user.json",
		},
		{
			name:     "fragment appended",
			base:     "https://example.com/doc.json",
			ref:      "#anchor",
			expected: "https://example.com/doc.json#anchor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := oas3.ResolveURI(tt.base, tt.ref)
			assert.Equal(t, tt.expected, result, "ResolveURI result should match expected")
		})
	}
}

// Helper function to create a test JSONSchema with optional $id and $anchor
func createTestJSONSchema(id, anchor string) *oas3.JSONSchema[oas3.Referenceable] {
	schema := &oas3.Schema{}

	if id != "" {
		schema.ID = pointer.From(id)
	}

	if anchor != "" {
		schema.Anchor = pointer.From(anchor)
	}

	return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](schema)
}
