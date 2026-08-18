package sortfmt

import (
	"bytes"
	"io"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestFormat_MatchesSortedJSONStyle(t *testing.T) {
	t.Parallel()

	input := `{"z":"café \ud83d\ude00 <>&","components":{"schemas":{"Z":{"required":["z","a"],"oneOf":[{"type":"string"},{"$ref":"#/A"}],"properties":{"b":{"description":"b","type":"string"},"a":{"type":"number","default":1e7}}},"A":{}}},"openapi":"3.0.0","servers":[],"tags":["z","a"],"parameters":[{"name":"z"},{"$ref":"#/p"},{"name":"a"}],"duplicate":"first","duplicate":"last","negativeZero":-0.0,"small":1e-5}`
	expected := `{
  "openapi": "3.0.0",
  "servers": [],
  "components": {
    "schemas": {
      "A": {},
      "Z": {
        "properties": {
          "a": {
            "type": "number",
            "default": 10000000.0
          },
          "b": {
            "description": "b",
            "type": "string"
          }
        },
        "required": [
          "a",
          "z"
        ],
        "oneOf": [
          {
            "$ref": "#/A"
          },
          {
            "type": "string"
          }
        ]
      }
    }
  },
  "parameters": [
    {
      "$ref": "#/p"
    },
    {
      "name": "a"
    },
    {
      "name": "z"
    }
  ],
  "duplicate": "last",
  "negativeZero": -0.0,
  "small": 1e-05,
  "tags": [
    "z",
    "a"
  ],
  "z": "caf\u00e9 \ud83d\ude00 <>&"
}
`

	var output bytes.Buffer
	err := Format(strings.NewReader(input), &output)
	require.NoError(t, err, "format should succeed")
	assert.Equal(t, expected, output.String(), "output should match the sorted JSON style")
}

func TestFormat_IsIdempotentAndPermutationInvariant(t *testing.T) {
	t.Parallel()

	first := `{"openapi":"3.0.0","paths":{"/z":{},"/a":{}},"components":{"schemas":{"Z":{"required":["z","a"]},"A":{}}},"parameters":[{"name":"z"},{"name":"a"}],"tags":["z","a"]}`
	second := `{"tags":["z","a"],"parameters":[{"name":"a"},{"name":"z"}],"components":{"schemas":{"A":{},"Z":{"required":["a","z"]}}},"paths":{"/a":{},"/z":{}},"openapi":"3.0.0"}`

	firstOutput := formatForTest(t, first)
	secondOutput := formatForTest(t, second)
	assert.Equal(t, firstOutput, secondOutput, "key and selected-array permutations should converge")

	idempotentOutput := formatForTest(t, firstOutput)
	assert.Equal(t, firstOutput, idempotentOutput, "formatting an output twice should not change it")

	differentTags := strings.Replace(first, `"tags":["z","a"]`, `"tags":["a","z"]`, 1)
	differentTagsOutput := formatForTest(t, differentTags)
	assert.NotEqual(t, firstOutput, differentTagsOutput, "non-selected array order should be preserved")
}

func TestFormat_InvalidJSONReturnsError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := Format(strings.NewReader(`{"openapi":}`), &output)
	require.Error(t, err, "invalid JSON should fail")
}

func TestFormat_ShortWriteReturnsError(t *testing.T) {
	t.Parallel()

	err := Format(strings.NewReader(`{"openapi":"3.0.0"}`), shortWriter{})
	require.ErrorIs(t, err, io.ErrShortWrite, "short writes should be reported")
}

func TestFormat_MatchesPythonNumberRangeSemantics(t *testing.T) {
	t.Parallel()

	input := `{"positiveOverflow":1e309,"negativeOverflow":-1e309,"underflow":1e-400}`
	expected := `{
  "negativeOverflow": -Infinity,
  "positiveOverflow": Infinity,
  "underflow": 0.0
}
`

	var output bytes.Buffer
	err := Format(strings.NewReader(input), &output)
	require.NoError(t, err, "range-limited floats should use Python JSON semantics")
	assert.Equal(t, expected, output.String(), "range-limited floats should match Python output")
}

func TestFormat_SortsArrayValuedTypeKeys(t *testing.T) {
	t.Parallel()

	input := `{"oneOf":[{"type":["string","null"]},{"type":["integer","null"]}],"anyOf":[{"type":["string","null"]}]}`
	expected := `{
  "anyOf": [
    {
      "type": [
        "string",
        "null"
      ]
    }
  ],
  "oneOf": [
    {
      "type": [
        "integer",
        "null"
      ]
    },
    {
      "type": [
        "string",
        "null"
      ]
    }
  ]
}
`

	var output bytes.Buffer
	err := Format(strings.NewReader(input), &output)
	require.NoError(t, err, "OpenAPI 3.1 array-valued types should sort")
	assert.Equal(t, expected, output.String(), "array-valued types should use Python list ordering")
}

func TestFormat_MixedSortKeyKindsMatchReferenceError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := Format(
		strings.NewReader(`{"oneOf":[{"type":"string"},{"type":["string","null"]}]}`),
		&output,
	)
	require.ErrorContains(t, err, "sort oneOf items", "mixed Python sort-key types should fail")
	assert.Empty(t, output.String(), "sorting errors should not produce partial output")
}

func TestFormat_RejectsLoneSurrogateEscapes(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`{"a":"\ud83d"}`,
		`{"a":"\ude00"}`,
		`{"a":"\ude00\ud83d"}`,
		`{"\ud83d":1}`,
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			err := Format(strings.NewReader(input), &output)
			require.ErrorContains(t, err, "lone surrogate escape")
			assert.Empty(t, output.String(), "parse errors should not produce output")
		})
	}
}

func TestFormat_AcceptsPairedAndEscapedSurrogates(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`{"a":"\ud83d\ude00"}`,
		`{"a":"\uD83D\uDE00"}`,
		`{"a":"C:\\ud83d"}`,
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			require.NoError(t, Format(strings.NewReader(input), &output))
		})
	}
}

func TestFormat_RejectsInvalidUTF8(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := Format(bytes.NewReader([]byte{'{', '"', 'a', '"', ':', '"', 0xff, '"', '}'}), &output)
	require.ErrorContains(t, err, "input is not valid UTF-8")
}

func TestFormat_MatchesPythonFallbackSortKeys(t *testing.T) {
	t.Parallel()

	input := `{"oneOf":[{"maximum":1e309},{"maximum":true}]}`
	expected := `{
  "oneOf": [
    {
      "maximum": true
    },
    {
      "maximum": Infinity
    }
  ]
}
`

	var output bytes.Buffer
	err := Format(strings.NewReader(input), &output)
	require.NoError(t, err, "fallback keys should use Python representations")
	assert.Equal(t, expected, output.String(), "non-finite fallback keys should use lowercase Python repr ordering")
}

func TestFormat_EdgeCaseGolden(t *testing.T) {
	t.Parallel()

	input, err := os.Open("testdata/edge_cases.json")
	require.NoError(t, err, "edge-case input should be readable")
	defer input.Close()

	expected, err := os.ReadFile("testdata/edge_cases.expected.json")
	require.NoError(t, err, "edge-case golden should be readable")

	var actual bytes.Buffer
	err = Format(input, &actual)
	require.NoError(t, err, "edge-case formatting should succeed")
	assert.Equal(t, expected, actual.Bytes(), "edge-case output should match the reference golden")

	var idempotent bytes.Buffer
	err = Format(bytes.NewReader(expected), &idempotent)
	require.NoError(t, err, "reference-formatted output should be accepted")
	assert.Equal(t, expected, idempotent.Bytes(), "reference-formatted output should be idempotent")
}

func TestPythonFloat_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{name: "positive zero", value: 0, expected: "0.0"},
		{name: "fixed upper threshold", value: 1e15, expected: "1000000000000000.0"},
		{name: "scientific upper threshold", value: 1e16, expected: "1e+16"},
		{name: "fixed lower threshold", value: 1e-4, expected: "0.0001"},
		{name: "scientific lower threshold", value: 1e-5, expected: "1e-05"},
		{name: "expanded integer", value: 1e7, expected: "10000000.0"},
		{name: "fraction", value: 1.25, expected: "1.25"},
		{name: "positive infinity", value: math.Inf(1), expected: "Infinity"},
		{name: "negative infinity", value: math.Inf(-1), expected: "-Infinity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := pythonFloat(tt.value)
			require.NoError(t, err, "float formatting should succeed")
			assert.Equal(t, tt.expected, actual, "float should use Python repr formatting")
		})
	}
}

func TestWrite_BooleanSpellings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value    string
		expected string
	}{
		{value: "true", expected: "true\n"},
		{value: "True", expected: "true\n"},
		{value: "TRUE", expected: "true\n"},
		{value: "false", expected: "false\n"},
		{value: "False", expected: "false\n"},
		{value: "FALSE", expected: "false\n"},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			err := Write(&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: tt.value}, &output)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, output.String())
		})
	}
}

func TestWrite_InvalidBooleanReturnsError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := Write(&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "yes"}, &output)
	require.ErrorContains(t, err, "invalid boolean value")
}

func TestPythonRepr_BooleanSpellings(t *testing.T) {
	t.Parallel()

	trueValue, err := pythonRepr(&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "TRUE"})
	require.NoError(t, err)
	assert.Equal(t, "True", trueValue)

	falseValue, err := pythonRepr(&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "False"})
	require.NoError(t, err)
	assert.Equal(t, "False", falseValue)
}

func TestSort_NilNodesReturnErrors(t *testing.T) {
	t.Parallel()

	stringNode := func(value string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	}
	tests := map[string]*yaml.Node{
		"root":           nil,
		"mapping key":    {Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{nil, stringNode("value")}},
		"mapping value":  {Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{stringNode("key"), nil}},
		"sequence child": {Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{nil}},
	}
	for name, node := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				require.Error(t, Sort(node))
			})
		})
	}
}

func TestWrite_NilMappingEntriesReturnErrors(t *testing.T) {
	t.Parallel()

	stringNode := func(value string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	}
	tests := map[string]*yaml.Node{
		"key":   {Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{nil, stringNode("value")}},
		"value": {Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{stringNode("key"), nil}},
	}
	for name, node := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				var output bytes.Buffer
				require.Error(t, Write(node, &output))
			})
		})
	}
}

func formatForTest(t *testing.T, input string) string {
	t.Helper()

	var output bytes.Buffer
	err := Format(strings.NewReader(input), &output)
	require.NoError(t, err, "format should succeed")
	return output.String()
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}
