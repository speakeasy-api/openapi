package oas3_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscriminator_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid discriminator with property name only",
			yml: `
propertyName: petType
`,
		},
		{
			name: "valid discriminator with property name and mapping",
			yml: `
propertyName: petType
mapping:
  dog: "#/components/schemas/Dog"
  cat: "#/components/schemas/Cat"
`,
		},
		{
			name: "valid discriminator with complex mapping",
			yml: `
propertyName: objectType
mapping:
  user: "#/components/schemas/User"
  admin: "#/components/schemas/AdminUser"
  guest: "#/components/schemas/GuestUser"
`,
		},
		{
			name: "valid discriminator with extensions",
			yml: `
propertyName: type
mapping:
  typeA: "#/components/schemas/TypeA"
x-test: some-value
x-custom: custom-value
`,
		},
		{
			name: "valid discriminator with empty mapping",
			yml: `
propertyName: discriminatorField
mapping: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var discriminator oas3.Discriminator
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &discriminator)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := discriminator.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, discriminator.Valid, "expected discriminator to be valid")
		})
	}
}

func TestDiscriminator_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing property name",
			yml: `mapping:
  dog: "#/components/schemas/Dog"
`,
			wantErrs: []string{
				"[1:1] error validation-required-field discriminator.propertyName is required",
			},
		},
		{
			name: "empty property name",
			yml: `
propertyName: ""
mapping:
  dog: "#/components/schemas/Dog"
`,
			wantErrs: []string{"[2:15] error validation-required-field discriminator.propertyName is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var discriminator oas3.Discriminator

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &discriminator)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := discriminator.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}

func TestDiscriminator_GetPropertyName_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		discriminator *oas3.Discriminator
		expected      string
	}{
		{
			name:          "nil discriminator returns empty",
			discriminator: nil,
			expected:      "",
		},
		{
			name:          "empty discriminator returns empty",
			discriminator: &oas3.Discriminator{},
			expected:      "",
		},
		{
			name:          "returns property name",
			discriminator: &oas3.Discriminator{PropertyName: "petType"},
			expected:      "petType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.discriminator.GetPropertyName())
		})
	}
}

func TestDiscriminator_GetMapping_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		discriminator *oas3.Discriminator
		expectNil     bool
	}{
		{
			name:          "nil discriminator returns nil",
			discriminator: nil,
			expectNil:     true,
		},
		{
			name:          "nil mapping returns nil",
			discriminator: &oas3.Discriminator{},
			expectNil:     true,
		},
		{
			name: "returns mapping",
			discriminator: &oas3.Discriminator{
				Mapping: sequencedmap.New[string, string](),
			},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.discriminator.GetMapping()
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestDiscriminator_GetDefaultMapping_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		discriminator *oas3.Discriminator
		expected      string
	}{
		{
			name:          "nil discriminator returns empty",
			discriminator: nil,
			expected:      "",
		},
		{
			name:          "nil default mapping returns empty",
			discriminator: &oas3.Discriminator{},
			expected:      "",
		},
		{
			name: "returns default mapping",
			discriminator: &oas3.Discriminator{
				DefaultMapping: pointer.From("#/components/schemas/Default"),
			},
			expected: "#/components/schemas/Default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.discriminator.GetDefaultMapping())
		})
	}
}

func TestDiscriminator_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		discriminator *oas3.Discriminator
		expectEmpty   bool
	}{
		{
			name:          "nil discriminator returns empty extensions",
			discriminator: nil,
			expectEmpty:   true,
		},
		{
			name:          "nil extensions returns empty extensions",
			discriminator: &oas3.Discriminator{},
			expectEmpty:   true,
		},
		{
			name: "returns extensions",
			discriminator: &oas3.Discriminator{
				Extensions: extensions.New(),
			},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.discriminator.GetExtensions()
			assert.NotNil(t, result)
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			}
		})
	}
}

func TestDiscriminator_IsEqual_Success(t *testing.T) {
	t.Parallel()

	mapping := sequencedmap.New[string, string]()
	mapping.Set("dog", "#/components/schemas/Dog")

	tests := []struct {
		name     string
		a        *oas3.Discriminator
		b        *oas3.Discriminator
		expected bool
	}{
		{
			name:     "both nil returns true",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "a nil b not nil returns false",
			a:        nil,
			b:        &oas3.Discriminator{PropertyName: "type"},
			expected: false,
		},
		{
			name:     "a not nil b nil returns false",
			a:        &oas3.Discriminator{PropertyName: "type"},
			b:        nil,
			expected: false,
		},
		{
			name:     "equal property names returns true",
			a:        &oas3.Discriminator{PropertyName: "type"},
			b:        &oas3.Discriminator{PropertyName: "type"},
			expected: true,
		},
		{
			name:     "different property names returns false",
			a:        &oas3.Discriminator{PropertyName: "type1"},
			b:        &oas3.Discriminator{PropertyName: "type2"},
			expected: false,
		},
		{
			name:     "both nil default mapping returns true",
			a:        &oas3.Discriminator{PropertyName: "type"},
			b:        &oas3.Discriminator{PropertyName: "type"},
			expected: true,
		},
		{
			name:     "one nil default mapping returns false",
			a:        &oas3.Discriminator{PropertyName: "type", DefaultMapping: pointer.From("default")},
			b:        &oas3.Discriminator{PropertyName: "type"},
			expected: false,
		},
		{
			name:     "different default mapping returns false",
			a:        &oas3.Discriminator{PropertyName: "type", DefaultMapping: pointer.From("default1")},
			b:        &oas3.Discriminator{PropertyName: "type", DefaultMapping: pointer.From("default2")},
			expected: false,
		},
		{
			name:     "equal default mapping returns true",
			a:        &oas3.Discriminator{PropertyName: "type", DefaultMapping: pointer.From("default")},
			b:        &oas3.Discriminator{PropertyName: "type", DefaultMapping: pointer.From("default")},
			expected: true,
		},
		{
			name:     "both nil mapping returns true",
			a:        &oas3.Discriminator{PropertyName: "type"},
			b:        &oas3.Discriminator{PropertyName: "type"},
			expected: true,
		},
		{
			name:     "one nil mapping returns false",
			a:        &oas3.Discriminator{PropertyName: "type", Mapping: mapping},
			b:        &oas3.Discriminator{PropertyName: "type"},
			expected: false,
		},
		{
			name:     "equal mapping returns true",
			a:        &oas3.Discriminator{PropertyName: "type", Mapping: mapping},
			b:        &oas3.Discriminator{PropertyName: "type", Mapping: mapping},
			expected: true,
		},
		{
			name:     "both nil extensions returns true",
			a:        &oas3.Discriminator{PropertyName: "type"},
			b:        &oas3.Discriminator{PropertyName: "type"},
			expected: true,
		},
		{
			name:     "one nil extensions returns false",
			a:        &oas3.Discriminator{PropertyName: "type", Extensions: extensions.New()},
			b:        &oas3.Discriminator{PropertyName: "type"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.a.IsEqual(tt.b))
		})
	}
}
