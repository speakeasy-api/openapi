package rules

import (
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// ============================================================
// Non-interactive fixes
// ============================================================

func TestAddErrorResponseFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addErrorResponseFix{statusCode: "401", description: "Unauthorized"}
		assert.Equal(t, "Add 401 response: Unauthorized", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
		require.NoError(t, f.SetInput(nil))
		require.NoError(t, f.Apply(nil))
	})

	t.Run("adds response to mapping", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		responsesNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("200"),
			yml.CreateMapNode(ctx, []*yaml.Node{
				yml.CreateStringNode("description"),
				yml.CreateStringNode("OK"),
			}),
		})
		f := &addErrorResponseFix{responsesNode: responsesNode, statusCode: "401", description: "Unauthorized"}
		require.NoError(t, f.ApplyNode(nil))

		_, val, found := yml.GetMapElementNodes(ctx, responsesNode, "401")
		require.True(t, found, "401 response should be added")
		_, desc, found := yml.GetMapElementNodes(ctx, val, "description")
		require.True(t, found, "response should have description")
		assert.Equal(t, "Unauthorized", desc.Value)
	})

	t.Run("idempotent when status exists", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		responsesNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("401"),
			yml.CreateMapNode(ctx, []*yaml.Node{
				yml.CreateStringNode("description"),
				yml.CreateStringNode("Existing"),
			}),
		})
		f := &addErrorResponseFix{responsesNode: responsesNode, statusCode: "401", description: "Unauthorized"}
		require.NoError(t, f.ApplyNode(nil))

		_, val, _ := yml.GetMapElementNodes(ctx, responsesNode, "401")
		_, desc, _ := yml.GetMapElementNodes(ctx, val, "description")
		assert.Equal(t, "Existing", desc.Value, "should not overwrite existing response")
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &addErrorResponseFix{responsesNode: nil}
		require.NoError(t, f.ApplyNode(nil))
	})

	t.Run("non-mapping node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &addErrorResponseFix{responsesNode: &yaml.Node{Kind: yaml.ScalarNode}}
		require.NoError(t, f.ApplyNode(nil))
	})
}

func TestAddRetryAfterHeaderFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addRetryAfterHeaderFix{}
		assert.Equal(t, "Add Retry-After header to 429 response", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
		require.NoError(t, f.SetInput(nil))
		require.NoError(t, f.Apply(nil))
	})

	t.Run("creates headers and adds Retry-After", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		responseNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("description"),
			yml.CreateStringNode("Too Many Requests"),
		})
		f := &addRetryAfterHeaderFix{responseNode: responseNode}
		require.NoError(t, f.ApplyNode(nil))

		_, headersNode, found := yml.GetMapElementNodes(ctx, responseNode, "headers")
		require.True(t, found, "headers should be added")
		_, retryAfter, found := yml.GetMapElementNodes(ctx, headersNode, "Retry-After")
		require.True(t, found, "Retry-After header should be added")
		_, desc, found := yml.GetMapElementNodes(ctx, retryAfter, "description")
		require.True(t, found)
		assert.Contains(t, desc.Value, "seconds")
		_, schema, found := yml.GetMapElementNodes(ctx, retryAfter, "schema")
		require.True(t, found)
		_, typ, found := yml.GetMapElementNodes(ctx, schema, "type")
		require.True(t, found)
		assert.Equal(t, "integer", typ.Value)
	})

	t.Run("adds to existing headers mapping", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		headersNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("X-Custom"),
			yml.CreateMapNode(ctx, []*yaml.Node{
				yml.CreateStringNode("description"),
				yml.CreateStringNode("Custom header"),
			}),
		})
		responseNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("headers"),
			headersNode,
		})
		f := &addRetryAfterHeaderFix{responseNode: responseNode}
		require.NoError(t, f.ApplyNode(nil))

		_, hNode, _ := yml.GetMapElementNodes(ctx, responseNode, "headers")
		_, _, found := yml.GetMapElementNodes(ctx, hNode, "Retry-After")
		assert.True(t, found, "Retry-After should be added alongside existing headers")
		_, _, found = yml.GetMapElementNodes(ctx, hNode, "X-Custom")
		assert.True(t, found, "existing headers should be preserved")
	})

	t.Run("idempotent when Retry-After exists", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		retryAfterHeader := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("description"),
			yml.CreateStringNode("Original"),
		})
		headersNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("Retry-After"),
			retryAfterHeader,
		})
		responseNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("headers"),
			headersNode,
		})
		f := &addRetryAfterHeaderFix{responseNode: responseNode}
		require.NoError(t, f.ApplyNode(nil))

		_, hNode, _ := yml.GetMapElementNodes(ctx, responseNode, "headers")
		_, raNode, _ := yml.GetMapElementNodes(ctx, hNode, "Retry-After")
		_, dNode, _ := yml.GetMapElementNodes(ctx, raNode, "description")
		assert.Equal(t, "Original", dNode.Value, "should not overwrite existing Retry-After")
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &addRetryAfterHeaderFix{responseNode: nil}
		require.NoError(t, f.ApplyNode(nil))
	})
}

// ============================================================
// Interactive single-prompt fixes
// ============================================================

func TestAddDescriptionFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addDescriptionFix{targetLabel: "schema 'Pet'"}
		assert.Equal(t, "Add description to schema 'Pet'", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
		assert.Contains(t, prompts[0].Message, "schema 'Pet'")
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &addDescriptionFix{}
		require.NoError(t, f.SetInput([]string{"A pet object"}))
		assert.Equal(t, "A pet object", f.description)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &addDescriptionFix{}
		require.Error(t, f.SetInput([]string{}))
		require.Error(t, f.SetInput([]string{"a", "b"}))
	})

	t.Run("applies description", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		targetNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			yml.CreateStringNode("object"),
		})
		f := &addDescriptionFix{targetNode: targetNode, description: "A pet object"}
		require.NoError(t, f.ApplyNode(nil))

		_, desc, found := yml.GetMapElementNodes(ctx, targetNode, "description")
		require.True(t, found)
		assert.Equal(t, "A pet object", desc.Value)
	})

	t.Run("empty description is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		targetNode := yml.CreateMapNode(ctx, nil)
		f := &addDescriptionFix{targetNode: targetNode, description: ""}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, targetNode, "description")
		assert.False(t, found)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &addDescriptionFix{targetNode: nil, description: "test"}
		require.NoError(t, f.ApplyNode(nil))
	})

	t.Run("apply is no-op", func(t *testing.T) {
		t.Parallel()
		f := &addDescriptionFix{}
		require.NoError(t, f.Apply(nil))
	})
}

func TestAddLicenseURLFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addLicenseURLFix{}
		assert.Equal(t, "Add URL to license", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &addLicenseURLFix{}
		require.NoError(t, f.SetInput([]string{"https://opensource.org/licenses/MIT"}))
		assert.Equal(t, "https://opensource.org/licenses/MIT", f.url)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &addLicenseURLFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("applies url to license node", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		licenseNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("name"),
			yml.CreateStringNode("MIT"),
		})
		f := &addLicenseURLFix{licenseNode: licenseNode, url: "https://opensource.org/licenses/MIT"}
		require.NoError(t, f.ApplyNode(nil))

		_, urlNode, found := yml.GetMapElementNodes(ctx, licenseNode, "url")
		require.True(t, found)
		assert.Equal(t, "https://opensource.org/licenses/MIT", urlNode.Value)
	})

	t.Run("empty url is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		licenseNode := yml.CreateMapNode(ctx, nil)
		f := &addLicenseURLFix{licenseNode: licenseNode, url: ""}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, licenseNode, "url")
		assert.False(t, found)
	})
}

func TestAddContactPropertyFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addContactPropertyFix{property: "email"}
		assert.Equal(t, "Add email to contact", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
		assert.Contains(t, prompts[0].Message, "email")
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &addContactPropertyFix{}
		require.NoError(t, f.SetInput([]string{"test@example.com"}))
		assert.Equal(t, "test@example.com", f.value)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &addContactPropertyFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("applies property to contact node", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		contactNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("name"),
			yml.CreateStringNode("Support"),
		})
		f := &addContactPropertyFix{contactNode: contactNode, property: "email", value: "support@example.com"}
		require.NoError(t, f.ApplyNode(nil))

		_, emailNode, found := yml.GetMapElementNodes(ctx, contactNode, "email")
		require.True(t, found)
		assert.Equal(t, "support@example.com", emailNode.Value)
	})

	t.Run("empty value is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		contactNode := yml.CreateMapNode(ctx, nil)
		f := &addContactPropertyFix{contactNode: contactNode, property: "email", value: ""}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, contactNode, "email")
		assert.False(t, found)
	})
}

func TestReplaceServerURLFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &replaceServerURLFix{}
		assert.Equal(t, "Replace server URL", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &replaceServerURLFix{}
		require.NoError(t, f.SetInput([]string{"https://api.real.com"}))
		assert.Equal(t, "https://api.real.com", f.newURL)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &replaceServerURLFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("replaces node value", func(t *testing.T) {
		t.Parallel()
		urlNode := yml.CreateStringNode("https://example.com")
		f := &replaceServerURLFix{urlNode: urlNode, newURL: "https://api.real.com"}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "https://api.real.com", urlNode.Value)
	})

	t.Run("empty url is no-op", func(t *testing.T) {
		t.Parallel()
		urlNode := yml.CreateStringNode("https://example.com")
		f := &replaceServerURLFix{urlNode: urlNode, newURL: ""}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "https://example.com", urlNode.Value)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &replaceServerURLFix{urlNode: nil, newURL: "https://api.real.com"}
		require.NoError(t, f.ApplyNode(nil))
	})
}

func TestAddOperationTagFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addOperationTagFix{}
		assert.Equal(t, "Add tag to operation", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &addOperationTagFix{}
		require.NoError(t, f.SetInput([]string{"users"}))
		assert.Equal(t, "users", f.tag)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &addOperationTagFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("creates tags array and adds tag", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		operationNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("summary"),
			yml.CreateStringNode("List users"),
		})
		f := &addOperationTagFix{operationNode: operationNode, tag: "users"}
		require.NoError(t, f.ApplyNode(nil))

		_, tagsNode, found := yml.GetMapElementNodes(ctx, operationNode, "tags")
		require.True(t, found, "tags should be created")
		assert.Equal(t, yaml.SequenceNode, tagsNode.Kind)
		require.Len(t, tagsNode.Content, 1)
		assert.Equal(t, "users", tagsNode.Content[0].Value)
	})

	t.Run("appends to existing tags", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		tagsNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
			Content: []*yaml.Node{
				yml.CreateStringNode("existing"),
			},
		}
		operationNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("tags"),
			tagsNode,
		})
		f := &addOperationTagFix{operationNode: operationNode, tag: "newTag"}
		require.NoError(t, f.ApplyNode(nil))

		_, updatedTags, _ := yml.GetMapElementNodes(ctx, operationNode, "tags")
		require.Len(t, updatedTags.Content, 2)
		assert.Equal(t, "existing", updatedTags.Content[0].Value)
		assert.Equal(t, "newTag", updatedTags.Content[1].Value)
	})

	t.Run("empty tag is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		operationNode := yml.CreateMapNode(ctx, nil)
		f := &addOperationTagFix{operationNode: operationNode, tag: ""}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, operationNode, "tags")
		assert.False(t, found)
	})
}

// ============================================================
// Interactive choice fixes
// ============================================================

func TestAddLicenseFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addLicenseFix{}
		assert.Equal(t, "Add license to info section", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptChoice, prompts[0].Type)
		assert.Contains(t, prompts[0].Choices, "MIT")
		assert.Contains(t, prompts[0].Choices, "Apache-2.0")
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &addLicenseFix{}
		require.NoError(t, f.SetInput([]string{"MIT"}))
		assert.Equal(t, "MIT", f.licenseName)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &addLicenseFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("adds license to info node", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		infoNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("title"),
			yml.CreateStringNode("Test API"),
		})
		f := &addLicenseFix{infoNode: infoNode, licenseName: "MIT"}
		require.NoError(t, f.ApplyNode(nil))

		_, licenseNode, found := yml.GetMapElementNodes(ctx, infoNode, "license")
		require.True(t, found)
		_, nameNode, found := yml.GetMapElementNodes(ctx, licenseNode, "name")
		require.True(t, found)
		assert.Equal(t, "MIT", nameNode.Value)
	})

	t.Run("empty license is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		infoNode := yml.CreateMapNode(ctx, nil)
		f := &addLicenseFix{infoNode: infoNode, licenseName: ""}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, infoNode, "license")
		assert.False(t, found)
	})
}

func TestSetIntegerFormatFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerFormatFix{}
		assert.Equal(t, "Set integer format", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptChoice, prompts[0].Type)
		assert.Contains(t, prompts[0].Choices, "int32")
		assert.Contains(t, prompts[0].Choices, "int64")
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerFormatFix{}
		require.NoError(t, f.SetInput([]string{"int64"}))
		assert.Equal(t, "int64", f.format)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerFormatFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("sets format on schema", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			yml.CreateStringNode("integer"),
		})
		f := &setIntegerFormatFix{schemaNode: schemaNode, format: "int32"}
		require.NoError(t, f.ApplyNode(nil))

		_, formatNode, found := yml.GetMapElementNodes(ctx, schemaNode, "format")
		require.True(t, found)
		assert.Equal(t, "int32", formatNode.Value)
	})

	t.Run("empty format is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		schemaNode := yml.CreateMapNode(ctx, nil)
		f := &setIntegerFormatFix{schemaNode: schemaNode, format: ""}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, schemaNode, "format")
		assert.False(t, found)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerFormatFix{schemaNode: nil, format: "int32"}
		require.NoError(t, f.ApplyNode(nil))
	})
}

// ============================================================
// Interactive multi-prompt fixes
// ============================================================

func TestAddContactFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addContactFix{}
		assert.Equal(t, "Add contact information to info section", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 3)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
		assert.Equal(t, validation.PromptFreeText, prompts[1].Type)
		assert.Equal(t, validation.PromptFreeText, prompts[2].Type)
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &addContactFix{}
		require.NoError(t, f.SetInput([]string{"Support", "https://support.example.com", "support@example.com"}))
		assert.Equal(t, "Support", f.name)
		assert.Equal(t, "https://support.example.com", f.url)
		assert.Equal(t, "support@example.com", f.email)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &addContactFix{}
		require.Error(t, f.SetInput([]string{"only one"}))
		require.Error(t, f.SetInput([]string{"a", "b"}))
	})

	t.Run("adds contact with all fields", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		infoNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("title"),
			yml.CreateStringNode("Test API"),
		})
		f := &addContactFix{infoNode: infoNode, name: "Support", url: "https://support.example.com", email: "support@example.com"}
		require.NoError(t, f.ApplyNode(nil))

		_, contactNode, found := yml.GetMapElementNodes(ctx, infoNode, "contact")
		require.True(t, found)
		_, nameNode, found := yml.GetMapElementNodes(ctx, contactNode, "name")
		require.True(t, found)
		assert.Equal(t, "Support", nameNode.Value)
		_, urlNode, found := yml.GetMapElementNodes(ctx, contactNode, "url")
		require.True(t, found)
		assert.Equal(t, "https://support.example.com", urlNode.Value)
		_, emailNode, found := yml.GetMapElementNodes(ctx, contactNode, "email")
		require.True(t, found)
		assert.Equal(t, "support@example.com", emailNode.Value)
	})

	t.Run("adds contact with partial fields", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		infoNode := yml.CreateMapNode(ctx, nil)
		f := &addContactFix{infoNode: infoNode, name: "Support", url: "", email: ""}
		require.NoError(t, f.ApplyNode(nil))

		_, contactNode, found := yml.GetMapElementNodes(ctx, infoNode, "contact")
		require.True(t, found, "contact should be added even with partial fields")
		_, nameNode, found := yml.GetMapElementNodes(ctx, contactNode, "name")
		require.True(t, found)
		assert.Equal(t, "Support", nameNode.Value)
		_, _, found = yml.GetMapElementNodes(ctx, contactNode, "url")
		assert.False(t, found, "empty url should not be added")
		_, _, found = yml.GetMapElementNodes(ctx, contactNode, "email")
		assert.False(t, found, "empty email should not be added")
	})

	t.Run("all empty fields is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		infoNode := yml.CreateMapNode(ctx, nil)
		f := &addContactFix{infoNode: infoNode, name: "", url: "", email: ""}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, infoNode, "contact")
		assert.False(t, found)
	})
}

func TestSetIntegerLimitsFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerLimitsFix{}
		assert.Equal(t, "Set integer minimum and maximum", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 2)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
		assert.Contains(t, prompts[0].Message, "Minimum")
		assert.Equal(t, validation.PromptFreeText, prompts[1].Type)
		assert.Contains(t, prompts[1].Message, "Maximum")
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerLimitsFix{}
		require.NoError(t, f.SetInput([]string{"0", "100"}))
		assert.Equal(t, int64(0), f.minVal)
		assert.Equal(t, int64(100), f.maxVal)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerLimitsFix{}
		require.Error(t, f.SetInput([]string{"1"}))
		require.Error(t, f.SetInput([]string{"1", "2", "3"}))
	})

	t.Run("set input invalid minimum", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerLimitsFix{}
		err := f.SetInput([]string{"abc", "100"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid minimum")
	})

	t.Run("set input invalid maximum", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerLimitsFix{}
		err := f.SetInput([]string{"0", "xyz"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid maximum")
	})

	t.Run("sets minimum and maximum on schema", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			yml.CreateStringNode("integer"),
		})
		f := &setIntegerLimitsFix{schemaNode: schemaNode, minVal: -100, maxVal: 100}
		require.NoError(t, f.ApplyNode(nil))

		_, minNode, found := yml.GetMapElementNodes(ctx, schemaNode, "minimum")
		require.True(t, found)
		assert.Equal(t, "-100", minNode.Value)
		_, maxNode, found := yml.GetMapElementNodes(ctx, schemaNode, "maximum")
		require.True(t, found)
		assert.Equal(t, "100", maxNode.Value)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &setIntegerLimitsFix{schemaNode: nil}
		require.NoError(t, f.ApplyNode(nil))
	})
}

// ============================================================
// Interactive numeric fix
// ============================================================

func TestSetNumericPropertyFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &setNumericPropertyFix{property: "maxLength", label: "Maximum string length"}
		assert.Equal(t, "Set maxLength", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
		assert.Equal(t, "Maximum string length", prompts[0].Message)
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &setNumericPropertyFix{}
		require.NoError(t, f.SetInput([]string{"255"}))
		assert.Equal(t, int64(255), f.value)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &setNumericPropertyFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("set input invalid number", func(t *testing.T) {
		t.Parallel()
		f := &setNumericPropertyFix{}
		err := f.SetInput([]string{"not-a-number"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid number")
	})

	t.Run("sets property on schema", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			yml.CreateStringNode("string"),
		})
		f := &setNumericPropertyFix{schemaNode: schemaNode, property: "maxLength", value: 255}
		require.NoError(t, f.ApplyNode(nil))

		_, valNode, found := yml.GetMapElementNodes(ctx, schemaNode, "maxLength")
		require.True(t, found)
		assert.Equal(t, "255", valNode.Value)
	})

	t.Run("works for maxItems", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			yml.CreateStringNode("array"),
		})
		f := &setNumericPropertyFix{schemaNode: schemaNode, property: "maxItems", value: 100}
		require.NoError(t, f.ApplyNode(nil))

		_, valNode, found := yml.GetMapElementNodes(ctx, schemaNode, "maxItems")
		require.True(t, found)
		assert.Equal(t, "100", valNode.Value)
	})

	t.Run("works for maxProperties", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			yml.CreateStringNode("object"),
		})
		f := &setNumericPropertyFix{schemaNode: schemaNode, property: "maxProperties", value: 50}
		require.NoError(t, f.ApplyNode(nil))

		_, valNode, found := yml.GetMapElementNodes(ctx, schemaNode, "maxProperties")
		require.True(t, found)
		assert.Equal(t, "50", valNode.Value)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &setNumericPropertyFix{schemaNode: nil, property: "maxLength", value: 100}
		require.NoError(t, f.ApplyNode(nil))
	})
}

// ============================================================
// Interactive confirmation fix
// ============================================================

func TestRemoveUnusedComponentFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &removeUnusedComponentFix{componentRef: "#/components/schemas/Pet"}
		assert.Equal(t, "Remove unused component #/components/schemas/Pet", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptChoice, prompts[0].Type)
		assert.Contains(t, prompts[0].Choices, "Yes")
		assert.Contains(t, prompts[0].Choices, "No")
		assert.Contains(t, prompts[0].Message, "#/components/schemas/Pet")
	})

	t.Run("set input yes", func(t *testing.T) {
		t.Parallel()
		f := &removeUnusedComponentFix{}
		require.NoError(t, f.SetInput([]string{"Yes"}))
		assert.True(t, f.confirmed)
	})

	t.Run("set input no", func(t *testing.T) {
		t.Parallel()
		f := &removeUnusedComponentFix{}
		require.NoError(t, f.SetInput([]string{"No"}))
		assert.False(t, f.confirmed)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &removeUnusedComponentFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("confirmed removes component from mapping", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		parentMap := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("Pet"),
			yml.CreateMapNode(ctx, []*yaml.Node{
				yml.CreateStringNode("type"),
				yml.CreateStringNode("object"),
			}),
			yml.CreateStringNode("User"),
			yml.CreateMapNode(ctx, []*yaml.Node{
				yml.CreateStringNode("type"),
				yml.CreateStringNode("object"),
			}),
		})
		f := &removeUnusedComponentFix{parentMapNode: parentMap, componentName: "Pet", confirmed: true}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, parentMap, "Pet")
		assert.False(t, found, "Pet should be removed")
		_, _, found = yml.GetMapElementNodes(ctx, parentMap, "User")
		assert.True(t, found, "User should be preserved")
	})

	t.Run("not confirmed is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		parentMap := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("Pet"),
			yml.CreateMapNode(ctx, nil),
		})
		f := &removeUnusedComponentFix{parentMapNode: parentMap, componentName: "Pet", confirmed: false}
		require.NoError(t, f.ApplyNode(nil))

		_, _, found := yml.GetMapElementNodes(ctx, parentMap, "Pet")
		assert.True(t, found, "Pet should NOT be removed when not confirmed")
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &removeUnusedComponentFix{parentMapNode: nil, componentName: "Pet", confirmed: true}
		require.NoError(t, f.ApplyNode(nil))
	})

	t.Run("apply is no-op", func(t *testing.T) {
		t.Parallel()
		f := &removeUnusedComponentFix{}
		require.NoError(t, f.Apply(nil))
	})
}

// ============================================================
// Model fix (Apply instead of ApplyNode)
// ============================================================

func TestAddServerFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addServerFix{}
		assert.Equal(t, "Add server URL", f.Description())
		assert.True(t, f.Interactive())
		prompts := f.Prompts()
		require.Len(t, prompts, 1)
		assert.Equal(t, validation.PromptFreeText, prompts[0].Type)
	})

	t.Run("set input success", func(t *testing.T) {
		t.Parallel()
		f := &addServerFix{}
		require.NoError(t, f.SetInput([]string{"https://api.example.com"}))
		assert.Equal(t, "https://api.example.com", f.url)
	})

	t.Run("set input wrong count", func(t *testing.T) {
		t.Parallel()
		f := &addServerFix{}
		require.Error(t, f.SetInput([]string{}))
	})

	t.Run("adds server to document", func(t *testing.T) {
		t.Parallel()
		doc := &openapi.OpenAPI{}
		f := &addServerFix{url: "https://api.example.com"}
		require.NoError(t, f.Apply(doc))

		require.Len(t, doc.Servers, 1)
		assert.Equal(t, "https://api.example.com", doc.Servers[0].URL)
	})

	t.Run("appends to existing servers", func(t *testing.T) {
		t.Parallel()
		doc := &openapi.OpenAPI{
			Servers: []*openapi.Server{{URL: "https://existing.com"}},
		}
		f := &addServerFix{url: "https://new.example.com"}
		require.NoError(t, f.Apply(doc))

		require.Len(t, doc.Servers, 2)
		assert.Equal(t, "https://existing.com", doc.Servers[0].URL)
		assert.Equal(t, "https://new.example.com", doc.Servers[1].URL)
	})

	t.Run("empty url is no-op", func(t *testing.T) {
		t.Parallel()
		doc := &openapi.OpenAPI{}
		f := &addServerFix{url: ""}
		require.NoError(t, f.Apply(doc))

		assert.Empty(t, doc.Servers)
	})

	t.Run("wrong doc type returns error", func(t *testing.T) {
		t.Parallel()
		f := &addServerFix{url: "https://api.example.com"}
		err := f.Apply("not an openapi doc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected *openapi.OpenAPI")
	})
}
