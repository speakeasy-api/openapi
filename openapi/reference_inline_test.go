package openapi

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReference_InlinedSerialization_Success(t *testing.T) {
	t.Parallel()

	// Start with YAML that has a reference
	yamlWithRef := `$ref: '#/components/parameters/UserIdParam'`

	// Unmarshal the reference
	var ref ReferencedParameter
	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yamlWithRef), &ref)
	require.NoError(t, err)
	assert.Empty(t, validationErrs)

	// Verify it's a reference
	assert.True(t, ref.IsReference(), "Should be a reference after unmarshaling")
	assert.Equal(t, "#/components/parameters/UserIdParam", string(ref.GetReference()))

	// Create the object to inline
	param := &Parameter{
		Name:        "userId",
		In:          ParameterInPath,
		Required:    pointer.From(true),
		Description: pointer.From("User ID parameter"),
	}

	// Inline the reference by setting the object and clearing the reference
	ref.Object = param
	ref.Reference = nil

	// Marshal to YAML
	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), &ref, &buf)
	require.NoError(t, err)

	yamlStr := buf.String()

	// Expected YAML should only contain the object properties, not the $ref
	expectedYAML := `name: 'userId'
in: 'path'
description: 'User ID parameter'
required: true
`

	assert.Equal(t, expectedYAML, yamlStr, "Inlined reference should serialize only object properties")
}
