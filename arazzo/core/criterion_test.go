package core

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCriterion_Unmarshal_Success(t *testing.T) {
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
			want: Criterion{
				Condition: marshaller.Node[string]{
					Key:       "condition",
					KeyNode:   testutils.CreateStringYamlNode("condition", 1, 1),
					Value:     "$statusCode == 200",
					ValueNode: testutils.CreateStringYamlNode("$statusCode == 200", 1, 12),
					Present:   true,
				},
				Type: marshaller.Node[CriterionTypeUnion]{},
				RootNode: testutils.CreateMapYamlNode([]*yaml.Node{
					testutils.CreateStringYamlNode("condition", 1, 1),
					testutils.CreateStringYamlNode("$statusCode == 200", 1, 12),
				}, 1, 1),
			},
		},
		{
			name: "simple with string type",
			args: args{
				testYaml: `condition: $statusCode == 200
type: simple`,
			},
			want: Criterion{
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
					Value:     CriterionTypeUnion{Type: pointer.From("simple"), RootNode: testutils.CreateStringYamlNode("simple", 2, 7)},
					ValueNode: testutils.CreateStringYamlNode("simple", 2, 7),
					Present:   true,
				},
				RootNode: testutils.CreateMapYamlNode([]*yaml.Node{
					testutils.CreateStringYamlNode("condition", 1, 1),
					testutils.CreateStringYamlNode("$statusCode == 200", 1, 12),
					testutils.CreateStringYamlNode("type", 2, 1),
					testutils.CreateStringYamlNode("simple", 2, 7),
				}, 1, 1),
			},
		},
		{
			name: "json path",
			args: args{
				testYaml: `context: $response.body
condition: $[?count(@.pets) > 0]
type: jsonpath`,
			},
			want: Criterion{
				Condition: marshaller.Node[string]{
					Key:       "condition",
					KeyNode:   testutils.CreateStringYamlNode("condition", 2, 1),
					Value:     "$[?count(@.pets) > 0]",
					ValueNode: testutils.CreateStringYamlNode("$[?count(@.pets) > 0]", 2, 12),
					Present:   true,
				},
				Context: marshaller.Node[*Expression]{
					Key:       "context",
					KeyNode:   testutils.CreateStringYamlNode("context", 1, 1),
					Value:     pointer.From[Expression]("$response.body"),
					ValueNode: testutils.CreateStringYamlNode("$response.body", 1, 10),
					Present:   true,
				},
				Type: marshaller.Node[CriterionTypeUnion]{
					Key:       "type",
					KeyNode:   testutils.CreateStringYamlNode("type", 3, 1),
					Value:     CriterionTypeUnion{Type: pointer.From("jsonpath"), RootNode: testutils.CreateStringYamlNode("jsonpath", 3, 7)},
					ValueNode: testutils.CreateStringYamlNode("jsonpath", 3, 7),
					Present:   true,
				},
				RootNode: testutils.CreateMapYamlNode([]*yaml.Node{
					testutils.CreateStringYamlNode("context", 1, 1),
					testutils.CreateStringYamlNode("$response.body", 1, 10),
					testutils.CreateStringYamlNode("condition", 2, 1),
					testutils.CreateStringYamlNode("$[?count(@.pets) > 0]", 2, 12),
					testutils.CreateStringYamlNode("type", 3, 1),
					testutils.CreateStringYamlNode("jsonpath", 3, 7),
				}, 1, 1),
			},
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
			want: Criterion{
				Condition: marshaller.Node[string]{
					Key:       "condition",
					KeyNode:   testutils.CreateStringYamlNode("condition", 2, 1),
					Value:     "$[?count(@.pets) > 0]",
					ValueNode: testutils.CreateStringYamlNode("$[?count(@.pets) > 0]", 2, 12),
					Present:   true,
				},
				Context: marshaller.Node[*Expression]{
					Key:       "context",
					KeyNode:   testutils.CreateStringYamlNode("context", 1, 1),
					Value:     pointer.From[Expression]("$response.body"),
					ValueNode: testutils.CreateStringYamlNode("$response.body", 1, 10),
					Present:   true,
				},
				Type: marshaller.Node[CriterionTypeUnion]{
					Key:     "type",
					KeyNode: testutils.CreateStringYamlNode("type", 3, 1),
					Value: CriterionTypeUnion{
						ExpressionType: &CriterionExpressionType{
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
						}, RootNode: testutils.CreateMapYamlNode([]*yaml.Node{
							testutils.CreateStringYamlNode("type", 4, 3),
							testutils.CreateStringYamlNode("jsonpath", 4, 9),
							testutils.CreateStringYamlNode("version", 5, 3),
							testutils.CreateStringYamlNode("draft-goessner-dispatch-jsonpath-00", 5, 12),
						}, 4, 3),
					},
					ValueNode: testutils.CreateMapYamlNode([]*yaml.Node{
						testutils.CreateStringYamlNode("type", 4, 3),
						testutils.CreateStringYamlNode("jsonpath", 4, 9),
						testutils.CreateStringYamlNode("version", 5, 3),
						testutils.CreateStringYamlNode("draft-goessner-dispatch-jsonpath-00", 5, 12),
					}, 4, 3),
					Present: true,
				},
				RootNode: testutils.CreateMapYamlNode([]*yaml.Node{
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
				}, 1, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc yaml.Node
			err := yaml.Unmarshal([]byte(tt.args.testYaml), &doc)
			require.NoError(t, err)

			c := Criterion{}
			err = c.Unmarshal(context.Background(), doc.Content[0])
			require.NoError(t, err)

			require.Equal(t, tt.want, c)
		})
	}
}
