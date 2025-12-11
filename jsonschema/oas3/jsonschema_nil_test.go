package oas3_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONSchema_IsSchema_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	assert.False(t, js.IsSchema(), "nil JSONSchema should return false for IsSchema")
}

func TestJSONSchema_GetSchema_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	assert.Nil(t, js.GetSchema(), "nil JSONSchema should return nil for GetSchema")
}

func TestJSONSchema_IsBool_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	assert.False(t, js.IsBool(), "nil JSONSchema should return false for IsBool")
}

func TestJSONSchema_GetBool_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	assert.Nil(t, js.GetBool(), "nil JSONSchema should return nil for GetBool")
}

func TestJSONSchema_GetParent_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	assert.Nil(t, js.GetParent(), "nil JSONSchema should return nil for GetParent")
}

func TestJSONSchema_GetTopLevelParent_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	assert.Nil(t, js.GetTopLevelParent(), "nil JSONSchema should return nil for GetTopLevelParent")
}

func TestJSONSchema_SetParent_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	// Should not panic
	js.SetParent(nil)
}

func TestJSONSchema_SetTopLevelParent_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	// Should not panic
	js.SetTopLevelParent(nil)
}

func TestJSONSchema_IsEqual_BothNil(t *testing.T) {
	t.Parallel()

	var js1 *oas3.JSONSchema[oas3.Referenceable]
	var js2 *oas3.JSONSchema[oas3.Referenceable]
	assert.True(t, js1.IsEqual(js2), "two nil JSONSchemas should be equal")
}

func TestJSONSchema_IsEqual_OneNil(t *testing.T) {
	t.Parallel()

	js1 := oas3.NewJSONSchemaFromBool(true)
	var js2 *oas3.JSONSchema[oas3.Referenceable]
	assert.False(t, js1.IsEqual(js2), "nil and non-nil JSONSchemas should not be equal")
	assert.False(t, js2.IsEqual(js1), "nil and non-nil JSONSchemas should not be equal")
}

func TestJSONSchema_Validate_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	errs := js.Validate(t.Context())
	assert.Empty(t, errs, "nil JSONSchema should return empty errors for Validate")
}

func TestJSONSchema_ShallowCopy_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	schemaCopy := js.ShallowCopy()
	assert.Nil(t, schemaCopy, "nil JSONSchema should return nil for ShallowCopy")
}

func TestJSONSchema_IsResolved_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	assert.False(t, js.IsResolved(), "nil JSONSchema should return false for IsResolved")
}

func TestJSONSchema_IsReference_Success(t *testing.T) {
	t.Parallel()

	// Non-reference schema
	js := oas3.NewJSONSchemaFromBool(true)
	assert.False(t, js.IsReference(), "boolean JSONSchema should not be a reference")

	// Schema with no ref
	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Title: pointer.From("test"),
	})
	assert.False(t, schema.IsReference(), "schema without ref should not be a reference")
}

func TestJSONSchema_GetExtensions_NilConcrete(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Concrete]
	exts := js.GetExtensions()
	require.NotNil(t, exts, "nil JSONSchema should return empty extensions")
}

func TestJSONSchema_GetReference_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	ref := js.GetReference()
	assert.Empty(t, ref, "nil JSONSchema should return empty for GetReference")
}

func TestJSONSchema_GetRef_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	ref := js.GetRef()
	assert.Empty(t, ref, "nil JSONSchema should return empty for GetRef")
}

func TestJSONSchema_GetResolvedObject_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	resolved := js.GetResolvedObject()
	assert.Nil(t, resolved, "nil JSONSchema should return nil for GetResolvedObject")
}

func TestJSONSchema_MustGetResolvedSchema_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	resolved := js.MustGetResolvedSchema()
	assert.Nil(t, resolved, "nil JSONSchema should return nil for MustGetResolvedSchema")
}

func TestJSONSchema_GetAbsRef_Nil(t *testing.T) {
	t.Parallel()

	var js *oas3.JSONSchema[oas3.Referenceable]
	ref := js.GetAbsRef()
	assert.Empty(t, ref, "nil JSONSchema should return empty for GetAbsRef")
}

func TestValidate_NilSchema_Success(t *testing.T) {
	t.Parallel()

	// Test with nil Referenceable schema
	var jsReferenceable *oas3.JSONSchema[oas3.Referenceable]
	errs := oas3.Validate(t.Context(), jsReferenceable)
	assert.Empty(t, errs, "Validate on nil Referenceable schema should return empty errors")

	// Test with nil Concrete schema
	var jsConcrete *oas3.JSONSchema[oas3.Concrete]
	errs = oas3.Validate(t.Context(), jsConcrete)
	assert.Empty(t, errs, "Validate on nil Concrete schema should return empty errors")
}

func TestValidate_BoolSchema_Success(t *testing.T) {
	t.Parallel()

	// Test with bool schema (not a schema, so returns nil)
	js := oas3.NewJSONSchemaFromBool(true)
	errs := oas3.Validate(t.Context(), js)
	assert.Empty(t, errs, "Validate on bool schema should return empty errors")
}
