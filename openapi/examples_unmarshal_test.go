package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestExample_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
		test func(t *testing.T, example *openapi.Example)
	}{
		{
			name: "all legacy fields",
			yml: `
summary: Example of a pet
description: A pet object example
value:
  id: 1
  name: doggie
  status: available
externalValue: https://example.com/examples/pet.json
x-test: some-value
`,
			test: func(t *testing.T, example *openapi.Example) {
				t.Helper()
				require.Equal(t, "Example of a pet", example.GetSummary())
				require.Equal(t, "A pet object example", example.GetDescription())
				require.Equal(t, "https://example.com/examples/pet.json", example.GetExternalValue())

				value := example.GetValue()
				require.NotNil(t, value)

				ext, ok := example.GetExtensions().Get("x-test")
				require.True(t, ok)
				require.Equal(t, "some-value", ext.Value)
			},
		},
		{
			name: "dataValue field",
			yml: `
summary: Data value example
description: Example using dataValue
dataValue:
  author: A. Writer
  title: The Newest Book
`,
			test: func(t *testing.T, example *openapi.Example) {
				t.Helper()
				require.Equal(t, "Data value example", example.GetSummary())
				require.Equal(t, "Example using dataValue", example.GetDescription())

				dataValue := example.GetDataValue()
				require.NotNil(t, dataValue)
			},
		},
		{
			name: "serializedValue field",
			yml: `
summary: Serialized value example
serializedValue: "flag=true"
`,
			test: func(t *testing.T, example *openapi.Example) {
				t.Helper()
				require.Equal(t, "Serialized value example", example.GetSummary())
				require.Equal(t, "flag=true", example.GetSerializedValue())
			},
		},
		{
			name: "dataValue and serializedValue together",
			yml: `
summary: Combined example
dataValue:
  author: A. Writer
  title: An Older Book
  rating: 4.5
serializedValue: '{"author":"A. Writer","title":"An Older Book","rating":4.5}'
`,
			test: func(t *testing.T, example *openapi.Example) {
				t.Helper()
				require.Equal(t, "Combined example", example.GetSummary())

				dataValue := example.GetDataValue()
				require.NotNil(t, dataValue)

				serializedValue := example.GetSerializedValue()
				require.Equal(t, `{"author":"A. Writer","title":"An Older Book","rating":4.5}`, serializedValue)
			},
		},
		{
			name: "serializedValue with JSON content",
			yml: `
serializedValue: '{"name":"Fluffy","petType":"Cat","color":"White"}'
`,
			test: func(t *testing.T, example *openapi.Example) {
				t.Helper()
				require.Equal(t, `{"name":"Fluffy","petType":"Cat","color":"White"}`, example.GetSerializedValue())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var example openapi.Example

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &example)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			tt.test(t, &example)
		})
	}
}
