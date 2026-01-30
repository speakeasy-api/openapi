package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMediaType_MultipartValidation_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "prefixEncoding with multipart/mixed",
			yml: `
description: Test response
content:
  multipart/mixed:
    schema:
      type: array
      prefixItems:
        - type: object
        - type: string
    prefixEncoding:
      - contentType: application/json
      - contentType: text/plain
`,
		},
		{
			name: "itemEncoding with multipart/form-data",
			yml: `
description: Test response
content:
  multipart/form-data:
    itemSchema:
      type: object
    itemEncoding:
      contentType: application/json
`,
		},
		{
			name: "encoding with multipart/form-data",
			yml: `
description: Test response
content:
  multipart/form-data:
    schema:
      type: object
      properties:
        file:
          type: string
    encoding:
      file:
        contentType: image/png
`,
		},
		{
			name: "encoding with application/x-www-form-urlencoded",
			yml: `
description: Test response
content:
  application/x-www-form-urlencoded:
    schema:
      type: object
      properties:
        name:
          type: string
    encoding:
      name:
        contentType: text/plain
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var response openapi.Response
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &response)
			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "unmarshal validation should succeed")

			errs := response.Validate(t.Context())
			require.Empty(t, errs, "validation should succeed")
			require.True(t, response.Valid, "response should be valid")
		})
	}
}

func TestMediaType_MultipartValidation_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "prefixEncoding with non-multipart media type",
			yml: `
description: Test response
content:
   application/json:
     schema:
       type: array
     prefixEncoding:
       - contentType: application/json
`,
			wantErrs: []string{
				"error validation-allowed-values mediaType.prefixEncoding is only valid when the media type is multipart",
			},
		},
		{
			name: "itemEncoding with non-multipart media type",
			yml: `
description: Test response
content:
   application/json:
     itemSchema:
       type: object
     itemEncoding:
       contentType: application/json
`,
			wantErrs: []string{
				"error validation-allowed-values mediaType.itemEncoding is only valid when the media type is multipart",
			},
		},
		{
			name: "encoding with non-multipart non-form-urlencoded media type",
			yml: `
description: Test response
content:
   application/json:
     schema:
       type: object
       properties:
         file:
           type: string
     encoding:
       file:
         contentType: image/png
`,
			wantErrs: []string{
				"error validation-allowed-values mediaType.encoding is only valid when the media type is multipart or application/x-www-form-urlencoded",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var response openapi.Response
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &response)
			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "unmarshal validation should succeed")

			errs := response.Validate(t.Context())
			require.NotEmpty(t, errs, "validation should fail")
			require.False(t, response.Valid, "response should be invalid")

			var errMessages []string
			for _, err := range errs {
				errMessages = append(errMessages, err.Error())
			}

			for _, wantErr := range tt.wantErrs {
				found := false
				for _, msg := range errMessages {
					if assert.Contains(t, msg, wantErr) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected error message not found: %s", wantErr)
			}
		})
	}
}
