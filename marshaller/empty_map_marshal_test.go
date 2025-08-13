package marshaller_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal_TestEmbeddedMapModel_Empty_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func() *tests.TestEmbeddedMapHighModel
		expected string
	}{
		{
			name: "uninitialized embedded map should render as empty object",
			setup: func() *tests.TestEmbeddedMapHighModel {
				model := &tests.TestEmbeddedMapHighModel{}
				// Don't initialize the embedded map - this simulates the bug
				return model
			},
			expected: "{}\n",
		},
		{
			name: "initialized empty embedded map should render as empty object",
			setup: func() *tests.TestEmbeddedMapHighModel {
				model := &tests.TestEmbeddedMapHighModel{}
				model.Map = *sequencedmap.New[string, string]()
				return model
			},
			expected: "{}\n",
		},
		{
			name: "embedded map with content should render normally",
			setup: func() *tests.TestEmbeddedMapHighModel {
				model := &tests.TestEmbeddedMapHighModel{}
				model.Map = *sequencedmap.New[string, string]()
				model.Set("key1", "value1")
				return model
			},
			expected: "key1: value1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := tt.setup()

			var buf bytes.Buffer
			err := marshaller.Marshal(t.Context(), model, &buf)
			require.NoError(t, err)

			actual := buf.String()
			assert.Equal(t, tt.expected, actual, "marshaled output should match expected")
		})
	}
}

func TestMarshal_TestEmbeddedMapWithFieldsModel_Empty_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func() *tests.TestEmbeddedMapWithFieldsHighModel
		expected string
	}{
		{
			name: "uninitialized embedded map with fields should render fields only",
			setup: func() *tests.TestEmbeddedMapWithFieldsHighModel {
				model := &tests.TestEmbeddedMapWithFieldsHighModel{}
				model.NameField = "test name"
				// Don't initialize the embedded map
				return model
			},
			expected: "name: test name\n",
		},
		{
			name: "initialized empty embedded map with fields should render fields only",
			setup: func() *tests.TestEmbeddedMapWithFieldsHighModel {
				model := &tests.TestEmbeddedMapWithFieldsHighModel{}
				model.Map = *sequencedmap.New[string, *tests.TestPrimitiveHighModel]()
				model.NameField = "test name"
				return model
			},
			expected: "name: test name\n",
		},
		{
			name: "embedded map with content and fields should render both",
			setup: func() *tests.TestEmbeddedMapWithFieldsHighModel {
				model := &tests.TestEmbeddedMapWithFieldsHighModel{}
				model.Map = *sequencedmap.New[string, *tests.TestPrimitiveHighModel]()
				model.NameField = "test name"
				model.Set("key1", &tests.TestPrimitiveHighModel{
					StringField: "value1",
					BoolField:   true,
				})
				return model
			},
			expected: "key1:\n  stringField: value1\n  boolField: true\n  intField: 0\n  float64Field: 0\nname: test name\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := tt.setup()

			var buf bytes.Buffer
			err := marshaller.Marshal(t.Context(), model, &buf)
			require.NoError(t, err)

			actual := buf.String()
			assert.Equal(t, tt.expected, actual, "marshaled output should match expected")
		})
	}
}

func TestMarshal_TestEmbeddedMapModel_RoundTrip_Empty_Success(t *testing.T) {
	t.Parallel()

	inputYAML := "{}\n"

	// Unmarshal empty object -> Marshal -> Compare
	reader := strings.NewReader(inputYAML)
	model := &tests.TestEmbeddedMapHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}
