package asyncapi

import (
	"errors"
	"testing"

	jsValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestDedupeErrors_Success(t *testing.T) {
	t.Parallel()

	first := errors.New("first")
	second := errors.New("second")

	actual := dedupeErrors([]error{first, errors.New("first"), second, first})
	assert.Equal(t, []error{first, second}, actual, "duplicate messages should be removed while keeping order")
}

func TestDraft07Errors_NotValidationError_Error(t *testing.T) {
	t.Parallel()

	node := &yaml.Node{Kind: yaml.MappingNode, Line: 2, Column: 3}
	actual := draft07Errors(errors.New("unexpected"), node)
	require.Len(t, actual, 1, "an unexpected error should be reported")
	assert.Equal(t, "[2:3] error validation-invalid-schema schema invalid: unexpected", actual[0].Error(), "error should match")
}

func TestFilterBranchTypeMismatches_Success(t *testing.T) {
	t.Parallel()

	typeOnly := &jsValidator.ValidationError{ErrorKind: &kind.Type{Got: "number", Want: []string{"array"}}}
	otherTypeOnly := &jsValidator.ValidationError{ErrorKind: &kind.Type{Got: "number", Want: []string{"object", "boolean"}}}
	typeWithCauses := &jsValidator.ValidationError{ErrorKind: &kind.Type{Got: "number", Want: []string{"string"}}, Causes: []*jsValidator.ValidationError{typeOnly}}
	other := &jsValidator.ValidationError{ErrorKind: &kind.MinLength{Got: 0, Want: 1}}

	tests := []struct {
		name     string
		err      *jsValidator.ValidationError
		expected []*jsValidator.ValidationError
	}{
		{
			name:     "causes of a non branching keyword are kept",
			err:      &jsValidator.ValidationError{ErrorKind: &kind.Required{Missing: []string{"a"}}, Causes: []*jsValidator.ValidationError{typeOnly, other}},
			expected: []*jsValidator.ValidationError{typeOnly, other},
		},
		{
			name:     "anyOf drops branches that only mismatch on type",
			err:      &jsValidator.ValidationError{ErrorKind: &kind.AnyOf{}, Causes: []*jsValidator.ValidationError{typeOnly, other}},
			expected: []*jsValidator.ValidationError{other},
		},
		{
			name:     "oneOf keeps type mismatches that have causes",
			err:      &jsValidator.ValidationError{ErrorKind: &kind.OneOf{}, Causes: []*jsValidator.ValidationError{typeOnly, typeWithCauses}},
			expected: []*jsValidator.ValidationError{typeWithCauses},
		},
		{
			name:     "all branches are kept when every branch only mismatches on type",
			err:      &jsValidator.ValidationError{ErrorKind: &kind.AnyOf{}, Causes: []*jsValidator.ValidationError{typeOnly, otherTypeOnly}},
			expected: []*jsValidator.ValidationError{typeOnly, otherTypeOnly},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, filterBranchTypeMismatches(tt.err), "filtered causes should match")
		})
	}
}

func TestRootCauses_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             *jsValidator.ValidationError
		expectedRule    string
		expectedMessage string
		expectedLine    int
	}{
		{
			name:            "missing required property",
			err:             &jsValidator.ValidationError{InstanceLocation: []string{"externalDocs"}, ErrorKind: &kind.Required{Missing: []string{"url"}}},
			expectedRule:    validation.RuleValidationRequiredField,
			expectedMessage: "`schema.externalDocs` missing property `url`",
			expectedLine:    3,
		},
		{
			name:            "other keyword at the root",
			err:             &jsValidator.ValidationError{ErrorKind: &kind.MinLength{Got: 0, Want: 1}},
			expectedRule:    validation.RuleValidationInvalidSchema,
			expectedMessage: "schema minLength: got 0, want 1",
			expectedLine:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte("type: object\nexternalDocs:\n  description: More\n"), &doc), "yaml should parse")
			root := doc.Content[0]

			errs := rootCauses(tt.err, root)
			require.Len(t, errs, 1, "should report a single error")

			var vErr *validation.Error
			require.ErrorAs(t, errs[0], &vErr, "should be a validation error")
			assert.Equal(t, tt.expectedRule, vErr.Rule, "rule should match")
			assert.Equal(t, tt.expectedMessage, vErr.UnderlyingError.Error(), "message should match")
			assert.Equal(t, tt.expectedLine, vErr.GetLineNumber(), "line should match")
		})
	}
}

func TestNodeAtPath_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	danglingAlias := &yaml.Node{Kind: yaml.AliasNode}

	list := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{yml.CreateStringNode("x"), yml.CreateStringNode("y")}}
	scalar := yml.CreateStringNode("c")
	mapping := yml.CreateMapNode(ctx, []*yaml.Node{
		yml.CreateStringNode("a"), list,
		yml.CreateStringNode("b"), scalar,
	})

	keyed := yml.CreateStringNode("v")
	withDanglingKey := yml.CreateMapNode(ctx, []*yaml.Node{
		danglingAlias, yml.CreateStringNode("ignored"),
		yml.CreateStringNode("k"), keyed,
	})
	withDanglingValue := yml.CreateMapNode(ctx, []*yaml.Node{
		yml.CreateStringNode("k"), danglingAlias,
	})

	tests := []struct {
		name     string
		root     *yaml.Node
		path     []string
		expected *yaml.Node
	}{
		{
			name:     "empty path returns the root",
			root:     mapping,
			path:     nil,
			expected: mapping,
		},
		{
			name:     "mapping key then sequence index",
			root:     mapping,
			path:     []string{"a", "1"},
			expected: list.Content[1],
		},
		{
			name:     "missing key falls back to the closest ancestor",
			root:     mapping,
			path:     []string{"missing"},
			expected: mapping,
		},
		{
			name:     "out of range index falls back to the sequence",
			root:     mapping,
			path:     []string{"a", "5"},
			expected: list,
		},
		{
			name:     "non numeric index falls back to the sequence",
			root:     mapping,
			path:     []string{"a", "x"},
			expected: list,
		},
		{
			name:     "path through a scalar falls back to the scalar",
			root:     mapping,
			path:     []string{"b", "c"},
			expected: scalar,
		},
		{
			name:     "keys that are dangling aliases are skipped",
			root:     withDanglingKey,
			path:     []string{"k"},
			expected: keyed,
		},
		{
			name:     "dangling alias value falls back to the root",
			root:     withDanglingValue,
			path:     []string{"k", "x"},
			expected: withDanglingValue,
		},
		{
			name:     "dangling alias as the final value falls back to the root",
			root:     withDanglingValue,
			path:     []string{"k"},
			expected: withDanglingValue,
		},
		{
			name:     "dangling alias root",
			root:     danglingAlias,
			path:     []string{"a"},
			expected: danglingAlias,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Same(t, tt.expected, nodeAtPath(tt.root, tt.path), "node should match")
		})
	}
}

func TestSchemaNodeToJSON_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	danglingAlias := &yaml.Node{Kind: yaml.AliasNode}

	tests := []struct {
		name     string
		node     *yaml.Node
		expected any
	}{
		{
			name:     "nil node",
			node:     nil,
			expected: nil,
		},
		{
			name: "keys that are dangling aliases are skipped",
			node: yml.CreateMapNode(ctx, []*yaml.Node{
				danglingAlias, yml.CreateStringNode("ignored"),
				yml.CreateStringNode("type"), yml.CreateStringNode("string"),
			}),
			expected: map[string]any{"type": "string"},
		},
		{
			name: "dependency keys that are dangling aliases are skipped",
			node: yml.CreateMapNode(ctx, []*yaml.Node{
				yml.CreateStringNode("dependencies"), yml.CreateMapNode(ctx, []*yaml.Node{
					danglingAlias, yml.CreateStringNode("ignored"),
					yml.CreateStringNode("a"), {Kind: yaml.SequenceNode, Content: []*yaml.Node{yml.CreateStringNode("b")}},
				}),
			}),
			expected: map[string]any{"dependencies": map[string]any{"a": []any{"b"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := schemaNodeToJSON(tt.node)
			require.NoError(t, err, "conversion should succeed")
			assert.Equal(t, tt.expected, actual, "converted value should match")
		})
	}
}

func TestValueToJSON_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	tests := []struct {
		name     string
		node     *yaml.Node
		expected any
	}{
		{
			name:     "nil node",
			node:     nil,
			expected: nil,
		},
		{
			name: "keys that are dangling aliases are skipped",
			node: yml.CreateMapNode(ctx, []*yaml.Node{
				{Kind: yaml.AliasNode}, yml.CreateStringNode("ignored"),
				yml.CreateStringNode("a"), yml.CreateBoolNode(true),
			}),
			expected: map[string]any{"a": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := valueToJSON(tt.node)
			require.NoError(t, err, "conversion should succeed")
			assert.Equal(t, tt.expected, actual, "converted value should match")
		})
	}
}

func TestValueToJSON_Error(t *testing.T) {
	t.Parallel()

	_, err := valueToJSON(&yaml.Node{Kind: yaml.DocumentNode})
	require.EqualError(t, err, "unsupported node kind `document`", "document nodes should not be converted")
}
