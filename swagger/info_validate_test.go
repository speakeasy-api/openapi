package swagger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/require"
)

func TestInfo_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "minimal_valid_info",
			yml: `title: Test API
version: 1.0.0`,
		},
		{
			name: "complete_info",
			yml: `title: Test API
version: 1.0.0
description: A test API
termsOfService: https://example.com/terms
contact:
  name: API Support
  url: https://example.com/support
  email: support@example.com
license:
  name: Apache 2.0
  url: https://www.apache.org/licenses/LICENSE-2.0.html`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var info swagger.Info

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &info)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := info.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestInfo_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name:     "missing_title",
			yml:      `version: 1.0.0`,
			wantErrs: []string{"info.title is missing"},
		},
		{
			name:     "missing_version",
			yml:      `title: Test API`,
			wantErrs: []string{"info.version is missing"},
		},
		{
			name: "invalid_contact_email",
			yml: `title: Test API
version: 1.0.0
contact:
  email: not-an-email`,
			wantErrs: []string{"contact.email is not a valid email address"},
		},
		{
			name: "missing_license_name",
			yml: `title: Test API
version: 1.0.0
license:
  url: https://example.com/license`,
			wantErrs: []string{"license.name is missing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var info swagger.Info

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &info)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := info.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "Expected validation errors")

			for _, wantErr := range tt.wantErrs {
				found := false
				for _, gotErr := range allErrors {
					if gotErr != nil && strings.Contains(gotErr.Error(), wantErr) {
						found = true
						break
					}
				}
				require.True(t, found, "Expected error containing '%s' not found in: %v", wantErr, allErrors)
			}
		})
	}
}

func TestContact_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_contact_with_all_fields",
			yml: `name: API Support
url: https://example.com/support
email: support@example.com`,
		},
		{
			name: "valid_contact_with_name_only",
			yml:  `name: API Support`,
		},
		{
			name: "valid_contact_with_email_only",
			yml:  `email: support@example.com`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var contact swagger.Contact

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &contact)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := contact.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestContact_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name:     "invalid_email",
			yml:      `email: not-an-email`,
			wantErrs: []string{"contact.email is not a valid email address"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var contact swagger.Contact

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &contact)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := contact.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "Expected validation errors")

			for _, wantErr := range tt.wantErrs {
				found := false
				for _, gotErr := range allErrors {
					if gotErr != nil && strings.Contains(gotErr.Error(), wantErr) {
						found = true
						break
					}
				}
				require.True(t, found, "Expected error containing '%s' not found in: %v", wantErr, allErrors)
			}
		})
	}
}

func TestLicense_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_license_with_url",
			yml: `name: Apache 2.0
url: https://www.apache.org/licenses/LICENSE-2.0.html`,
		},
		{
			name: "valid_license_without_url",
			yml:  `name: MIT`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var license swagger.License

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &license)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := license.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestLicense_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name:     "missing_name",
			yml:      `url: https://example.com/license`,
			wantErrs: []string{"license.name is missing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var license swagger.License

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &license)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := license.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "Expected validation errors")

			for _, wantErr := range tt.wantErrs {
				found := false
				for _, gotErr := range allErrors {
					if gotErr != nil && strings.Contains(gotErr.Error(), wantErr) {
						found = true
						break
					}
				}
				require.True(t, found, "Expected error containing '%s' not found in: %v", wantErr, allErrors)
			}
		})
	}
}
