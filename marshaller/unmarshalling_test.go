package marshaller_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/marshaller/tests/core"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestUnmarshal_PrimitiveTypes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		expected func(*core.TestPrimitiveModel)
	}{
		{
			name: "all primitive fields set",
			yml: `
stringField: "test string"
stringPtrField: "test ptr string"
boolField: true
boolPtrField: false
intField: 42
intPtrField: 24
float64Field: 3.14
float64PtrField: 2.71
x-custom: "extension value"
`,
			expected: func(model *core.TestPrimitiveModel) {
				require.Equal(t, "test string", model.StringField.Value)
				require.True(t, model.StringField.Present)
				require.Equal(t, "test ptr string", *model.StringPtrField.Value)
				require.True(t, model.StringPtrField.Present)
				require.True(t, model.BoolField.Value)
				require.True(t, model.BoolField.Present)
				require.False(t, *model.BoolPtrField.Value)
				require.True(t, model.BoolPtrField.Present)
				require.Equal(t, 42, model.IntField.Value)
				require.True(t, model.IntField.Present)
				require.Equal(t, 24, *model.IntPtrField.Value)
				require.True(t, model.IntPtrField.Present)
				require.InDelta(t, 3.14, model.Float64Field.Value, 0.001)
				require.True(t, model.Float64Field.Present)
				require.InDelta(t, 2.71, *model.Float64PtrField.Value, 0.001)
				require.True(t, model.Float64PtrField.Present)

				// Check extensions
				require.NotNil(t, model.Extensions)
				ext, ok := model.Extensions.Get("x-custom")
				require.True(t, ok)
				require.NotNil(t, ext.Value)
				// Extensions store raw yaml.Node values
				var extValue string
				err := ext.Value.Decode(&extValue)
				require.NoError(t, err)
				require.Equal(t, "extension value", extValue)
			},
		},
		{
			name: "only required fields set",
			yml: `
stringField: "required only"
boolField: true
intField: 42
float64Field: 3.14
`,
			expected: func(model *core.TestPrimitiveModel) {
				require.Equal(t, "required only", model.StringField.Value)
				require.True(t, model.StringField.Present)
				require.True(t, model.BoolField.Value)
				require.True(t, model.BoolField.Present)
				require.Equal(t, 42, model.IntField.Value)
				require.True(t, model.IntField.Present)
				require.InDelta(t, 3.14, model.Float64Field.Value, 0.001)
				require.True(t, model.Float64Field.Present)

				// Optional fields should not be present
				require.False(t, model.StringPtrField.Present)
				require.False(t, model.BoolPtrField.Present)
				require.False(t, model.IntPtrField.Present)
				require.False(t, model.Float64PtrField.Present)
			},
		},
		{
			name: "null pointer fields",
			yml: `
stringField: "test"
boolField: true
intField: 42
float64Field: 3.14
stringPtrField: null
boolPtrField: null
intPtrField: null
float64PtrField: null
`,
			expected: func(model *core.TestPrimitiveModel) {
				require.Equal(t, "test", model.StringField.Value)
				require.True(t, model.StringField.Present)
				require.True(t, model.BoolField.Value)
				require.True(t, model.BoolField.Present)
				require.Equal(t, 42, model.IntField.Value)
				require.True(t, model.IntField.Present)
				require.InDelta(t, 3.14, model.Float64Field.Value, 0.001)
				require.True(t, model.Float64Field.Present)

				// Null pointer fields should be present but with nil values
				require.True(t, model.StringPtrField.Present)
				require.Nil(t, model.StringPtrField.Value)
				require.True(t, model.BoolPtrField.Present)
				require.Nil(t, model.BoolPtrField.Value)
				require.True(t, model.IntPtrField.Present)
				require.Nil(t, model.IntPtrField.Value)
				require.True(t, model.Float64PtrField.Present)
				require.Nil(t, model.Float64PtrField.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var model core.TestPrimitiveModel
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, tt.yml), &model)
			require.NoError(t, err)
			require.Empty(t, validationErrs)
			require.True(t, model.Valid)
			require.True(t, model.ValidYaml)

			tt.expected(&model)
		})
	}
}

func TestUnmarshal_PrimitiveTypes_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing required fields",
			yml: `
stringPtrField: "optional field"
`,
			wantErrs: []string{
				"[2:1] error validation-required-field `testPrimitiveModel.boolField` is required",
				"[2:1] error validation-required-field `testPrimitiveModel.float64Field` is required",
				"[2:1] error validation-required-field `testPrimitiveModel.intField` is required",
				"[2:1] error validation-required-field `testPrimitiveModel.stringField` is required",
			},
		},
		{
			name: "type mismatch - string field gets array",
			yml: `
stringField: ["not", "a", "string"]
boolField: true
intField: 42
float64Field: 3.14
`,
			wantErrs: []string{"[2:14] error validation-type-mismatch testPrimitiveModel.stringField expected `string`, got `sequence`"},
		},
		{
			name: "type mismatch - bool field gets string",
			yml: `
stringField: "test"
boolField: "not a bool"
intField: 42
float64Field: 3.14
`,
			wantErrs: []string{"[3:12] error validation-type-mismatch testPrimitiveModel.boolField line 3: cannot construct !!str `not a bool` into bool"},
		},
		{
			name: "type mismatch - int field gets string",
			yml: `
stringField: "test"
boolField: true
intField: "not an int"
float64Field: 3.14
`,
			wantErrs: []string{"[4:11] error validation-type-mismatch testPrimitiveModel.intField line 4: cannot construct !!str `not an int` into int"},
		},
		{
			name: "type mismatch - float field gets string",
			yml: `
stringField: "test"
boolField: true
intField: 42
float64Field: "not a float"
`,
			wantErrs: []string{"[5:15] error validation-type-mismatch testPrimitiveModel.float64Field line 5: cannot construct !!str `not a f...` into float64"},
		},
		{
			name: "multiple validation errors",
			yml: `
boolField: "not a bool"
intField: "not an int"
`,
			wantErrs: []string{
				"[2:1] error validation-required-field `testPrimitiveModel.float64Field` is required",
				"[2:1] error validation-required-field `testPrimitiveModel.stringField` is required",
				"[2:12] error validation-type-mismatch testPrimitiveModel.boolField line 2: cannot construct !!str `not a bool` into bool",
				"[3:11] error validation-type-mismatch testPrimitiveModel.intField line 3: cannot construct !!str `not an int` into int",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var model core.TestPrimitiveModel
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, tt.yml), &model)
			require.NoError(t, err)
			require.NotEmpty(t, validationErrs)
			validation.SortValidationErrors(validationErrs)

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range validationErrs {
				errMessages = append(errMessages, err.Error())
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}

func TestUnmarshal_CoreModelStructs_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		expected func(*core.TestComplexModel)
	}{
		{
			name: "nested core model",
			yml: `
nestedModelValue:
  stringField: "required value"
  boolField: true
  intField: 42
  float64Field: 3.14
nestedModel:
  stringField: "nested value"
  boolField: true
  intField: 100
  float64Field: 3.14
arrayField:
  - "item1"
  - "item2"
  - "item3"
eitherModelOrPrimitive:
  stringField: "either model value"
  boolField: true
  intField: 123
  float64Field: 1.23
`,
			expected: func(model *core.TestComplexModel) {
				require.True(t, model.NestedModel.Present)
				require.NotNil(t, model.NestedModel.Value)
				require.Equal(t, "nested value", model.NestedModel.Value.StringField.Value)
				require.Equal(t, 100, model.NestedModel.Value.IntField.Value)

				require.True(t, model.ArrayField.Present)
				require.Len(t, model.ArrayField.Value, 3)
				require.Equal(t, "item1", model.ArrayField.Value[0])
				require.Equal(t, "item2", model.ArrayField.Value[1])
				require.Equal(t, "item3", model.ArrayField.Value[2])
			},
		},
		{
			name: "nested model value (not pointer)",
			yml: `
nestedModelValue:
  stringField: "value model"
  boolField: true
  intField: 42
  float64Field: 2.71
eitherModelOrPrimitive: 456
`,
			expected: func(model *core.TestComplexModel) {
				require.True(t, model.NestedModelValue.Present)
				require.Equal(t, "value model", model.NestedModelValue.Value.StringField.Value)
				require.True(t, model.NestedModelValue.Value.BoolField.Value)
			},
		},
		{
			name: "node array field",
			yml: `
nestedModelValue:
  stringField: "required value"
  boolField: true
  intField: 42
  float64Field: 3.14
nodeArrayField:
  - "node1"
  - "node2"
eitherModelOrPrimitive:
  stringField: "array test model"
  boolField: false
  intField: 789
  float64Field: 9.87
`,
			expected: func(model *core.TestComplexModel) {
				require.True(t, model.NodeArrayField.Present)
				require.Len(t, model.NodeArrayField.Value, 2)
				require.Equal(t, "node1", model.NodeArrayField.Value[0].Value)
				require.True(t, model.NodeArrayField.Value[0].Present)
				require.Equal(t, "node2", model.NodeArrayField.Value[1].Value)
				require.True(t, model.NodeArrayField.Value[1].Present)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var model core.TestComplexModel
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, tt.yml), &model)
			require.NoError(t, err)
			require.Empty(t, validationErrs)
			require.True(t, model.Valid)
			require.True(t, model.ValidYaml)

			tt.expected(&model)
		})
	}
}

func TestUnmarshal_CoreModelStructs_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "nested model validation error",
			yml: `
nestedModelValue:
  stringField: "required value"
  boolField: true
  intField: 42
  float64Field: 3.14
nestedModel:
  intField: 100
  # missing required stringField, boolField, float64Field
`,
			wantErrs: []string{
				"[8:3] error validation-required-field `testPrimitiveModel.stringField` is required",
				"[8:3] error validation-required-field `testPrimitiveModel.boolField` is required",
				"[8:3] error validation-required-field `testPrimitiveModel.float64Field` is required",
			},
		},
		{
			name: "type mismatch - object field gets array",
			yml: `
nestedModelValue:
  stringField: "required value"
  boolField: true
  intField: 42
  float64Field: 3.14
nestedModel:
  - "this should be an object"
`,
			wantErrs: []string{"[8:3] error validation-type-mismatch testComplexModel.nestedModel expected `object`, got `sequence`"},
		},
		{
			name: "type mismatch - array field gets object",
			yml: `
nestedModelValue:
  stringField: "required value"
  boolField: true
  intField: 42
  float64Field: 3.14
arrayField:
  key: "this should be an array"
`,
			wantErrs: []string{"[8:3] error validation-type-mismatch testComplexModel.arrayField expected `sequence`, got `object`"},
		},
		{
			name: "deeply nested validation error",
			yml: `
nestedModelValue:
  stringField: "required value"
  boolField: true
  intField: 42
  float64Field: 3.14
structArrayField:
  - stringField: "valid"
    boolField: true
    intField: 100
    float64Field: 1.23
  - intField: 42
    boolField: true
    float64Field: 4.56
    # missing required stringField in second element
`,
			wantErrs: []string{"[12:5] error validation-required-field `testPrimitiveModel.stringField` is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var model core.TestComplexModel
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, tt.yml), &model)
			require.NoError(t, err)
			require.NotEmpty(t, validationErrs)

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range validationErrs {
				errMessages = append(errMessages, err.Error())
			}

			for _, expectedErr := range tt.wantErrs {
				found := false
				for _, errMsg := range errMessages {
					if strings.Contains(errMsg, expectedErr) {
						found = true
						break
					}
				}
				require.True(t, found, "expected error message '%s' not found in: %v", expectedErr, errMessages)
			}
		})
	}
}

func TestUnmarshal_NonCoreModel_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: "test name"
value: 42
description: "test description"
`

	var model core.TestNonCoreModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "test name", model.Name)
	require.Equal(t, 42, model.Value)
	require.NotNil(t, model.Description)
	require.Equal(t, "test description", *model.Description)
}

func TestUnmarshal_CustomUnmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
customField: "custom value"
x-extension: "ext value"
`

	var model core.TestCustomUnmarshalModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, model.Valid)
	require.True(t, model.ValidYaml)

	// Check that custom unmarshal was called
	require.True(t, model.UnmarshalCalled)
	require.Equal(t, "custom value", model.CustomField.Value)
	require.True(t, model.CustomField.Present)

	// Check extensions
	ext, ok := model.Extensions.Get("x-extension")
	require.True(t, ok)
	require.NotNil(t, ext.Value)
	// Extensions store raw yaml.Node values
	var extValue string
	err = ext.Value.Decode(&extValue)
	require.NoError(t, err)
	require.Equal(t, "ext value", extValue)
}

func TestUnmarshal_Aliases_Success(t *testing.T) {
	t.Parallel()

	yml := `
aliasField: &alias "aliased value"
aliasArray:
  - *alias
  - "regular value"
aliasStruct: &structAlias
  stringField: "struct value"
  boolField: true
  intField: 42
  float64Field: 3.14
x-alias-ext: *alias
`

	var model core.TestAliasModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, model.Valid)
	require.True(t, model.ValidYaml)

	// Check alias resolution
	require.Equal(t, "aliased value", model.AliasField.Value)
	require.True(t, model.AliasField.Present)

	require.True(t, model.AliasArray.Present)
	require.Len(t, model.AliasArray.Value, 2)
	require.Equal(t, "aliased value", model.AliasArray.Value[0])
	require.Equal(t, "regular value", model.AliasArray.Value[1])

	require.True(t, model.AliasStruct.Present)
	require.NotNil(t, model.AliasStruct.Value)
	require.Equal(t, "struct value", model.AliasStruct.Value.StringField.Value)
	require.Equal(t, 42, model.AliasStruct.Value.IntField.Value)

	// Check extension alias
	ext, ok := model.Extensions.Get("x-alias-ext")
	require.True(t, ok)
	require.NotNil(t, ext.Value)
	// Extensions store raw yaml.Node values
	var extValue string
	err = ext.Value.Decode(&extValue)
	require.NoError(t, err)
	require.Equal(t, "aliased value", extValue)
}

func TestUnmarshal_EmbeddedMap_Success(t *testing.T) {
	t.Parallel()

	yml := `
dynamicKey1: "value1"
dynamicKey2: "value2"
`

	var model core.TestEmbeddedMapModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, model.Valid)
	require.True(t, model.ValidYaml)

	// Check embedded map values
	require.NotNil(t, model.Map)
	val1, ok := model.Get("dynamicKey1")
	require.True(t, ok)
	require.Equal(t, "value1", val1.Value)
	require.True(t, val1.Present)

	val2, ok := model.Get("dynamicKey2")
	require.True(t, ok)
	require.Equal(t, "value2", val2.Value)
	require.True(t, val2.Present)
}

func TestUnmarshal_EmbeddedMapWithFields_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: "test name"
dynamicKey1:
  stringField: "dynamic value 1"
  boolField: true
  intField: 100
  float64Field: 1.23
dynamicKey2:
  stringField: "dynamic value 2"
  boolField: false
  intField: 42
  float64Field: 4.56
x-extension: "ext value"
`

	var model core.TestEmbeddedMapWithFieldsModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, model.Valid)
	require.True(t, model.ValidYaml)

	// Check regular field
	require.Equal(t, "test name", model.NameField.Value)
	require.True(t, model.NameField.Present)

	// Check embedded map values
	require.NotNil(t, model.Map)
	val1, ok := model.Get("dynamicKey1")
	require.True(t, ok)
	require.NotNil(t, val1.Value)
	require.Equal(t, "dynamic value 1", val1.Value.StringField.Value)

	val2, ok := model.Get("dynamicKey2")
	require.True(t, ok)
	require.NotNil(t, val2.Value)
	require.Equal(t, "dynamic value 2", val2.Value.StringField.Value)
	require.Equal(t, 42, val2.Value.IntField.Value)

	// Check extensions
	ext, ok := model.Extensions.Get("x-extension")
	require.True(t, ok)
	require.NotNil(t, ext.Value)
	// Extensions store raw yaml.Node values
	var extValue string
	err = ext.Value.Decode(&extValue)
	require.NoError(t, err)
	require.Equal(t, "ext value", extValue)
}

func TestUnmarshal_RequiredPointer_Success(t *testing.T) {
	t.Parallel()

	yml := `
requiredPtr: "required pointer value"
optionalPtr: "optional pointer value"
`

	var model core.TestRequiredPointerModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, model.Valid)
	require.True(t, model.ValidYaml)

	// Check required pointer field
	require.True(t, model.RequiredPtr.Present)
	require.NotNil(t, model.RequiredPtr.Value)
	require.Equal(t, "required pointer value", *model.RequiredPtr.Value)

	// Check optional pointer field
	require.True(t, model.OptionalPtr.Present)
	require.NotNil(t, model.OptionalPtr.Value)
	require.Equal(t, "optional pointer value", *model.OptionalPtr.Value)
}

func TestUnmarshal_RequiredPointer_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing required pointer field",
			yml: `
optionalPtr: "only optional set"
`,
			wantErrs: []string{"[2:1] error validation-required-field `testRequiredPointerModel.requiredPtr` is required"},
		},
		{
			name: "required pointer field with null value should be valid",
			yml: `
requiredPtr: null
`,
			wantErrs: []string{}, // null is a valid value for required pointer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var model core.TestRequiredPointerModel
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, tt.yml), &model)
			require.NoError(t, err)

			if len(tt.wantErrs) == 0 {
				require.Empty(t, validationErrs)
			} else {
				require.NotEmpty(t, validationErrs)

				// Check that all expected error messages are present
				var errMessages []string
				for _, err := range validationErrs {
					errMessages = append(errMessages, err.Error())
				}

				for _, expectedErr := range tt.wantErrs {
					found := false
					for _, errMsg := range errMessages {
						if strings.Contains(errMsg, expectedErr) {
							found = true
							break
						}
					}
					require.True(t, found, "expected error message '%s' not found in: %v", expectedErr, errMessages)
				}
			}
		})
	}
}

func TestUnmarshal_RequiredNilableTypes_Success(t *testing.T) {
	t.Parallel()

	yml := `
requiredPtr: "required pointer value"
requiredSlice: ["item1", "item2"]
requiredMap:
  key1: "value1"
  key2: "value2"
requiredStruct:
  stringField: "nested required"
  stringPtrField: "nested ptr required"
  boolField: true
  intField: 42
  float64Field: 3.14
requiredEither: "either string value"
requiredRawNode: "raw node value"
`

	var model core.TestRequiredNilableModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, model.Valid)
	require.True(t, model.ValidYaml)

	// Check required fields are populated
	require.True(t, model.RequiredPtr.Present)
	require.NotNil(t, model.RequiredPtr.Value)
	require.Equal(t, "required pointer value", *model.RequiredPtr.Value)

	require.True(t, model.RequiredSlice.Present)
	require.Len(t, model.RequiredSlice.Value, 2)
	require.Equal(t, "item1", model.RequiredSlice.Value[0])
	require.Equal(t, "item2", model.RequiredSlice.Value[1])

	require.True(t, model.RequiredMap.Present)
	require.NotNil(t, model.RequiredMap.Value)
	require.Equal(t, "value1", model.RequiredMap.Value.GetOrZero("key1"))
	require.Equal(t, "value2", model.RequiredMap.Value.GetOrZero("key2"))

	require.True(t, model.RequiredStruct.Present)
	require.NotNil(t, model.RequiredStruct.Value)
	require.Equal(t, "nested required", model.RequiredStruct.Value.StringField.Value)

	// Optional fields should not be present
	require.False(t, model.OptionalPtr.Present)
	require.False(t, model.OptionalSlice.Present)
	require.False(t, model.OptionalMap.Present)
	require.False(t, model.OptionalStruct.Present)
}

func TestUnmarshal_RequiredNilableTypes_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing all required fields",
			yml: `
optionalPtr: "only optional set"
`,
			wantErrs: []string{
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredEither` is required",
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredMap` is required",
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredPtr` is required",
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredRawNode` is required",
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredSlice` is required",
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredStruct` is required",
			},
		},
		{
			name: "missing some required fields",
			yml: `
requiredPtr: "present"
requiredSlice: ["item1"]
# missing requiredMap, requiredStruct, requiredEither, requiredRawNode
`,
			wantErrs: []string{
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredEither` is required",
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredMap` is required",
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredRawNode` is required",
				"[2:1] error validation-required-field `testRequiredNilableModel.requiredStruct` is required",
			},
		},
		{
			name: "required struct with validation error",
			yml: `
requiredPtr: "present"
requiredSlice: ["item1"]
requiredMap:
  key1: "value1"
requiredStruct:
  # missing required stringField, boolField, intField, float64Field
  stringPtrField: "optional field present"
requiredEither: "string value"
requiredRawNode: "raw value"
`,
			wantErrs: []string{
				"[8:3] error validation-required-field `testPrimitiveModel.boolField` is required",
				"[8:3] error validation-required-field `testPrimitiveModel.float64Field` is required",
				"[8:3] error validation-required-field `testPrimitiveModel.intField` is required",
				"[8:3] error validation-required-field `testPrimitiveModel.stringField` is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var model core.TestRequiredNilableModel
			validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, tt.yml), &model)
			require.NoError(t, err)
			require.NotEmpty(t, validationErrs)
			validation.SortValidationErrors(validationErrs)

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range validationErrs {
				errMessages = append(errMessages, err.Error())
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}

func TestUnmarshal_TypeConversion_Error(t *testing.T) {
	t.Parallel()

	// This test reproduces the issue from openapi.Callback where:
	// - Core model uses string keys (like "post", "get")
	// - High-level model expects HTTPMethod keys
	// - Marshaller fails with "expected key to be of type HTTPMethod, got string"

	yml := `
post:
  stringField: "POST operation"
  boolField: true
  intField: 42
  float64Field: 3.14
get:
  stringField: "GET operation"
  boolField: false
  intField: 100
  float64Field: 1.23
put:
  stringField: "PUT operation"
  boolField: true
  intField: 200
  float64Field: 2.34
`

	var model core.TestTypeConversionCoreModel
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, yml), &model)

	// This should work fine for the core model (string keys)
	require.NoError(t, err)
	require.Empty(t, validationErrs)
	require.True(t, model.Valid)
	require.True(t, model.ValidYaml)

	// Verify core model populated correctly
	require.NotNil(t, model.Map)
	require.Equal(t, 3, model.Len())

	postOp, exists := model.Get("post")
	require.True(t, exists)
	require.Equal(t, "POST operation", postOp.Value.StringField.Value)

	getOp, exists := model.Get("get")
	require.True(t, exists)
	require.Equal(t, "GET operation", getOp.Value.StringField.Value)

	putOp, exists := model.Get("put")
	require.True(t, exists)
	require.Equal(t, "PUT operation", putOp.Value.StringField.Value)
}

func TestUnmarshal_NilOut_Error(t *testing.T) {
	t.Parallel()

	tts := []struct {
		name string
		yml  string
	}{
		{
			name: "simple yaml with nil out",
			yml: `
stringField: "test string"
boolField: true
intField: 42
float64Field: 3.14
`,
		},
		{
			name: "empty yaml with nil out",
			yml:  `{}`,
		},
	}

	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Define a nil pointer to a high-level model
			var model *tests.TestPrimitiveHighModel

			// This should not panic and should return a proper error
			validationErrs, err := marshaller.Unmarshal(t.Context(), strings.NewReader(tt.yml), model)

			// We expect an error, not a panic
			require.Error(t, err, "should return error when out is nil")
			require.Nil(t, validationErrs, "validation errors should be nil when there's a fundamental error")
			require.Contains(t, err.Error(), "out parameter cannot be nil", "error should indicate nil out parameter")
		})
	}
}

func TestUnmarshalNode_NilOut_Error(t *testing.T) {
	t.Parallel()

	yml := `
stringField: "test string"
boolField: true
intField: 42
float64Field: 3.14
`

	node := parseYAML(t, yml)

	// Define a nil pointer to a high-level model
	var model *tests.TestPrimitiveHighModel

	// This should not panic and should return a proper error
	validationErrs, err := marshaller.UnmarshalNode(t.Context(), "", node, model)

	// We expect an error, not a panic
	require.Error(t, err, "should return error when out is nil")
	require.Nil(t, validationErrs, "validation errors should be nil when there's a fundamental error")
	require.Contains(t, err.Error(), "out parameter cannot be nil", "error should indicate nil out parameter")
}

func TestDecodeNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		expected string
	}{
		{
			name:     "decode string value",
			yml:      `"test string"`,
			expected: "test string",
		},
		{
			name:     "decode unquoted string value",
			yml:      `test value`,
			expected: "test value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := parseYAML(t, tt.yml)
			// Navigate to scalar node (first content of document)
			scalarNode := node.Content[0]

			var result string
			validationErrs, err := marshaller.DecodeNode(t.Context(), "test", scalarNode, &result)
			require.NoError(t, err)
			require.Empty(t, validationErrs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions
func parseYAML(t *testing.T, yml string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)
	return &node
}
