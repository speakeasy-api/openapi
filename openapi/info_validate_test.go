package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestInfo_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid info with all fields",
			yml: `
title: Test API
version: 1.0.0
summary: A test API
description: A comprehensive test API
termsOfService: https://example.com/terms
contact:
  name: API Support
  url: https://example.com/support
  email: support@example.com
license:
  name: MIT
  url: https://opensource.org/licenses/MIT
`,
		},
		{
			name: "valid info with minimal required fields",
			yml: `
title: Test API
version: 1.0.0
`,
		},
		{
			name: "valid info with contact only",
			yml: `
title: Test API
version: 1.0.0
contact:
  name: API Support
`,
		},
		{
			name: "valid info with license only",
			yml: `
title: Test API
version: 1.0.0
license:
  name: Apache 2.0
`,
		},
		{
			name: "valid info with license identifier",
			yml: `
title: Test API
version: 1.0.0
license:
  name: Apache 2.0
  identifier: Apache-2.0
`,
		},
		{
			name: "valid info with valid termsOfService URI",
			yml: `
title: Test API
version: 1.0.0
termsOfService: https://example.com/terms-of-service
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var info openapi.Info
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &info)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := info.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, info.Valid, "expected info to be valid")
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
			name: "missing title",
			yml: `
version: 1.0.0
`,
			wantErrs: []string{"[2:1] info field title is missing"},
		},
		{
			name: "empty title",
			yml: `
title: ""
version: 1.0.0
`,
			wantErrs: []string{"[2:8] info field title is required"},
		},
		{
			name: "missing version",
			yml: `
title: Test API
`,
			wantErrs: []string{"[2:1] info field version is missing"},
		},
		{
			name: "empty version",
			yml: `
title: Test API
version: ""
`,
			wantErrs: []string{"[3:10] info field version is required"},
		},
		{
			name: "invalid termsOfService URI",
			yml: `
title: Test API
version: 1.0.0
termsOfService: ":invalid"
`,
			wantErrs: []string{"[4:17] info field termsOfService is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "invalid contact URL",
			yml: `
title: Test API
version: 1.0.0
contact:
  name: Support
  url: ":invalid"
`,
			wantErrs: []string{"[6:8] contact field url is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "invalid contact email",
			yml: `
title: Test API
version: 1.0.0
contact:
  name: Support
  email: "not-an-email"
`,
			wantErrs: []string{"[6:10] contact field email is not a valid email address: mail: missing '@' or angle-addr"},
		},
		{
			name: "invalid license URL",
			yml: `
title: Test API
version: 1.0.0
license:
  name: MIT
  url: ":invalid"
`,
			wantErrs: []string{"[6:8] license field url is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "missing license name",
			yml: `
title: Test API
version: 1.0.0
license:
  url: https://opensource.org/licenses/MIT
`,
			wantErrs: []string{"[5:3] license field name is missing"},
		},
		{
			name: "multiple validation errors",
			yml: `
title: ""
version: ""
contact:
  email: "invalid-email"
license:
  name: ""
`,
			wantErrs: []string{
				"[2:8] info field title is required",
				"[3:10] info field version is required",
				"[5:10] contact field email is not a valid email address: mail: missing '@' or angle-addr",
				"[7:9] license field name is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var info openapi.Info
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &info)
			require.NoError(t, err)

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			allErrors = append(allErrors, validationErrs...)

			validateErrs := info.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
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

func TestContact_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid contact with all fields",
			yml: `
name: API Support
url: https://example.com/support
email: support@example.com
`,
		},
		{
			name: "valid contact with name only",
			yml: `
name: API Support
`,
		},
		{
			name: "valid contact with email only",
			yml: `
email: support@example.com
`,
		},
		{
			name: "valid contact with URL only",
			yml: `
url: https://example.com/support
`,
		},
		{
			name: "empty contact",
			yml: `
name: ""
`,
		},
		{
			name: "valid contact with complex email",
			yml: `
name: Support Team
email: support+team@example.com
`,
		},
		{
			name: "valid contact with URL path",
			yml: `
name: Support
url: https://api.example.com/v1/support/contact
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var contact openapi.Contact
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &contact)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := contact.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, contact.Valid, "expected contact to be valid")
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
			name: "invalid URL",
			yml: `
name: Support
url: ":invalid"
`,
			wantErrs: []string{"[3:6] contact field url is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "invalid email",
			yml: `
name: Support
email: "not-an-email"
`,
			wantErrs: []string{"[3:8] contact field email is not a valid email address: mail: missing '@' or angle-addr"},
		},
		{
			name: "invalid URL with spaces",
			yml: `
name: Support
url: ":invalid url"
`,
			wantErrs: []string{"[3:6] contact field url is not a valid uri: parse \":invalid url\": missing protocol scheme"},
		},
		{
			name: "invalid email missing @",
			yml: `
name: Support
email: "supportexample.com"
`,
			wantErrs: []string{"[3:8] contact field email is not a valid email address: mail: missing '@' or angle-addr"},
		},
		{
			name: "multiple validation errors",
			yml: `
name: Support
url: ":invalid"
email: "invalid-email"
`,
			wantErrs: []string{
				"[3:6] contact field url is not a valid uri: parse \":invalid\": missing protocol scheme",
				"[4:8] contact field email is not a valid email address: mail: missing '@' or angle-addr",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var contact openapi.Contact
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &contact)
			require.NoError(t, err)

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			allErrors = append(allErrors, validationErrs...)

			validateErrs := contact.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
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

func TestLicense_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid license with name and URL",
			yml: `
name: MIT License
url: https://opensource.org/licenses/MIT
`,
		},
		{
			name: "valid license with name and identifier",
			yml: `
name: Apache 2.0
identifier: Apache-2.0
`,
		},
		{
			name: "valid license with name only",
			yml: `
name: Custom License
`,
		},
		{
			name: "valid license with all fields",
			yml: `
name: Apache 2.0
identifier: Apache-2.0
url: https://www.apache.org/licenses/LICENSE-2.0.html
`,
		},
		{
			name: "valid license with SPDX identifier",
			yml: `
name: BSD 3-Clause License
identifier: BSD-3-Clause
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var license openapi.License
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &license)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := license.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, license.Valid, "expected license to be valid")
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
			name: "missing name",
			yml: `
url: https://opensource.org/licenses/MIT
`,
			wantErrs: []string{"[2:1] license field name is missing"},
		},
		{
			name: "empty name",
			yml: `
name: ""
url: https://opensource.org/licenses/MIT
`,
			wantErrs: []string{"[2:7] license field name is required"},
		},
		{
			name: "invalid URL",
			yml: `
name: MIT
url: ":invalid"
`,
			wantErrs: []string{"[3:6] license field url is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "invalid URL with spaces",
			yml: `
name: MIT
url: ":invalid url"
`,
			wantErrs: []string{"[3:6] license field url is not a valid uri: parse \":invalid url\": missing protocol scheme"},
		},
		{
			name: "multiple validation errors",
			yml: `
name: ""
url: ":invalid"
`,
			wantErrs: []string{
				"[2:7] license field name is required",
				"[3:6] license field url is not a valid uri: parse \":invalid\": missing protocol scheme",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var license openapi.License
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &license)
			require.NoError(t, err)

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			allErrors = append(allErrors, validationErrs...)

			validateErrs := license.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
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
