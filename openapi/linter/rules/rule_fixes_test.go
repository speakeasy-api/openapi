package rules

import (
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// ============================================================
// Tests for fix structs defined in individual rule files
// ============================================================

func TestRemoveHostTrailingSlashFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &removeHostTrailingSlashFix{}
		assert.Equal(t, "Remove trailing slash from server URL", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
		require.NoError(t, f.SetInput(nil))
		require.NoError(t, f.Apply(nil))
	})

	t.Run("removes trailing slash", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("https://api.example.com/")
		f := &removeHostTrailingSlashFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "https://api.example.com", node.Value)
	})

	t.Run("removes multiple trailing slashes", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("https://api.example.com///")
		f := &removeHostTrailingSlashFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "https://api.example.com", node.Value)
	})

	t.Run("no trailing slash is no-op", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("https://api.example.com")
		f := &removeHostTrailingSlashFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "https://api.example.com", node.Value)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &removeHostTrailingSlashFix{node: nil}
		require.NoError(t, f.ApplyNode(nil))
	})

	t.Run("describe change", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("https://api.example.com/")
		f := &removeHostTrailingSlashFix{node: node}
		before, after := f.DescribeChange()
		assert.Equal(t, "https://api.example.com/", before, "before should be original value")
		assert.Equal(t, "https://api.example.com", after, "after should have slash removed")
	})

	t.Run("describe change nil node", func(t *testing.T) {
		t.Parallel()
		f := &removeHostTrailingSlashFix{node: nil}
		before, after := f.DescribeChange()
		assert.Empty(t, before, "before should be empty for nil node")
		assert.Empty(t, after, "after should be empty for nil node")
	})
}

func TestRemoveTrailingSlashFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &removeTrailingSlashFix{}
		assert.Equal(t, "Remove trailing slash from path", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
	})

	t.Run("removes trailing slash from path", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("/pets/")
		f := &removeTrailingSlashFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "/pets", node.Value)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &removeTrailingSlashFix{node: nil}
		require.NoError(t, f.ApplyNode(nil))
	})

	t.Run("describe change", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("/pets/")
		f := &removeTrailingSlashFix{node: node}
		before, after := f.DescribeChange()
		assert.Equal(t, "/pets/", before, "before should be original value")
		assert.Equal(t, "/pets", after, "after should have slash removed")
	})

	t.Run("describe change nil node", func(t *testing.T) {
		t.Parallel()
		f := &removeTrailingSlashFix{node: nil}
		before, after := f.DescribeChange()
		assert.Empty(t, before, "before should be empty for nil node")
		assert.Empty(t, after, "after should be empty for nil node")
	})
}

func TestRemoveDuplicateEnumFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &removeDuplicateEnumFix{}
		assert.Equal(t, "Remove duplicate enum entries", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
	})

	t.Run("removes single duplicate", func(t *testing.T) {
		t.Parallel()
		enumNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
			Content: []*yaml.Node{
				yml.CreateStringNode("active"),
				yml.CreateStringNode("inactive"),
				yml.CreateStringNode("active"), // duplicate at index 2
			},
		}
		f := &removeDuplicateEnumFix{enumNode: enumNode, duplicateIndices: []int{2}}
		require.NoError(t, f.ApplyNode(nil))

		require.Len(t, enumNode.Content, 2)
		assert.Equal(t, "active", enumNode.Content[0].Value)
		assert.Equal(t, "inactive", enumNode.Content[1].Value)
	})

	t.Run("removes multiple duplicates", func(t *testing.T) {
		t.Parallel()
		enumNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
			Content: []*yaml.Node{
				yml.CreateStringNode("a"),
				yml.CreateStringNode("b"),
				yml.CreateStringNode("a"), // duplicate at index 2
				yml.CreateStringNode("b"), // duplicate at index 3
			},
		}
		f := &removeDuplicateEnumFix{enumNode: enumNode, duplicateIndices: []int{2, 3}}
		require.NoError(t, f.ApplyNode(nil))

		require.Len(t, enumNode.Content, 2)
		assert.Equal(t, "a", enumNode.Content[0].Value)
		assert.Equal(t, "b", enumNode.Content[1].Value)
	})

	t.Run("nil enum node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &removeDuplicateEnumFix{enumNode: nil, duplicateIndices: []int{0}}
		require.NoError(t, f.ApplyNode(nil))
	})

	t.Run("empty indices is no-op", func(t *testing.T) {
		t.Parallel()
		enumNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				yml.CreateStringNode("a"),
			},
		}
		f := &removeDuplicateEnumFix{enumNode: enumNode, duplicateIndices: nil}
		require.NoError(t, f.ApplyNode(nil))

		require.Len(t, enumNode.Content, 1)
	})
}

func TestSortTagsFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &sortTagsFix{}
		assert.Equal(t, "Sort tags alphabetically", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
	})

	t.Run("sorts tags alphabetically", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		tagsNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
			Content: []*yaml.Node{
				yml.CreateMapNode(ctx, []*yaml.Node{
					yml.CreateStringNode("name"),
					yml.CreateStringNode("users"),
				}),
				yml.CreateMapNode(ctx, []*yaml.Node{
					yml.CreateStringNode("name"),
					yml.CreateStringNode("admin"),
				}),
				yml.CreateMapNode(ctx, []*yaml.Node{
					yml.CreateStringNode("name"),
					yml.CreateStringNode("pets"),
				}),
			},
		}
		f := &sortTagsFix{tagsNode: tagsNode}
		require.NoError(t, f.ApplyNode(nil))

		require.Len(t, tagsNode.Content, 3)
		assert.Equal(t, "admin", getTagName(tagsNode.Content[0]))
		assert.Equal(t, "pets", getTagName(tagsNode.Content[1]))
		assert.Equal(t, "users", getTagName(tagsNode.Content[2]))
	})

	t.Run("case insensitive sort", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		tagsNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
			Content: []*yaml.Node{
				yml.CreateMapNode(ctx, []*yaml.Node{
					yml.CreateStringNode("name"),
					yml.CreateStringNode("Zebra"),
				}),
				yml.CreateMapNode(ctx, []*yaml.Node{
					yml.CreateStringNode("name"),
					yml.CreateStringNode("apple"),
				}),
			},
		}
		f := &sortTagsFix{tagsNode: tagsNode}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "apple", getTagName(tagsNode.Content[0]))
		assert.Equal(t, "Zebra", getTagName(tagsNode.Content[1]))
	})

	t.Run("single tag is no-op", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		tagsNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				yml.CreateMapNode(ctx, []*yaml.Node{
					yml.CreateStringNode("name"),
					yml.CreateStringNode("only"),
				}),
			},
		}
		f := &sortTagsFix{tagsNode: tagsNode}
		require.NoError(t, f.ApplyNode(nil))

		require.Len(t, tagsNode.Content, 1)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &sortTagsFix{tagsNode: nil}
		require.NoError(t, f.ApplyNode(nil))
	})
}

func TestAddGlobalTagFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addGlobalTagFix{tagName: "users"}
		assert.Equal(t, "Add tag `users` to global tags", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
		require.NoError(t, f.SetInput(nil))
	})

	t.Run("adds tag to document", func(t *testing.T) {
		t.Parallel()
		doc := &openapi.OpenAPI{}
		f := &addGlobalTagFix{tagName: "users"}
		require.NoError(t, f.Apply(doc))

		require.Len(t, doc.Tags, 1)
		assert.Equal(t, "users", doc.Tags[0].Name)
	})

	t.Run("idempotent when tag exists", func(t *testing.T) {
		t.Parallel()
		doc := &openapi.OpenAPI{
			Tags: []*openapi.Tag{{Name: "users"}},
		}
		f := &addGlobalTagFix{tagName: "users"}
		require.NoError(t, f.Apply(doc))

		require.Len(t, doc.Tags, 1, "should not duplicate tag")
	})

	t.Run("appends to existing tags", func(t *testing.T) {
		t.Parallel()
		doc := &openapi.OpenAPI{
			Tags: []*openapi.Tag{{Name: "pets"}},
		}
		f := &addGlobalTagFix{tagName: "users"}
		require.NoError(t, f.Apply(doc))

		require.Len(t, doc.Tags, 2)
		assert.Equal(t, "pets", doc.Tags[0].Name)
		assert.Equal(t, "users", doc.Tags[1].Name)
	})

	t.Run("wrong doc type returns error", func(t *testing.T) {
		t.Parallel()
		f := &addGlobalTagFix{tagName: "users"}
		err := f.Apply("not a doc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected *openapi.OpenAPI")
	})
}

func TestUpgradeToHTTPSFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &upgradeToHTTPSFix{}
		assert.Equal(t, "Upgrade server URL to HTTPS", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
	})

	t.Run("upgrades http to https", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("http://api.example.com")
		f := &upgradeToHTTPSFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "https://api.example.com", node.Value)
	})

	t.Run("preserves path after upgrade", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("http://api.example.com/v1/")
		f := &upgradeToHTTPSFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "https://api.example.com/v1/", node.Value)
	})

	t.Run("already https is no-op", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("https://api.example.com")
		f := &upgradeToHTTPSFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "https://api.example.com", node.Value)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &upgradeToHTTPSFix{node: nil}
		require.NoError(t, f.ApplyNode(nil))
	})

	t.Run("describe change", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("http://api.example.com")
		f := &upgradeToHTTPSFix{node: node}
		before, after := f.DescribeChange()
		assert.Equal(t, "http://api.example.com", before, "before should be original value")
		assert.Equal(t, "https://api.example.com", after, "after should be upgraded to HTTPS")
	})

	t.Run("describe change nil node", func(t *testing.T) {
		t.Parallel()
		f := &upgradeToHTTPSFix{node: nil}
		before, after := f.DescribeChange()
		assert.Empty(t, before, "before should be empty for nil node")
		assert.Empty(t, after, "after should be empty for nil node")
	})

	t.Run("describe change already https", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateStringNode("https://api.example.com")
		f := &upgradeToHTTPSFix{node: node}
		before, after := f.DescribeChange()
		assert.Empty(t, before, "before should be empty when already HTTPS")
		assert.Empty(t, after, "after should be empty when already HTTPS")
	})
}

func TestRemoveNullableFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &removeNullableFix{}
		assert.Equal(t, "Replace nullable with type array including null", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
	})

	t.Run("converts scalar type to array with null", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		typeNode := yml.CreateStringNode("string")
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			typeNode,
			yml.CreateStringNode("nullable"),
			yml.CreateBoolNode(true),
		})
		f := &removeNullableFix{schemaNode: schemaNode, typeValueNode: typeNode}
		require.NoError(t, f.ApplyNode(nil))

		// Verify nullable was removed
		_, _, found := yml.GetMapElementNodes(ctx, schemaNode, "nullable")
		assert.False(t, found, "nullable should be removed")

		// Verify type was converted to [string, null]
		_, updatedType, found := yml.GetMapElementNodes(ctx, schemaNode, "type")
		require.True(t, found)
		assert.Equal(t, yaml.SequenceNode, updatedType.Kind)
		require.Len(t, updatedType.Content, 2)
		assert.Equal(t, "string", updatedType.Content[0].Value)
		assert.Equal(t, "null", updatedType.Content[1].Value)
	})

	t.Run("appends null to existing type array", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		typeNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
			Content: []*yaml.Node{
				yml.CreateStringNode("string"),
				yml.CreateStringNode("integer"),
			},
		}
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			typeNode,
			yml.CreateStringNode("nullable"),
			yml.CreateBoolNode(true),
		})
		f := &removeNullableFix{schemaNode: schemaNode, typeValueNode: typeNode}
		require.NoError(t, f.ApplyNode(nil))

		require.Len(t, typeNode.Content, 3)
		assert.Equal(t, "string", typeNode.Content[0].Value)
		assert.Equal(t, "integer", typeNode.Content[1].Value)
		assert.Equal(t, "null", typeNode.Content[2].Value)
	})

	t.Run("does not duplicate null in type array", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		typeNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
			Content: []*yaml.Node{
				yml.CreateStringNode("string"),
				yml.CreateStringNode("null"),
			},
		}
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			typeNode,
			yml.CreateStringNode("nullable"),
			yml.CreateBoolNode(true),
		})
		f := &removeNullableFix{schemaNode: schemaNode, typeValueNode: typeNode}
		require.NoError(t, f.ApplyNode(nil))

		require.Len(t, typeNode.Content, 2, "should not add duplicate null")
	})

	t.Run("no type field adds type null", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("nullable"),
			yml.CreateBoolNode(true),
		})
		f := &removeNullableFix{schemaNode: schemaNode, typeValueNode: nil}
		require.NoError(t, f.ApplyNode(nil))

		_, typeNode, found := yml.GetMapElementNodes(ctx, schemaNode, "type")
		require.True(t, found, "type field should be added")
		assert.Equal(t, "null", typeNode.Value)
	})

	t.Run("nil schema is no-op", func(t *testing.T) {
		t.Parallel()
		f := &removeNullableFix{schemaNode: nil}
		require.NoError(t, f.ApplyNode(nil))
	})
}

func TestAppendRFC8725Fix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &appendRFC8725Fix{}
		assert.Equal(t, "Add RFC8725 mention to security scheme description", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
	})

	t.Run("appends to existing description", func(t *testing.T) {
		t.Parallel()
		descNode := yml.CreateStringNode("OAuth2 Bearer token")
		f := &appendRFC8725Fix{
			schemeNode: yml.CreateMapNode(context.Background(), nil),
			descNode:   descNode,
		}
		require.NoError(t, f.ApplyNode(nil))

		assert.Contains(t, descNode.Value, "RFC8725")
		assert.True(t, strings.HasPrefix(descNode.Value, "OAuth2 Bearer token"))
	})

	t.Run("does not duplicate RFC8725 mention", func(t *testing.T) {
		t.Parallel()
		descNode := yml.CreateStringNode("Already mentions RFC8725 best practices.")
		f := &appendRFC8725Fix{
			schemeNode: yml.CreateMapNode(context.Background(), nil),
			descNode:   descNode,
		}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "Already mentions RFC8725 best practices.", descNode.Value)
	})

	t.Run("creates description when none exists", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		schemeNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("type"),
			yml.CreateStringNode("http"),
		})
		f := &appendRFC8725Fix{schemeNode: schemeNode, descNode: nil}
		require.NoError(t, f.ApplyNode(nil))

		_, desc, found := yml.GetMapElementNodes(ctx, schemeNode, "description")
		require.True(t, found, "description should be created")
		assert.Contains(t, desc.Value, "RFC8725")
	})

	t.Run("nil scheme is no-op", func(t *testing.T) {
		t.Parallel()
		f := &appendRFC8725Fix{schemeNode: nil}
		require.NoError(t, f.ApplyNode(nil))
	})
}

func TestSetAdditionalPropertiesFalseFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &setAdditionalPropertiesFalseFix{}
		assert.Equal(t, "Set additionalProperties to false", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
	})

	t.Run("changes true to false", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateBoolNode(true)
		f := &setAdditionalPropertiesFalseFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "false", node.Value)
	})

	t.Run("already false is no-op", func(t *testing.T) {
		t.Parallel()
		node := yml.CreateBoolNode(false)
		f := &setAdditionalPropertiesFalseFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, "false", node.Value)
	})

	t.Run("non-scalar node is no-op", func(t *testing.T) {
		t.Parallel()
		node := &yaml.Node{Kind: yaml.MappingNode}
		f := &setAdditionalPropertiesFalseFix{node: node}
		require.NoError(t, f.ApplyNode(nil))

		assert.Equal(t, yaml.MappingNode, node.Kind)
	})

	t.Run("nil node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &setAdditionalPropertiesFalseFix{node: nil}
		require.NoError(t, f.ApplyNode(nil))
	})
}

func TestAddPathParameterFix(t *testing.T) {
	t.Parallel()

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()
		f := &addPathParameterFix{paramName: "userId", schemaType: "integer"}
		assert.Equal(t, "Add missing path parameter 'userId' (type: integer)", f.Description())
		assert.False(t, f.Interactive())
		assert.Nil(t, f.Prompts())
		require.NoError(t, f.SetInput(nil))
		require.NoError(t, f.Apply(nil))
	})

	t.Run("metadata with format", func(t *testing.T) {
		t.Parallel()
		f := &addPathParameterFix{paramName: "requestUuid", schemaType: "string", schemaFormat: "uuid"}
		assert.Equal(t, "Add missing path parameter 'requestUuid' (type: string, format: uuid)", f.Description())
	})

	t.Run("adds param to existing parameters sequence", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		// Create an operation node with existing parameters
		paramsSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		opNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("parameters"),
			paramsSeq,
		})

		f := &addPathParameterFix{
			operationNode: opNode,
			paramName:     "userId",
			schemaType:    "integer",
		}
		require.NoError(t, f.ApplyNode(nil))

		// Verify parameter was added
		_, updatedParams, found := yml.GetMapElementNodes(ctx, opNode, "parameters")
		require.True(t, found, "parameters should exist")
		require.Equal(t, yaml.SequenceNode, updatedParams.Kind)
		assert.Len(t, updatedParams.Content, 1, "should have one parameter")

		// Verify parameter content
		param := updatedParams.Content[0]
		_, nameNode, _ := yml.GetMapElementNodes(ctx, param, "name")
		_, inNode, _ := yml.GetMapElementNodes(ctx, param, "in")
		_, reqNode, _ := yml.GetMapElementNodes(ctx, param, "required")
		assert.Equal(t, "userId", nameNode.Value)
		assert.Equal(t, "path", inNode.Value)
		assert.Equal(t, "true", reqNode.Value)

		_, schemaNode, _ := yml.GetMapElementNodes(ctx, param, "schema")
		require.NotNil(t, schemaNode)
		_, typeNode, _ := yml.GetMapElementNodes(ctx, schemaNode, "type")
		assert.Equal(t, "integer", typeNode.Value)
	})

	t.Run("creates parameters sequence when missing", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		// Create an operation node without parameters
		opNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("summary"),
			yml.CreateStringNode("Get user"),
		})

		f := &addPathParameterFix{
			operationNode: opNode,
			paramName:     "userId",
			schemaType:    "string",
		}
		require.NoError(t, f.ApplyNode(nil))

		// Verify parameters was created
		_, updatedParams, found := yml.GetMapElementNodes(ctx, opNode, "parameters")
		require.True(t, found, "parameters should be created")
		require.Equal(t, yaml.SequenceNode, updatedParams.Kind)
		assert.Len(t, updatedParams.Content, 1, "should have one parameter")
	})

	t.Run("includes format when specified", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		opNode := yml.CreateMapNode(ctx, nil)

		f := &addPathParameterFix{
			operationNode: opNode,
			paramName:     "requestUuid",
			schemaType:    "string",
			schemaFormat:  "uuid",
		}
		require.NoError(t, f.ApplyNode(nil))

		_, paramsNode, _ := yml.GetMapElementNodes(ctx, opNode, "parameters")
		param := paramsNode.Content[0]
		_, schemaNode, _ := yml.GetMapElementNodes(ctx, param, "schema")
		_, formatNode, found := yml.GetMapElementNodes(ctx, schemaNode, "format")
		require.True(t, found, "format should be present")
		assert.Equal(t, "uuid", formatNode.Value)
	})

	t.Run("idempotent - does not add duplicate", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		// Pre-populate with existing path param
		existingParam := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("name"),
			yml.CreateStringNode("userId"),
			yml.CreateStringNode("in"),
			yml.CreateStringNode("path"),
		})
		paramsSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{existingParam}}
		opNode := yml.CreateMapNode(ctx, []*yaml.Node{
			yml.CreateStringNode("parameters"),
			paramsSeq,
		})

		f := &addPathParameterFix{
			operationNode: opNode,
			paramName:     "userId",
			schemaType:    "integer",
		}
		require.NoError(t, f.ApplyNode(nil))

		_, updatedParams, _ := yml.GetMapElementNodes(ctx, opNode, "parameters")
		assert.Len(t, updatedParams.Content, 1, "should still have one parameter (no duplicate)")
	})

	t.Run("nil operation node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &addPathParameterFix{operationNode: nil, paramName: "userId", schemaType: "string"}
		require.NoError(t, f.ApplyNode(nil))
	})

	t.Run("non-mapping operation node is no-op", func(t *testing.T) {
		t.Parallel()
		f := &addPathParameterFix{
			operationNode: &yaml.Node{Kind: yaml.SequenceNode},
			paramName:     "userId",
			schemaType:    "string",
		}
		require.NoError(t, f.ApplyNode(nil))
	})
}

func TestInferPathParamType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		paramName      string
		expectedType   string
		expectedFormat string
	}{
		{name: "userId inferred as integer", paramName: "userId", expectedType: "integer", expectedFormat: ""},
		{name: "postId inferred as integer", paramName: "postId", expectedType: "integer", expectedFormat: ""},
		{name: "orgid inferred as integer", paramName: "orgid", expectedType: "integer", expectedFormat: ""},
		{name: "requestUuid inferred as string uuid", paramName: "requestUuid", expectedType: "string", expectedFormat: "uuid"},
		{name: "sessionGuid inferred as string uuid", paramName: "sessionGuid", expectedType: "string", expectedFormat: "uuid"},
		{name: "UUID uppercase inferred as string uuid", paramName: "UUID", expectedType: "string", expectedFormat: "uuid"},
		{name: "name inferred as string", paramName: "name", expectedType: "string", expectedFormat: ""},
		{name: "slug inferred as string", paramName: "slug", expectedType: "string", expectedFormat: ""},
		{name: "version inferred as string", paramName: "version", expectedType: "string", expectedFormat: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			schemaType, format := inferPathParamType(tt.paramName)
			assert.Equal(t, tt.expectedType, schemaType, "schema type should match")
			assert.Equal(t, tt.expectedFormat, format, "format should match")
		})
	}
}
