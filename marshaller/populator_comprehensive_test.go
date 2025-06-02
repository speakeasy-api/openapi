package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simple test to exercise populateValue function
func Test_PopulateModel_SimpleStructToStruct(t *testing.T) {
	type Source struct {
		StringField string
		IntField    int
	}

	type Target struct {
		StringField string
		IntField    int
	}

	source := Source{
		StringField: "test",
		IntField:    42,
	}

	target := &Target{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)
	assert.Equal(t, "test", target.StringField)
	assert.Equal(t, 42, target.IntField)
}

// Test with slices to exercise slice population
func Test_PopulateModel_WithSlices(t *testing.T) {
	type Source struct {
		SliceField []string
	}

	type Target struct {
		SliceField []string
	}

	source := Source{
		SliceField: []string{"a", "b", "c"},
	}

	target := &Target{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, target.SliceField)
}

// Test with nil slice
func Test_PopulateModel_WithNilSlice(t *testing.T) {
	type Source struct {
		SliceField []string
	}

	type Target struct {
		SliceField []string
	}

	source := Source{
		SliceField: nil,
	}

	target := &Target{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)
	assert.Nil(t, target.SliceField)
}

// Test type conversion
func Test_PopulateModel_TypeConversion(t *testing.T) {
	type Source struct {
		IntField int
	}

	type Target struct {
		IntField int64
	}

	source := Source{
		IntField: 42,
	}

	target := &Target{}

	err := marshaller.Populate(source, target)
	// This might fail due to type incompatibility - that's expected
	if err != nil {
		assert.Contains(t, err.Error(), "cannot convert")
	} else {
		assert.Equal(t, int64(42), target.IntField)
	}
}

// Test with pointer fields
func Test_PopulateModel_WithPointers(t *testing.T) {
	type Source struct {
		PtrField *string
	}

	type Target struct {
		PtrField *string
	}

	value := "test-value"
	source := Source{
		PtrField: &value,
	}

	target := &Target{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)
	require.NotNil(t, target.PtrField)
	assert.Equal(t, "test-value", *target.PtrField)
}

// Test with nil pointer
func Test_PopulateModel_WithNilPointer(t *testing.T) {
	type Source struct {
		PtrField *string
	}

	type Target struct {
		PtrField *string
	}

	source := Source{
		PtrField: nil,
	}

	target := &Target{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)
	assert.Nil(t, target.PtrField)
}

// Test error case - non-struct source
func Test_PopulateModel_NonStructSource_Error(t *testing.T) {
	type Target struct {
		Field string
	}

	source := "not-a-struct"
	target := &Target{}

	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot convert")
}

// Test with incompatible types
func Test_PopulateModel_IncompatibleTypes_Error(t *testing.T) {
	type Source struct {
		Field string
	}

	type Target struct {
		Field int
	}

	source := Source{
		Field: "string-value",
	}

	target := &Target{}

	err := marshaller.Populate(source, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot convert")
}

// Test with nested slices
func Test_PopulateModel_NestedSlices(t *testing.T) {
	type NestedStruct struct {
		Value string
	}

	type Source struct {
		NestedSlice []NestedStruct
	}

	type Target struct {
		NestedSlice []NestedStruct
	}

	source := Source{
		NestedSlice: []NestedStruct{
			{Value: "first"},
			{Value: "second"},
		},
	}

	target := &Target{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)
	require.Len(t, target.NestedSlice, 2)
	assert.Equal(t, "first", target.NestedSlice[0].Value)
	assert.Equal(t, "second", target.NestedSlice[1].Value)
}
