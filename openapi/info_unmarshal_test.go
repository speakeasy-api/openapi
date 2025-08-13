package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestInfo_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
title: Test OpenAPI Document
version: 1.0.0
summary: A summary
description: A description
termsOfService: https://example.com/terms
contact:
  name: API Support
  url: https://example.com/support
  email: support@example.com
  x-test: some-value
license:
  name: Apache 2.0
  identifier: Apache-2.0
  url: https://www.apache.org/licenses/LICENSE-2.0.html
  x-test: some-value
x-test: some-value
`

	var info openapi.Info

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &info)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "Test OpenAPI Document", info.GetTitle())
	require.Equal(t, "1.0.0", info.GetVersion())
	require.Equal(t, "A summary", info.GetSummary())
	require.Equal(t, "A description", info.GetDescription())
	require.Equal(t, "https://example.com/terms", info.GetTermsOfService())

	contact := info.GetContact()
	require.NotNil(t, contact)
	require.Equal(t, "API Support", contact.GetName())
	require.Equal(t, "https://example.com/support", contact.GetURL())
	require.Equal(t, "support@example.com", contact.GetEmail())

	ext, ok := contact.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)

	license := info.GetLicense()
	require.NotNil(t, license)
	require.Equal(t, "Apache 2.0", license.GetName())
	require.Equal(t, "Apache-2.0", license.GetIdentifier())
	require.Equal(t, "https://www.apache.org/licenses/LICENSE-2.0.html", license.GetURL())

	ext, ok = license.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)

	ext, ok = info.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}

func TestContact_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: API Support
url: https://example.com/support
email: support@example.com
x-test: some-value
`

	var contact openapi.Contact

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &contact)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "API Support", contact.GetName())
	require.Equal(t, "https://example.com/support", contact.GetURL())
	require.Equal(t, "support@example.com", contact.GetEmail())

	ext, ok := contact.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}

func TestLicense_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: Apache 2.0
identifier: Apache-2.0
url: https://www.apache.org/licenses/LICENSE-2.0.html
x-test: some-value
`

	var license openapi.License

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &license)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "Apache 2.0", license.GetName())
	require.Equal(t, "Apache-2.0", license.GetIdentifier())
	require.Equal(t, "https://www.apache.org/licenses/LICENSE-2.0.html", license.GetURL())

	ext, ok := license.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}
