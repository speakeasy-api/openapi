package marshaller_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	testmodels "github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// joinErrors converts a slice of errors to a single string for assertion checks.
func joinErrors(errs []error) string {
	errStrs := make([]string, len(errs))
	for i, e := range errs {
		errStrs[i] = e.Error()
	}
	return strings.Join(errStrs, " ")
}

func TestUnmarshal_DuplicateKey_ReturnsValidationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yaml           string
		expectedErrors int
		errorContains  []string
	}{
		{
			name: "single duplicate key",
			yaml: `stringField: "first value"
boolField: true
stringField: "second value"
intField: 42
float64Field: 3.14
`,
			expectedErrors: 1,
			errorContains:  []string{"stringField", "duplicate"},
		},
		{
			name: "multiple duplicate keys",
			yaml: `stringField: "first value"
boolField: true
stringField: "second value"
intField: 42
boolField: false
float64Field: 3.14
`,
			expectedErrors: 2,
			errorContains:  []string{"stringField", "boolField", "duplicate"},
		},
		{
			name: "same key three times",
			yaml: `stringField: "first value"
boolField: true
stringField: "second value"
intField: 42
stringField: "third value"
float64Field: 3.14
`,
			expectedErrors: 2,
			errorContains:  []string{"stringField", "duplicate"},
		},
		{
			name: "no duplicates",
			yaml: `stringField: "test string"
boolField: true
intField: 42
float64Field: 3.14
`,
			expectedErrors: 0,
			errorContains:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := strings.NewReader(tt.yaml)
			model := &testmodels.TestPrimitiveHighModel{}
			validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
			require.NoError(t, err, "unmarshal should not return a fatal error")

			assert.Len(t, validationErrs, tt.expectedErrors, "should have expected number of validation errors")

			if tt.errorContains != nil {
				errStr := joinErrors(validationErrs)
				for _, contains := range tt.errorContains {
					assert.Contains(t, errStr, contains, "validation error should contain expected text")
				}
			}
		})
	}
}

func TestUnmarshal_DuplicateKey_LastValueWins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedValue string
	}{
		{
			name: "last value wins for string field",
			yaml: `stringField: "first value"
boolField: true
stringField: "second value"
intField: 42
float64Field: 3.14
`,
			expectedValue: "second value",
		},
		{
			name: "last value wins with three occurrences",
			yaml: `stringField: "first value"
boolField: true
stringField: "second value"
intField: 42
stringField: "third value"
float64Field: 3.14
`,
			expectedValue: "third value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := strings.NewReader(tt.yaml)
			model := &testmodels.TestPrimitiveHighModel{}
			validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
			require.NoError(t, err, "unmarshal should not return a fatal error")

			// We expect validation errors for duplicates, but the model should still be populated
			assert.NotEmpty(t, validationErrs, "should have validation errors for duplicate keys")

			// Per YAML spec, the last value should win
			assert.Equal(t, tt.expectedValue, model.StringField, "last value should be used")
		})
	}
}

func TestUnmarshal_DuplicateKey_NestedModel(t *testing.T) {
	t.Parallel()

	yaml := `nestedModelValue:
  stringField: "first nested"
  boolField: true
  stringField: "second nested"
  intField: 100
  float64Field: 1.23
eitherModelOrPrimitive: 999
`

	reader := strings.NewReader(yaml)
	model := &testmodels.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err, "unmarshal should not return a fatal error")

	// Should have validation error for nested duplicate
	assert.NotEmpty(t, validationErrs, "should have validation errors for nested duplicate keys")

	// Last value should win
	assert.Equal(t, "second nested", model.NestedModelValue.StringField, "last value should be used in nested model")
}

func TestUnmarshal_DuplicateKey_WithExtensions(t *testing.T) {
	t.Parallel()

	yaml := `stringField: "test string"
boolField: true
x-custom: "first extension"
intField: 42
x-custom: "second extension"
float64Field: 3.14
`

	reader := strings.NewReader(yaml)
	model := &testmodels.TestPrimitiveHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err, "unmarshal should not return a fatal error")

	// Should have validation error for duplicate extension key
	assert.NotEmpty(t, validationErrs, "should have validation errors for duplicate extension keys")

	errStr := joinErrors(validationErrs)
	assert.Contains(t, errStr, "x-custom", "validation error should mention the duplicate extension key")
}

func TestUnmarshal_DuplicateKey_RaceCondition(t *testing.T) {
	t.Parallel()

	// This test verifies that duplicate keys don't cause race conditions
	// by testing concurrent unmarshalling with duplicate keys
	yaml := `stringField: "value1"
boolField: true
stringField: "value2"
intField: 42
stringField: "value3"
float64Field: 3.14
boolField: false
intField: 100
`

	// Run multiple times to increase chance of catching race condition
	for i := 0; i < 10; i++ {
		t.Run("iteration", func(t *testing.T) {
			t.Parallel()

			reader := strings.NewReader(yaml)
			model := &testmodels.TestPrimitiveHighModel{}
			validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
			require.NoError(t, err, "unmarshal should not return a fatal error")

			// Should have validation errors for duplicates
			assert.NotEmpty(t, validationErrs, "should have validation errors")

			// Values should be consistent (last value wins)
			assert.Equal(t, "value3", model.StringField, "string field should have last value")
			assert.False(t, model.BoolField, "bool field should have last value")
			assert.Equal(t, 100, model.IntField, "int field should have last value")
		})
	}
}

func TestUnmarshal_DuplicateKey_EmbeddedMap(t *testing.T) {
	t.Parallel()

	yaml := `dynamicKey1: "value1"
dynamicKey2: "value2"
dynamicKey1: "value3"
`

	reader := strings.NewReader(yaml)
	model := &testmodels.TestEmbeddedMapHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err, "unmarshal should not return a fatal error")

	// Should have validation error for duplicate key
	assert.NotEmpty(t, validationErrs, "should have validation errors for duplicate key in embedded map")

	errStr := joinErrors(validationErrs)
	assert.Contains(t, errStr, "dynamicKey1", "validation error should mention the duplicate key")

	// Last value should win
	val, exists := model.Get("dynamicKey1")
	assert.True(t, exists, "key should exist in map")
	assert.Equal(t, "value3", val, "last value should be used")
}

func TestUnmarshal_DuplicateKey_EmbeddedMapWithFields(t *testing.T) {
	t.Parallel()

	yaml := `name: "test name"
dynamicKey1:
  stringField: "first nested"
  boolField: true
  intField: 100
  float64Field: 1.23
dynamicKey1:
  stringField: "second nested"
  boolField: false
  intField: 200
  float64Field: 4.56
`

	reader := strings.NewReader(yaml)
	model := &testmodels.TestEmbeddedMapWithFieldsHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err, "unmarshal should not return a fatal error")

	// Should have validation error for duplicate key
	assert.NotEmpty(t, validationErrs, "should have validation errors for duplicate key in embedded map with fields")

	errStr := joinErrors(validationErrs)
	assert.Contains(t, errStr, "dynamicKey1", "validation error should mention the duplicate key")

	// Last value should win
	val, exists := model.Get("dynamicKey1")
	assert.True(t, exists, "key should exist in map")
	require.NotNil(t, val, "value should not be nil")
	assert.Equal(t, "second nested", val.StringField, "last value's string field should be used")
	assert.False(t, val.BoolField, "last value's bool field should be used")
	assert.Equal(t, 200, val.IntField, "last value's int field should be used")
}
