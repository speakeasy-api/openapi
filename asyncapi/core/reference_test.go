package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// failingCoreModel is a core model whose unmarshalling always fails.
type failingCoreModel struct {
	marshaller.CoreModel `model:"failingCoreModel"`
}

var errUnmarshalFailed = errors.New("unmarshal failed")

func (*failingCoreModel) Unmarshal(context.Context, string, *yaml.Node) ([]error, error) {
	return nil, errUnmarshalFailed
}

func parseYAML(t *testing.T, yml string) *yaml.Node {
	t.Helper()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node), "yaml should parse")
	return &node
}

func TestReference_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedRef   *string
		expectedValid bool
	}{
		{
			name:          "reference object",
			yaml:          "$ref: '#/components/servers/s'\n",
			expectedRef:   pointer.From("#/components/servers/s"),
			expectedValid: true,
		},
		{
			name:          "inline object",
			yaml:          "host: example.com\nprotocol: kafka\n",
			expectedRef:   nil,
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ref core.Reference[*core.Server]
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, tt.yaml), &ref)
			require.NoError(t, err, "unmarshal should succeed")
			assert.Empty(t, validationErrs, "unmarshal should not report validation errors")
			assert.Equal(t, tt.expectedRef, ref.Reference.Value, "reference should match")
			assert.Equal(t, tt.expectedValid, ref.GetValid(), "validity should match")
		})
	}
}

func TestReference_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	var ref core.Reference[*core.Server]
	_, err := ref.Unmarshal(t.Context(), "server", nil)
	require.Error(t, err, "unmarshalling a nil node should fail")
	assert.Equal(t, "node is nil", err.Error(), "error should describe the nil node")

	marshaller.RegisterType(func() *failingCoreModel { return &failingCoreModel{} })
	var failing core.Reference[*failingCoreModel]
	_, err = failing.Unmarshal(t.Context(), "failing", parseYAML(t, "a: b\n").Content[0])
	require.ErrorIs(t, err, errUnmarshalFailed, "inline object unmarshalling errors should be returned")
}

func TestReference_Unmarshal_TypeMismatch_Error(t *testing.T) {
	t.Parallel()

	var ref core.Reference[*core.Server]
	validationErrs, err := ref.Unmarshal(t.Context(), "server", parseYAML(t, "- a\n").Content[0])
	require.NoError(t, err, "unmarshal should succeed")
	require.Len(t, validationErrs, 1, "a non-object should report one validation error")

	var vErr *validation.Error
	require.ErrorAs(t, validationErrs[0], &vErr, "error should be a validation error")
	assert.Equal(t, validation.RuleValidationTypeMismatch, vErr.Rule, "error should be a type mismatch")
	assert.Equal(t, "[1:1] error validation-type-mismatch server reference expected `object`, got sequence", vErr.Error(), "error message should match")
	assert.False(t, ref.GetValid(), "reference should be invalid")
}

func TestReference_SyncChanges_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    any
		expected string
	}{
		{
			name:     "model is not a struct",
			model:    "server",
			expected: "Reference.SyncChanges expected a struct, got string",
		},
		{
			name: "inline object has the wrong kind",
			model: struct {
				Reference *string
				Object    int
			}{Object: 1},
			expected: "SyncValue expected target to be int, got struct",
		},
		{
			name: "reference model is not a pointer",
			model: struct {
				Reference *string
				Object    *int
			}{Reference: pointer.From("#/components/servers/s")},
			expected: "syncChanges expected source to be a pointer, got struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ref core.Reference[*core.Server]
			_, err := ref.SyncChanges(t.Context(), tt.model, nil)
			require.Error(t, err, "sync should fail")
			assert.Equal(t, tt.expected, err.Error(), "sync error should match")
		})
	}
}
