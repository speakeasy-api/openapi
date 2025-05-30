package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
)

type TestCoreStruct struct {
	Value string
}

func Test_Model_SetCore_Success(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() (*marshaller.Model[TestCoreStruct], *TestCoreStruct)
		verify func(t *testing.T, model *marshaller.Model[TestCoreStruct], expectedCore *TestCoreStruct)
	}{
		{
			name: "set valid core",
			setup: func() (*marshaller.Model[TestCoreStruct], *TestCoreStruct) {
				model := &marshaller.Model[TestCoreStruct]{}
				core := &TestCoreStruct{Value: "test"}
				return model, core
			},
			verify: func(t *testing.T, model *marshaller.Model[TestCoreStruct], expectedCore *TestCoreStruct) {
				assert.Equal(t, *expectedCore, *model.GetCore())
			},
		},
		{
			name: "set nil core",
			setup: func() (*marshaller.Model[TestCoreStruct], *TestCoreStruct) {
				model := &marshaller.Model[TestCoreStruct]{}
				model.GetCore().Value = "existing"
				return model, nil
			},
			verify: func(t *testing.T, model *marshaller.Model[TestCoreStruct], expectedCore *TestCoreStruct) {
				assert.Equal(t, "existing", model.GetCore().Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, core := tt.setup()
			model.SetCore(core)
			tt.verify(t, model, core)
		})
	}
}

func Test_Model_SetCoreValue_Success(t *testing.T) {
	tests := []struct {
		name     string
		coreValue any
		expected TestCoreStruct
	}{
		{
			name:     "set core value with pointer",
			coreValue: &TestCoreStruct{Value: "test-ptr"},
			expected: TestCoreStruct{Value: "test-ptr"},
		},
		{
			name:     "set core value direct",
			coreValue: TestCoreStruct{Value: "test-direct"},
			expected: TestCoreStruct{Value: "test-direct"},
		},
		{
			name:     "set incompatible type",
			coreValue: "string",
			expected: TestCoreStruct{}, // should remain zero value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &marshaller.Model[TestCoreStruct]{}
			model.SetCoreValue(tt.coreValue)
			assert.Equal(t, tt.expected, *model.GetCore())
		})
	}
}