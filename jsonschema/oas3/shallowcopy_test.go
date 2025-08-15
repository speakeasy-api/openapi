package oas3_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONSchema_ShallowCopy_Success(t *testing.T) {
	t.Parallel()

	// Create a JSONSchema with properties
	original := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Type: oas3.NewTypeFromString("object"),
		Properties: sequencedmap.New(
			sequencedmap.NewElem("name", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
				Type: oas3.NewTypeFromString("string"),
			})),
		),
	})

	// Create a shallow copy
	copied := original.ShallowCopy()
	require.NotNil(t, copied, "shallow copy should not be nil")

	// Initially they should be equal
	assert.True(t, original.IsEqual(copied), "original and copy should be equal initially")

	// Modify the copy by adding a new property
	copied.GetLeft().Properties.Set("email", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Type: oas3.NewTypeFromString("string"),
	}))

	// Now they should not be equal
	assert.False(t, original.IsEqual(copied), "original and copy should not be equal after modification")
}
