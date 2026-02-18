package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func createCriterionWithRootNode(c Criterion, rootNode *yaml.Node) Criterion {
	c.SetRootNode(rootNode)
	c.SetValid(true, true)
	return c
}

func createCriterionTypeUnionWithRootNode(ctu CriterionTypeUnion, rootNode *yaml.Node) CriterionTypeUnion {
	ctu.SetRootNode(rootNode)
	ctu.SetValid(true, true)
	return ctu
}

func createCriterionExpressionTypeWithRootNode(cet CriterionExpressionType, rootNode *yaml.Node) CriterionExpressionType {
	cet.SetRootNode(rootNode)
	cet.SetValid(true, true)
	return cet
}

func TestCriterion_Unmarshal_Success(t *testing.T) {
	t.Parallel()
	type args struct {
		testYaml string
	}
	tests := []struct {
		name string
		args args
		want Criterion
	}{
		{
			name: "simple",
			args: args{
				testYaml: `condition: $statusCode == 200`,
			},
			want: createCriterionWithRootNode(Criterion{
				Condition: marshaller.Node[string]{
					Key:       "condition",
					KeyNode:   testutils.CreateStringYamlNode("condition", 1, 1),
					Value:     "$statusCode == 200",
					ValueNode: testutils.CreateStringYamlNode("$statusCode == 200", 1, 12),
					Present:   true,
				},
				Type: marshaller.Node[CriterionTypeUnion]{},
			}, testutils.CreateMapYamlNode([]*yaml.Node{
				testutils.CreateStringYamlNode("condition", 1, 1),
				testutils.CreateStringYamlNode("$statusCode == 200", 1, 12),
			}, 1, 1)),
		},
		{
			name: "simple with string type",
			args: args{
				testYaml: `condition: $statusCode == 200
type: simple`,
			},
			want: createCriterionWithRootNode(Criterion{
				Condition: marshaller.Node[string]{
					Key:       "condition",
					KeyNode:   testutils.CreateStringYamlNode("condition", 1, 1),
					Value:     "$statusCode == 200",
					ValueNode: testutils.CreateStringYamlNode("$statusCode == 200", 1, 12),
					Present:   true,
				},
				Type: marshaller.Node[CriterionTypeUnion]{
					Key:       "type",
					KeyNode:   testutils.CreateStringYamlNode("type", 2, 1),
					Value:     createCriterionTypeUnionWithRootNode(CriterionTypeUnion{Type: pointer.From("simple")}, testutils.CreateStringYamlNode("simple", 2, 7)),
					ValueNode: testutils.CreateStringYamlNode("simple", 2, 7),
					Present:   true,
				},
			}, testutils.CreateMapYamlNode([]*yaml.Node{
				testutils.CreateStringYamlNode("condition", 1, 1),
				testutils.CreateStringYamlNode("$statusCode == 200", 1, 12),
				testutils.CreateStringYamlNode("type", 2, 1),
				testutils.CreateStringYamlNode("simple", 2, 7),
			}, 1, 1)),
		},
		{
			name: "json path",
			args: args{
				testYaml: `context: $response.body
condition: $[?count(@.pets) > 0]
type: jsonpath`,
			},
			want: createCriterionWithRootNode(Criterion{
				Condition: marshaller.Node[string]{
					Key:       "condition",
					KeyNode:   testutils.CreateStringYamlNode("condition", 2, 1),
					Value:     "$[?count(@.pets) > 0]",
					ValueNode: testutils.CreateStringYamlNode("$[?count(@.pets) > 0]", 2, 12),
					Present:   true,
				},
				Context: marshaller.Node[*string]{
					Key:       "context",
					KeyNode:   testutils.CreateStringYamlNode("context", 1, 1),
					Value:     pointer.From("$response.body"),
					ValueNode: testutils.CreateStringYamlNode("$response.body", 1, 10),
					Present:   true,
				},
				Type: marshaller.Node[CriterionTypeUnion]{
					Key:       "type",
					KeyNode:   testutils.CreateStringYamlNode("type", 3, 1),
					Value:     createCriterionTypeUnionWithRootNode(CriterionTypeUnion{Type: pointer.From("jsonpath")}, testutils.CreateStringYamlNode("jsonpath", 3, 7)),
					ValueNode: testutils.CreateStringYamlNode("jsonpath", 3, 7),
					Present:   true,
				},
			}, testutils.CreateMapYamlNode([]*yaml.Node{
				testutils.CreateStringYamlNode("context", 1, 1),
				testutils.CreateStringYamlNode("$response.body", 1, 10),
				testutils.CreateStringYamlNode("condition", 2, 1),
				testutils.CreateStringYamlNode("$[?count(@.pets) > 0]", 2, 12),
				testutils.CreateStringYamlNode("type", 3, 1),
				testutils.CreateStringYamlNode("jsonpath", 3, 7),
			}, 1, 1)),
		},
		{
			name: "json path with type and version",
			args: args{
				testYaml: `context: $response.body
condition: $[?count(@.pets) > 0]
type:
  type: jsonpath
  version: draft-goessner-dispatch-jsonpath-00`,
			},
			want: createCriterionWithRootNode(Criterion{
				Condition: marshaller.Node[string]{
					Key:       "condition",
					KeyNode:   testutils.CreateStringYamlNode("condition", 2, 1),
					Value:     "$[?count(@.pets) > 0]",
					ValueNode: testutils.CreateStringYamlNode("$[?count(@.pets) > 0]", 2, 12),
					Present:   true,
				},
				Context: marshaller.Node[*string]{
					Key:       "context",
					KeyNode:   testutils.CreateStringYamlNode("context", 1, 1),
					Value:     pointer.From("$response.body"),
					ValueNode: testutils.CreateStringYamlNode("$response.body", 1, 10),
					Present:   true,
				},
				Type: marshaller.Node[CriterionTypeUnion]{
					Key:     "type",
					KeyNode: testutils.CreateStringYamlNode("type", 3, 1),
					Value: createCriterionTypeUnionWithRootNode(CriterionTypeUnion{
						ExpressionType: func() *CriterionExpressionType {
							cet := createCriterionExpressionTypeWithRootNode(CriterionExpressionType{
								Type: marshaller.Node[string]{
									Key:       "type",
									KeyNode:   testutils.CreateStringYamlNode("type", 4, 3),
									Value:     "jsonpath",
									ValueNode: testutils.CreateStringYamlNode("jsonpath", 4, 9),
									Present:   true,
								},
								Version: marshaller.Node[string]{
									Key:       "version",
									KeyNode:   testutils.CreateStringYamlNode("version", 5, 3),
									Value:     "draft-goessner-dispatch-jsonpath-00",
									ValueNode: testutils.CreateStringYamlNode("draft-goessner-dispatch-jsonpath-00", 5, 12),
									Present:   true,
								},
							}, testutils.CreateMapYamlNode([]*yaml.Node{
								testutils.CreateStringYamlNode("type", 4, 3),
								testutils.CreateStringYamlNode("jsonpath", 4, 9),
								testutils.CreateStringYamlNode("version", 5, 3),
								testutils.CreateStringYamlNode("draft-goessner-dispatch-jsonpath-00", 5, 12),
							}, 4, 3))
							return &cet
						}(),
					}, testutils.CreateMapYamlNode([]*yaml.Node{
						testutils.CreateStringYamlNode("type", 4, 3),
						testutils.CreateStringYamlNode("jsonpath", 4, 9),
						testutils.CreateStringYamlNode("version", 5, 3),
						testutils.CreateStringYamlNode("draft-goessner-dispatch-jsonpath-00", 5, 12),
					}, 4, 3)),
					ValueNode: testutils.CreateMapYamlNode([]*yaml.Node{
						testutils.CreateStringYamlNode("type", 4, 3),
						testutils.CreateStringYamlNode("jsonpath", 4, 9),
						testutils.CreateStringYamlNode("version", 5, 3),
						testutils.CreateStringYamlNode("draft-goessner-dispatch-jsonpath-00", 5, 12),
					}, 4, 3),
					Present: true,
				},
			}, testutils.CreateMapYamlNode([]*yaml.Node{
				testutils.CreateStringYamlNode("context", 1, 1),
				testutils.CreateStringYamlNode("$response.body", 1, 10),
				testutils.CreateStringYamlNode("condition", 2, 1),
				testutils.CreateStringYamlNode("$[?count(@.pets) > 0]", 2, 12),
				testutils.CreateStringYamlNode("type", 3, 1),
				testutils.CreateMapYamlNode([]*yaml.Node{
					testutils.CreateStringYamlNode("type", 4, 3),
					testutils.CreateStringYamlNode("jsonpath", 4, 9),
					testutils.CreateStringYamlNode("version", 5, 3),
					testutils.CreateStringYamlNode("draft-goessner-dispatch-jsonpath-00", 5, 12),
				}, 4, 3),
			}, 1, 1)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var doc yaml.Node
			err := yaml.Unmarshal([]byte(tt.args.testYaml), &doc)
			require.NoError(t, err)

			c := Criterion{}

			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", doc.Content[0], &c)
			require.NoError(t, err)
			require.Empty(t, validationErrs, "Expected no validation errors")

			require.Equal(t, tt.want, c)
		})
	}
}

func TestCriterionTypeUnion_Unmarshal_NilNode_Error(t *testing.T) {
	t.Parallel()

	var union CriterionTypeUnion
	_, err := union.Unmarshal(t.Context(), "test", nil)
	require.Error(t, err, "should return error for nil node")
	require.Contains(t, err.Error(), "node is nil", "error should mention nil node")
}

func TestCriterionTypeUnion_Unmarshal_InvalidNodeKind_Error(t *testing.T) {
	t.Parallel()

	var union CriterionTypeUnion
	node := &yaml.Node{Kind: yaml.SequenceNode}
	validationErrs, err := union.Unmarshal(t.Context(), "test", node)
	require.NoError(t, err, "should not return fatal error")
	require.NotEmpty(t, validationErrs, "should have validation errors")
	require.Contains(t, validationErrs[0].Error(), "expected string or object", "error should mention expected types")
}

func TestCriterionTypeUnion_SyncChanges_Int_Error(t *testing.T) {
	t.Parallel()

	union := &CriterionTypeUnion{}
	union.SetValid(true, true)

	_, err := union.SyncChanges(t.Context(), 42, nil)
	require.Error(t, err, "should return error for int model")
	require.Contains(t, err.Error(), "expected a struct, got `int`", "error should mention struct expectation")
}
