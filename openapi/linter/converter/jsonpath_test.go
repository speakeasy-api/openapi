package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapJSONPath_IndexCollections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		path        string
		collection  string
		isKeyAccess bool
		fieldAccess string
		httpMethod  string
		isDirect    bool
	}{
		{
			name:       "paths wildcard maps to inlinePathItems",
			path:       "$.paths[*]",
			collection: "inlinePathItems",
		},
		{
			name:        "paths with tilde maps to inlinePathItems with key access",
			path:        "$.paths[*]~",
			collection:  "inlinePathItems",
			isKeyAccess: true,
		},
		{
			name:       "paths double wildcard maps to operations",
			path:       "$.paths[*][*]",
			collection: "operations",
		},
		{
			name:       "paths GET method maps to operations with method filter",
			path:       "$.paths[*].get",
			collection: "operations",
			httpMethod: "get",
		},
		{
			name:       "paths POST method maps to operations with method filter",
			path:       "$.paths[*].post",
			collection: "operations",
			httpMethod: "post",
		},
		{
			name:       "paths operation responses maps to inlineResponses",
			path:       "$.paths[*][*].responses[*]",
			collection: "inlineResponses",
		},
		{
			name:       "paths operation requestBody maps to inlineRequestBodies",
			path:       "$.paths[*][*].requestBody",
			collection: "inlineRequestBodies",
		},
		{
			name:       "recursive parameters maps to inlineParameters",
			path:       "$..parameters[*]",
			collection: "inlineParameters",
		},
		{
			name:       "servers wildcard maps to servers",
			path:       "$.servers[*]",
			collection: "servers",
		},
		{
			name:        "servers url field maps to servers with field access",
			path:        "$.servers[*].url",
			collection:  "servers",
			fieldAccess: "url",
		},
		{
			name:       "tags wildcard maps to tags",
			path:       "$.tags[*]",
			collection: "tags",
		},
		{
			name:        "tags name field maps to tags with field access",
			path:        "$.tags[*].name",
			collection:  "tags",
			fieldAccess: "name",
		},
		{
			name:       "component schemas maps to componentSchemas",
			path:       "$.components.schemas[*]",
			collection: "componentSchemas",
		},
		{
			name:        "component schemas with tilde maps to componentSchemas with key access",
			path:        "$.components.schemas[*]~",
			collection:  "componentSchemas",
			isKeyAccess: true,
		},
		{
			name:       "component responses maps to componentResponses",
			path:       "$.components.responses[*]",
			collection: "componentResponses",
		},
		{
			name:       "component parameters maps to componentParameters",
			path:       "$.components.parameters[*]",
			collection: "componentParameters",
		},
		{
			name:       "component security schemes maps to componentSecuritySchemes",
			path:       "$.components.securitySchemes[*]",
			collection: "componentSecuritySchemes",
		},
		{
			name:       "component examples maps to componentExamples",
			path:       "$.components.examples[*]",
			collection: "componentExamples",
		},
		{
			name:       "OAS2 definitions maps to componentSchemas",
			path:       "$.definitions[*]",
			collection: "componentSchemas",
		},
		{
			name:       "recursive properties maps to inlineSchemas",
			path:       "$..properties[*]",
			collection: "inlineSchemas",
		},
		{
			name:       "description nodes maps to descriptionNodes",
			path:       "$..description",
			collection: "descriptionNodes",
		},
		{
			name:       "summary nodes maps to summaryNodes",
			path:       "$..summary",
			collection: "summaryNodes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := MapJSONPath(tt.path)
			assert.False(t, m.Unsupported, "should not be unsupported")
			assert.Equal(t, tt.collection, m.Collection, "collection")
			assert.Equal(t, tt.isKeyAccess, m.IsKeyAccess, "isKeyAccess")
			assert.Equal(t, tt.fieldAccess, m.FieldAccess, "fieldAccess")
			assert.Equal(t, tt.httpMethod, m.HTTPMethod, "httpMethod")
			assert.Equal(t, tt.isDirect, m.IsDirect, "isDirect")
		})
	}
}

func TestMapJSONPath_DirectAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		directAccess string
		fieldAccess  string
	}{
		{
			name:         "root document",
			path:         "$",
			directAccess: "docInfo.document",
		},
		{
			name:         "info object",
			path:         "$.info",
			directAccess: "docInfo.document.getInfo()",
		},
		{
			name:         "info version field",
			path:         "$.info.version",
			directAccess: "docInfo.document.getInfo()",
			fieldAccess:  "version",
		},
		{
			name:         "info contact",
			path:         "$.info.contact",
			directAccess: "docInfo.document.getInfo()?.getContact()",
		},
		{
			name:         "info license",
			path:         "$.info.license",
			directAccess: "docInfo.document.getInfo()?.getLicense()",
		},
		{
			name:         "components object",
			path:         "$.components",
			directAccess: "docInfo.document.getComponents()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := MapJSONPath(tt.path)
			assert.False(t, m.Unsupported, "should not be unsupported")
			assert.True(t, m.IsDirect, "should be direct access")
			assert.Equal(t, tt.directAccess, m.DirectAccess, "direct access expression")
			assert.Equal(t, tt.fieldAccess, m.FieldAccess, "field access")
		})
	}
}

func TestMapJSONPath_FilterStripping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		collection string
		hasFilter  bool
	}{
		{
			name:       "parameter filter stripped and matched",
			path:       "$..parameters[?(@.in == 'query')]",
			collection: "inlineParameters",
			hasFilter:  true,
		},
		{
			name:       "paths with filter stripped and matched",
			path:       "$.paths.*[?(@['x-speakeasy-ignore'] != true)]",
			collection: "operations",
			hasFilter:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := MapJSONPath(tt.path)
			assert.False(t, m.Unsupported, "should not be unsupported")
			assert.Equal(t, tt.collection, m.Collection, "collection after filter stripping")
			if tt.hasFilter {
				assert.NotEmpty(t, m.Filter, "should have captured filter")
			}
		})
	}
}

func TestMapJSONPath_Unsupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "deeply nested custom path",
			path: "$.paths[*][*].responses[*].content[*].schema.properties[*]",
		},
		{
			name: "completely unknown structure",
			path: "$.x-custom.nested.deeply[*].something",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := MapJSONPath(tt.path)
			assert.True(t, m.Unsupported, "should be unsupported")
			assert.Equal(t, tt.path, m.OriginalPath, "should preserve original path")
		})
	}
}

func TestStripFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           string
		expectedCleaned string
		expectedFilter  string
	}{
		{
			name:            "no filter",
			input:           "$.paths[*]",
			expectedCleaned: "$.paths[*]",
			expectedFilter:  "",
		},
		{
			name:            "simple filter",
			input:           "$..parameters[?(@.in == 'query')]",
			expectedCleaned: "$..parameters[*]",
			expectedFilter:  "[?(@.in == 'query')]",
		},
		{
			name:            "property access filter",
			input:           "$.paths.*[?(@['x-speakeasy-ignore'] != true)]",
			expectedCleaned: "$.paths.*[*]",
			expectedFilter:  "[?(@['x-speakeasy-ignore'] != true)]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cleaned, filter := stripFilters(tt.input)
			assert.Equal(t, tt.expectedCleaned, cleaned, "cleaned path")
			assert.Equal(t, tt.expectedFilter, filter, "extracted filter")
		})
	}
}
