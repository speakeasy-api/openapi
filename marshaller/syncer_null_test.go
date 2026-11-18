package marshaller_test

import (
	"context"
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

var errSyncerNullUnmarshal = errors.New("unmarshal failed")

// syncerNullFailing is a value whose unmarshalling always fails.
type syncerNullFailing struct{}

func (*syncerNullFailing) Unmarshal(context.Context, string, *yaml.Node) ([]error, error) {
	return nil, errSyncerNullUnmarshal
}

// syncerNullCore has optional scalar fields that can't represent being unset, and a field that can't be unmarshalled.
type syncerNullCore struct {
	marshaller.CoreModel `model:"syncerNullCore"`

	Label   marshaller.Node[string]             `key:"label" required:"false"`
	Count   marshaller.Node[int]                `key:"count" required:"false"`
	Failing marshaller.Node[*syncerNullFailing] `key:"failing"`
}

type syncerNullHigh struct {
	marshaller.Model[syncerNullCore]

	Label   string
	Count   int
	Failing *string
}

// syncerNullMismatchHigh has a count of a different kind than its core.
type syncerNullMismatchHigh struct {
	marshaller.Model[syncerNullCore]

	Label   string
	Count   string
	Failing *string
}

func syncerNullParse(t *testing.T, input string) *yaml.Node {
	t.Helper()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(input), &node), "yaml should parse")
	return node.Content[0]
}

func TestSync_AuthoredNullAlias_Success(t *testing.T) {
	t.Parallel()

	// intPtrField is an alias to the null anchored at stringPtrField.
	input := "stringField: s\nstringPtrField: &none null\nboolField: true\nintField: 1\nintPtrField: *none\nfloat64Field: 1.5\n"

	cases := []struct {
		name     string
		value    *string
		expected string
	}{
		{
			name:     "unset values keep the anchored null and the alias to it",
			value:    nil,
			expected: input,
		},
		{
			name:  "set anchored value replaces the null and drops the alias of the unset field",
			value: pointer.From("set"),
			// Setting the anchored value writes through to the node the alias shares, so the alias no longer
			// points at a null and the unset field is left out.
			expected: "stringField: s\nstringPtrField: &none set\nboolField: true\nintField: 1\nfloat64Field: 1.5\n",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var highModel tests.TestPrimitiveHighModel
			_, err := marshaller.UnmarshalNode(t.Context(), "", syncerNullParse(t, input), &highModel)
			require.NoError(t, err, "unmarshal should succeed")
			require.True(t, highModel.GetCore().IntPtrField.Present, "alias to a null should be present")

			highModel.StringPtrField = tt.value

			resultNode, err := marshaller.SyncValue(t.Context(), &highModel, highModel.GetCore(), highModel.GetRootNode(), false)
			require.NoError(t, err, "sync should succeed")

			actual, err := yaml.Marshal(resultNode)
			require.NoError(t, err, "yaml should marshal")
			assert.Equal(t, tt.expected, string(actual), "synced yaml should match")
			assert.Nil(t, highModel.GetCore().IntPtrField.Value, "the aliased field should stay unset")
		})
	}
}

func TestSync_OptionalZeroScalar_Success(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		model           *syncerNullHigh
		expected        string
		expectedPresent [2]bool
	}{
		{
			name:            "zero label is left out",
			model:           &syncerNullHigh{Count: 2},
			expected:        "count: 2\n",
			expectedPresent: [2]bool{false, true},
		},
		{
			name:            "zero count is left out",
			model:           &syncerNullHigh{Label: "l"},
			expected:        "label: l\n",
			expectedPresent: [2]bool{true, false},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core := &syncerNullCore{}
			resultNode, err := marshaller.SyncValue(t.Context(), tt.model, core, nil, false)
			require.NoError(t, err, "sync should succeed")

			actual, err := yaml.Marshal(resultNode)
			require.NoError(t, err, "yaml should marshal")
			assert.Equal(t, tt.expected, string(actual), "zero optional scalars should be left out")
			assert.Equal(t, tt.expectedPresent, [2]bool{core.Label.Present, core.Count.Present}, "only written fields should be present")
		})
	}
}

func TestSync_AuthoredNull_Error(t *testing.T) {
	t.Parallel()

	marshaller.RegisterType(func() *syncerNullFailing { return &syncerNullFailing{} })

	cases := []struct {
		name     string
		model    any
		input    string
		expected string
	}{
		{
			name:     "authored null that can't be unmarshalled",
			model:    &syncerNullHigh{Label: "l"},
			input:    "label: l\nfailing: null\n",
			expected: errSyncerNullUnmarshal.Error(),
		},
		{
			name:     "value of a different kind than its core",
			model:    &syncerNullMismatchHigh{Count: "2"},
			input:    "count: 1\n",
			expected: "SyncValue expected target to be string, got int",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marshaller.SyncValue(t.Context(), tt.model, &syncerNullCore{}, syncerNullParse(t, tt.input), false)
			require.Error(t, err, "sync should fail")
			assert.Equal(t, tt.expected, err.Error(), "sync error should match")
		})
	}
}
