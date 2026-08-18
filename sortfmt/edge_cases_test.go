package sortfmt

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestFormat_ReadError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := Format(failingReader{}, &output)
	require.ErrorContains(t, err, "read document")
	require.ErrorIs(t, err, errTestIO)
}

func TestParseJSON_RejectsTrailingValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{name: "second value", input: `{} []`, contains: "multiple JSON values"},
		{name: "invalid trailing data", input: `{} invalid`, contains: "invalid character"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseJSON([]byte(tt.input))
			require.ErrorContains(t, err, tt.contains)
		})
	}
}

func TestFormat_UnicodeEscapeBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expected      string
		errorContains string
	}{
		{
			name:     "ordinary escape",
			input:    `{"a":"\u0061"}`,
			expected: "{\n  \"a\": \"a\"\n}\n",
		},
		{
			name:          "high surrogate without pair",
			input:         `{"a":"\ud83dX"}`,
			errorContains: "lone surrogate escape",
		},
		{
			name:          "high surrogate followed by non-low escape",
			input:         `{"a":"\ud83d\u0041"}`,
			errorContains: "lone surrogate escape",
		},
		{
			name:          "escaped backslash breaks pair",
			input:         `{"a":"\ud83d\\ude00"}`,
			errorContains: "lone surrogate escape",
		},
		{
			name:          "surrogates cannot pair across strings",
			input:         `{"a":"\ud83d","b":"\ude00"}`,
			errorContains: "lone surrogate escape",
		},
		{
			name:          "reported byte offset",
			input:         `{"a":"\ud83d"}`,
			errorContains: "at byte 6",
		},
		{
			name:     "escaped backslash before valid pair",
			input:    `{"a":"\\\ud83d\ude00"}`,
			expected: "{\n  \"a\": \"\\\\\\ud83d\\ude00\"\n}\n",
		},
		{
			name:     "escaped quote",
			input:    `{"a":"quote: \""}`,
			expected: "{\n  \"a\": \"quote: \\\"\"\n}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			err := Format(strings.NewReader(tt.input), &output)
			if tt.errorContains != "" {
				require.ErrorContains(t, err, tt.errorContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, output.String())
		})
	}
}

func FuzzFormatRoundTrip(f *testing.F) {
	for _, seed := range []string{`\`, `\\u`, `"\ud83d`, "\u2028", "😀"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		input, err := json.Marshal(map[string]string{"k": value})
		require.NoError(t, err)

		var normalized map[string]string
		require.NoError(t, json.Unmarshal(input, &normalized))

		var formatted bytes.Buffer
		require.NoError(t, Format(bytes.NewReader(input), &formatted))

		var decoded map[string]string
		require.NoError(t, json.Unmarshal(formatted.Bytes(), &decoded))
		assert.Equal(t, normalized, decoded, "formatting should preserve JSON string semantics")

		var idempotent bytes.Buffer
		require.NoError(t, Format(bytes.NewReader(formatted.Bytes()), &idempotent))
		assert.Equal(t, formatted.Bytes(), idempotent.Bytes(), "formatted output should be idempotent")
	})
}

func TestParseHexCodeUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		start    int
		expected uint16
		ok       bool
	}{
		{name: "decimal", input: "0031", expected: 0x31, ok: true},
		{name: "lowercase", input: "d83d", expected: 0xd83d, ok: true},
		{name: "uppercase", input: "DE00", expected: 0xde00, ok: true},
		{name: "offset", input: "xx0041yy", start: 2, expected: 0x41, ok: true},
		{name: "too short", input: "123", ok: false},
		{name: "invalid digit", input: "12g4", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, ok := parseHexCodeUnit([]byte(tt.input), tt.start)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestJSONValueToNode_UnsupportedNestedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{name: "root", value: make(chan struct{})},
		{name: "mapping child", value: map[string]any{"bad": make(chan struct{})}},
		{name: "sequence child", value: []any{make(chan struct{})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := jsonValueToNode(tt.value)
			require.ErrorContains(t, err, "unsupported JSON value type")
		})
	}
}

func TestSort_RejectsMalformedNodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		node     *yaml.Node
		contains string
	}{
		{
			name:     "empty document",
			node:     &yaml.Node{Kind: yaml.DocumentNode},
			contains: "document must contain exactly one value",
		},
		{
			name:     "unsupported node",
			node:     &yaml.Node{Kind: yaml.AliasNode},
			contains: "unsupported YAML node kind",
		},
		{
			name: "unmatched mapping key",
			node: &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{stringNode("key")},
			},
			contains: "mapping has an unmatched key",
		},
		{
			name: "non-string mapping key",
			node: &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{intNode("1"), stringNode("value")},
			},
			contains: "JSON object key must be a string",
		},
		{
			name: "unsupported mapping value",
			node: &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{stringNode("key"), {Kind: yaml.AliasNode}},
			},
			contains: "unsupported YAML node kind",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorContains(t, Sort(tt.node), tt.contains)
		})
	}
}

func TestSort_PreservesRootSequenceOrder(t *testing.T) {
	t.Parallel()

	node := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: []*yaml.Node{stringNode("z"), stringNode("a")},
	}
	require.NoError(t, Sort(node))
	assert.Equal(t, "z", node.Content[0].Value)
	assert.Equal(t, "a", node.Content[1].Value)
}

func TestSort_PreservesMappingWhenNestedSortFails(t *testing.T) {
	t.Parallel()

	document, err := parseJSON([]byte(`{"c":3,"b":{"oneOf":[{"type":"string"},{"type":["string","null"]}]},"a":1}`))
	require.NoError(t, err)
	root := document.Content[0]
	originalContent := append([]*yaml.Node(nil), root.Content...)

	err = Sort(document)
	require.ErrorContains(t, err, "sort oneOf items")
	assert.Equal(t, originalContent, root.Content, "a nested error must not truncate or reorder the containing mapping")
}

func TestFormat_ParentSpecificKeyOrdering(t *testing.T) {
	t.Parallel()

	input := `{"paths":{"/a":{"get":{},"description":"","parameters":[]}},"properties":{"type":{},"name":{},"a":{}}}`
	expected := `{
  "paths": {
    "/a": {
      "description": "",
      "parameters": [],
      "get": {}
    }
  },
  "properties": {
    "a": {},
    "name": {},
    "type": {}
  }
}
`

	var output bytes.Buffer
	require.NoError(t, Format(strings.NewReader(input), &output))
	assert.Equal(t, expected, output.String())
}

func TestPythonSortKey_ListOrderingAndErrors(t *testing.T) {
	t.Parallel()

	less, err := (pythonSortKey{kind: pythonSortKeyStringList, stringList: []string{"a"}}).less(
		pythonSortKey{kind: pythonSortKeyStringList, stringList: []string{"a", "b"}},
	)
	require.NoError(t, err)
	assert.True(t, less, "a shorter equal-prefix list should sort first")

	_, err = (pythonSortKey{kind: pythonSortKeyKind(99)}).less(pythonSortKey{kind: pythonSortKeyKind(99)})
	require.ErrorContains(t, err, "unsupported selected-list sort key kind")
}

func TestListSortKey_RejectsInvalidCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		node     *yaml.Node
		contains string
	}{
		{
			name:     "non-string scalar key",
			node:     mappingNode("name", intNode("1")),
			contains: "name sort key must be a string or string array",
		},
		{
			name:     "non-string array item",
			node:     mappingNode("type", sequenceNode(stringNode("string"), intNode("1"))),
			contains: "type array sort key item 1 must be a string",
		},
		{
			name:     "invalid mapping fallback",
			node:     mappingNode("other", &yaml.Node{Kind: yaml.AliasNode}),
			contains: "unsupported YAML node kind",
		},
		{
			name:     "invalid scalar fallback",
			node:     &yaml.Node{Kind: yaml.AliasNode},
			contains: "unsupported YAML node kind",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := listSortKey(tt.node)
			require.ErrorContains(t, err, tt.contains)
		})
	}
}

func TestPythonRepr_CompositeAndErrors(t *testing.T) {
	t.Parallel()

	composite := mappingNode("key", sequenceNode(
		stringNode("value"),
		boolNode("true"),
		intNode("42"),
		floatNode("1.5"),
		nullNode(),
	))
	actual, err := pythonRepr(composite)
	require.NoError(t, err)
	assert.Equal(t, `{'key': ['value', True, 42, 1.5, None]}`, actual)

	tests := []struct {
		name     string
		node     *yaml.Node
		contains string
	}{
		{name: "invalid mapping child", node: mappingNode("key", boolNode("yes")), contains: "invalid boolean value"},
		{name: "invalid sequence child", node: sequenceNode(boolNode("yes")), contains: "invalid boolean value"},
		{name: "invalid boolean", node: boolNode("yes"), contains: "invalid boolean value"},
		{name: "unsupported scalar", node: &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!timestamp"}, contains: "unsupported scalar tag"},
		{name: "unsupported node", node: &yaml.Node{Kind: yaml.AliasNode}, contains: "unsupported YAML node kind"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := pythonRepr(tt.node)
			require.ErrorContains(t, err, tt.contains)
		})
	}
}

func TestQuotePythonString_EscapesLikePythonRepr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "prefer double quote", input: "it's", expected: `"it's"`},
		{name: "escape selected quote", input: `both ' and "`, expected: `'both \' and "'`},
		{name: "common escapes", input: "\\\n\r\t", expected: `'\\\n\r\t'`},
		{name: "byte escape", input: "\u0001", expected: `'\x01'`},
		{name: "unicode escape", input: "\u200b", expected: `'\u200b'`},
		{name: "long unicode escape", input: "\U0001d173", expected: `'\U0001d173'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, quotePythonString(tt.input))
		})
	}
}

func TestNumberFormattingErrorsAndRepr(t *testing.T) {
	t.Parallel()

	_, err := formatInteger("not-an-integer")
	require.ErrorContains(t, err, "invalid JSON integer")

	integer, err := formatFloat("42")
	require.NoError(t, err)
	assert.Equal(t, "42", integer)

	_, err = formatFloat("1.2.3")
	require.ErrorContains(t, err, "invalid JSON number")

	tests := []struct {
		input    string
		expected string
	}{
		{input: "1e309", expected: "inf"},
		{input: "-1e309", expected: "-inf"},
		{input: "1.5", expected: "1.5"},
	}
	for _, tt := range tests {
		actual, err := formatPythonReprFloat(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, actual)
	}

	_, err = formatPythonReprFloat("1.2.3")
	require.ErrorContains(t, err, "invalid JSON number")
}

func TestWrite_RejectsMalformedNodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		node     *yaml.Node
		contains string
	}{
		{name: "nil node", node: nil, contains: "cannot write a nil node"},
		{name: "empty document", node: &yaml.Node{Kind: yaml.DocumentNode}, contains: "document must contain exactly one value"},
		{name: "unsupported node", node: &yaml.Node{Kind: yaml.AliasNode}, contains: "unsupported YAML node kind"},
		{name: "unmatched mapping key", node: &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{stringNode("key")}}, contains: "mapping has an unmatched key"},
		{name: "non-string mapping key", node: mappingNodeWithKey(intNode("1"), stringNode("value")), contains: "JSON object key must be a string"},
		{name: "nil sequence child", node: sequenceNode(nil), contains: "cannot write a nil node"},
		{name: "invalid integer", node: intNode("invalid"), contains: "invalid JSON integer"},
		{name: "invalid float", node: floatNode("1.2.3"), contains: "invalid JSON number"},
		{name: "unsupported scalar", node: &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!timestamp"}, contains: "unsupported scalar tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			require.ErrorContains(t, Write(tt.node, &output), tt.contains)
		})
	}
}

func TestPythonFloatAndExpansionAdditionalCases(t *testing.T) {
	t.Parallel()

	actual, err := pythonFloat(math.NaN())
	require.NoError(t, err)
	assert.Equal(t, "NaN", actual)

	tests := []struct {
		name     string
		mantissa string
		exponent int
		expected string
	}{
		{name: "negative before digits", mantissa: "-1.2", exponent: -2, expected: "-0.012"},
		{name: "negative after digits", mantissa: "-1.2", exponent: 3, expected: "-1200.0"},
		{name: "negative within digits", mantissa: "-1.23", exponent: 1, expected: "-12.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, expandFloat(tt.mantissa, tt.exponent))
		})
	}
}

var errTestIO = errors.New("test I/O error")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errTestIO
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func boolNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: value}
}

func intNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: value}
}

func floatNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: value}
}

func nullNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
}

func sequenceNode(values ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: values}
}

func mappingNode(key string, value *yaml.Node) *yaml.Node {
	return mappingNodeWithKey(stringNode(key), value)
}

func mappingNodeWithKey(key, value *yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{key, value}}
}

var _ io.Reader = failingReader{}
