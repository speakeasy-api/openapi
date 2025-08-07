package oas3_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
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
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &discriminator)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := discriminator.Validate(context.Background())
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
			yml: `
mapping:
  dog: "#/components/schemas/Dog"
`,
			wantErrs: []string{"[2:1] field propertyName is missing"},
		},
		{
			name: "empty property name",
			yml: `
propertyName: ""
mapping:
  dog: "#/components/schemas/Dog"
`,
			wantErrs: []string{"[2:15] propertyName is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var discriminator oas3.Discriminator

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &discriminator)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := discriminator.Validate(context.Background())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
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
