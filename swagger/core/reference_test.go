package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestReference_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		isRef    bool
		expected string
	}{
		{
			name: "unmarshals reference with $ref",
			yaml: `
$ref: '#/definitions/Pet'
`,
			isRef:    true,
			expected: "#/definitions/Pet",
		},
		{
			name: "unmarshals inlined parameter object",
			yaml: `
name: petId
in: path
type: string
required: true
`,
			isRef: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var ref Reference[*Parameter]
			_, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &ref)
			require.NoError(t, err, "unmarshal should succeed")

			if tt.isRef {
				require.NotNil(t, ref.Reference.Value, "should have reference value")
				assert.Equal(t, tt.expected, *ref.Reference.Value, "should have correct reference")
			} else {
				require.NotNil(t, ref.Object, "should have inlined object")
			}
		})
	}
}

func TestReference_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	t.Run("returns error when node is nil", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		var ref Reference[*Parameter]
		_, err := ref.Unmarshal(ctx, "", nil)
		require.Error(t, err, "should return error for nil node")
		assert.Contains(t, err.Error(), "node is nil", "error should mention nil node")
	})

	t.Run("returns validation error for non-object node", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		var ref Reference[*Parameter]
		node := &yaml.Node{Kind: yaml.ScalarNode, Value: "just a string"}
		validationErrs, err := ref.Unmarshal(ctx, "test", node)
		require.NoError(t, err, "should not return fatal error")
		require.NotEmpty(t, validationErrs, "should have validation errors")
		assert.False(t, ref.GetValid(), "should not be valid")
	})
}

func TestReference_SyncChanges_ErrorCases(t *testing.T) {
	t.Parallel()

	t.Run("errors on non-struct model", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		coreRef := &Reference[*Parameter]{}
		coreRef.SetValid(true, true)

		_, err := coreRef.SyncChanges(ctx, "not a struct", nil)
		require.Error(t, err, "should return error for non-struct model")
		assert.Contains(t, err.Error(), "expected a struct", "error should mention struct expectation")
	})

	t.Run("errors on non-pointer non-struct model", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		coreRef := &Reference[*Parameter]{}
		coreRef.SetValid(true, true)

		_, err := coreRef.SyncChanges(ctx, 42, nil)
		require.Error(t, err, "should return error for int model")
		assert.Contains(t, err.Error(), "expected a struct", "error should mention struct expectation")
	})
}
