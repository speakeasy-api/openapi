package openapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponents_Nil(t *testing.T) {
	t.Parallel()

	var c *openapi.Components

	assert.Nil(t, c.GetSchemas(), "nil Components should return nil for GetSchemas")
	assert.Nil(t, c.GetResponses(), "nil Components should return nil for GetResponses")
	assert.Nil(t, c.GetParameters(), "nil Components should return nil for GetParameters")
	assert.Nil(t, c.GetExamples(), "nil Components should return nil for GetExamples")
	assert.Nil(t, c.GetRequestBodies(), "nil Components should return nil for GetRequestBodies")
	assert.Nil(t, c.GetHeaders(), "nil Components should return nil for GetHeaders")
	assert.Nil(t, c.GetSecuritySchemes(), "nil Components should return nil for GetSecuritySchemes")
	assert.Nil(t, c.GetLinks(), "nil Components should return nil for GetLinks")
	assert.Nil(t, c.GetCallbacks(), "nil Components should return nil for GetCallbacks")
	assert.Nil(t, c.GetPathItems(), "nil Components should return nil for GetPathItems")
	require.NotNil(t, c.GetExtensions(), "nil Components should return empty extensions")
}

func TestEncoding_GetExtensions_Nil(t *testing.T) {
	t.Parallel()

	var e *openapi.Encoding
	exts := e.GetExtensions()
	require.NotNil(t, exts, "nil Encoding should return empty extensions")
}

func TestExample_Nil(t *testing.T) {
	t.Parallel()

	var e *openapi.Example

	assert.Empty(t, e.GetSummary(), "nil Example should return empty string for GetSummary")
	assert.Empty(t, e.GetDescription(), "nil Example should return empty string for GetDescription")
	assert.Nil(t, e.GetValue(), "nil Example should return nil for GetValue")
	assert.Empty(t, e.GetExternalValue(), "nil Example should return empty string for GetExternalValue")
	assert.Nil(t, e.GetDataValue(), "nil Example should return nil for GetDataValue")
	require.NotNil(t, e.GetExtensions(), "nil Example should return empty extensions")
}

func TestHeader_Nil(t *testing.T) {
	t.Parallel()

	var h *openapi.Header

	assert.Empty(t, h.GetDescription(), "nil Header should return empty string for GetDescription")
	assert.False(t, h.GetRequired(), "nil Header should return false for GetRequired")
	assert.False(t, h.GetDeprecated(), "nil Header should return false for GetDeprecated")
	require.NotNil(t, h.GetExtensions(), "nil Header should return empty extensions")
}

func TestMediaType_Nil(t *testing.T) {
	t.Parallel()

	var m *openapi.MediaType

	assert.Nil(t, m.GetSchema(), "nil MediaType should return nil for GetSchema")
	assert.Nil(t, m.GetExample(), "nil MediaType should return nil for GetExample")
	assert.Nil(t, m.GetExamples(), "nil MediaType should return nil for GetExamples")
	assert.Nil(t, m.GetEncoding(), "nil MediaType should return nil for GetEncoding")
	require.NotNil(t, m.GetExtensions(), "nil MediaType should return empty extensions")
}

func TestOperation_Nil(t *testing.T) {
	t.Parallel()

	var o *openapi.Operation

	assert.Nil(t, o.GetTags(), "nil Operation should return nil for GetTags")
	assert.Empty(t, o.GetSummary(), "nil Operation should return empty string for GetSummary")
	assert.Empty(t, o.GetDescription(), "nil Operation should return empty string for GetDescription")
	assert.Nil(t, o.GetExternalDocs(), "nil Operation should return nil for GetExternalDocs")
	assert.Empty(t, o.GetOperationID(), "nil Operation should return empty string for GetOperationID")
	assert.Nil(t, o.GetParameters(), "nil Operation should return nil for GetParameters")
	assert.Nil(t, o.GetRequestBody(), "nil Operation should return nil for GetRequestBody")
	assert.Nil(t, o.GetResponses(), "nil Operation should return nil for GetResponses")
	assert.Nil(t, o.GetCallbacks(), "nil Operation should return nil for GetCallbacks")
	assert.False(t, o.GetDeprecated(), "nil Operation should return false for GetDeprecated")
	assert.Nil(t, o.GetSecurity(), "nil Operation should return nil for GetSecurity")
	assert.Nil(t, o.GetServers(), "nil Operation should return nil for GetServers")
	require.NotNil(t, o.GetExtensions(), "nil Operation should return empty extensions")
}

func TestParameter_Nil(t *testing.T) {
	t.Parallel()

	var p *openapi.Parameter

	assert.Empty(t, p.GetName(), "nil Parameter should return empty string for GetName")
	assert.Empty(t, p.GetIn(), "nil Parameter should return empty for GetIn")
	assert.Empty(t, p.GetDescription(), "nil Parameter should return empty string for GetDescription")
	assert.False(t, p.GetRequired(), "nil Parameter should return false for GetRequired")
	assert.False(t, p.GetDeprecated(), "nil Parameter should return false for GetDeprecated")
	assert.False(t, p.GetAllowEmptyValue(), "nil Parameter should return false for GetAllowEmptyValue")
	require.NotNil(t, p.GetExtensions(), "nil Parameter should return empty extensions")
}

func TestRequestBody_Nil(t *testing.T) {
	t.Parallel()

	var r *openapi.RequestBody

	assert.Empty(t, r.GetDescription(), "nil RequestBody should return empty string for GetDescription")
	assert.Nil(t, r.GetContent(), "nil RequestBody should return nil for GetContent")
	assert.False(t, r.GetRequired(), "nil RequestBody should return false for GetRequired")
}

func TestResponse_Nil(t *testing.T) {
	t.Parallel()

	var r *openapi.Response

	assert.Empty(t, r.GetDescription(), "nil Response should return empty string for GetDescription")
	assert.Nil(t, r.GetHeaders(), "nil Response should return nil for GetHeaders")
	assert.Nil(t, r.GetContent(), "nil Response should return nil for GetContent")
	assert.Nil(t, r.GetLinks(), "nil Response should return nil for GetLinks")
	require.NotNil(t, r.GetExtensions(), "nil Response should return empty extensions")
}

func TestResponses_GetExtensions_Nil(t *testing.T) {
	t.Parallel()

	var r *openapi.Responses
	exts := r.GetExtensions()
	require.NotNil(t, exts, "nil Responses should return empty extensions")
}

func TestServer_Nil(t *testing.T) {
	t.Parallel()

	var s *openapi.Server

	assert.Empty(t, s.GetURL(), "nil Server should return empty string for GetURL")
	assert.Empty(t, s.GetDescription(), "nil Server should return empty string for GetDescription")
	assert.Nil(t, s.GetVariables(), "nil Server should return nil for GetVariables")
	require.NotNil(t, s.GetExtensions(), "nil Server should return empty extensions")
}

func TestServerVariable_Nil(t *testing.T) {
	t.Parallel()

	var s *openapi.ServerVariable

	assert.Nil(t, s.GetEnum(), "nil ServerVariable should return nil for GetEnum")
	assert.Empty(t, s.GetDefault(), "nil ServerVariable should return empty string for GetDefault")
	assert.Empty(t, s.GetDescription(), "nil ServerVariable should return empty string for GetDescription")
}

func TestTag_Nil(t *testing.T) {
	t.Parallel()

	var tag *openapi.Tag

	assert.Empty(t, tag.GetName(), "nil Tag should return empty string for GetName")
	assert.Empty(t, tag.GetDescription(), "nil Tag should return empty string for GetDescription")
	assert.Nil(t, tag.GetExternalDocs(), "nil Tag should return nil for GetExternalDocs")
	require.NotNil(t, tag.GetExtensions(), "nil Tag should return empty extensions")
}

func TestSecurityScheme_Nil(t *testing.T) {
	t.Parallel()

	var s *openapi.SecurityScheme

	assert.Empty(t, s.GetType(), "nil SecurityScheme should return empty for GetType")
	assert.Empty(t, s.GetDescription(), "nil SecurityScheme should return empty string for GetDescription")
	assert.Empty(t, s.GetName(), "nil SecurityScheme should return empty string for GetName")
	assert.Empty(t, s.GetIn(), "nil SecurityScheme should return empty for GetIn")
	assert.Empty(t, s.GetScheme(), "nil SecurityScheme should return empty string for GetScheme")
	assert.Empty(t, s.GetBearerFormat(), "nil SecurityScheme should return empty string for GetBearerFormat")
	assert.Nil(t, s.GetFlows(), "nil SecurityScheme should return nil for GetFlows")
	assert.Empty(t, s.GetOpenIdConnectUrl(), "nil SecurityScheme should return empty string for GetOpenIdConnectUrl")
	assert.Empty(t, s.GetOAuth2MetadataUrl(), "nil SecurityScheme should return empty string for GetOAuth2MetadataUrl")
	assert.False(t, s.GetDeprecated(), "nil SecurityScheme should return false for GetDeprecated")
	require.NotNil(t, s.GetExtensions(), "nil SecurityScheme should return empty extensions")
}

func TestOAuthFlows_Nil(t *testing.T) {
	t.Parallel()

	var o *openapi.OAuthFlows

	assert.Nil(t, o.GetImplicit(), "nil OAuthFlows should return nil for GetImplicit")
	assert.Nil(t, o.GetPassword(), "nil OAuthFlows should return nil for GetPassword")
	assert.Nil(t, o.GetClientCredentials(), "nil OAuthFlows should return nil for GetClientCredentials")
	assert.Nil(t, o.GetAuthorizationCode(), "nil OAuthFlows should return nil for GetAuthorizationCode")
	assert.Nil(t, o.GetDeviceAuthorization(), "nil OAuthFlows should return nil for GetDeviceAuthorization")
	require.NotNil(t, o.GetExtensions(), "nil OAuthFlows should return empty extensions")
}

func TestOAuthFlow_Nil(t *testing.T) {
	t.Parallel()

	var o *openapi.OAuthFlow

	assert.Empty(t, o.GetAuthorizationURL(), "nil OAuthFlow should return empty string for GetAuthorizationURL")
	assert.Empty(t, o.GetDeviceAuthorizationURL(), "nil OAuthFlow should return empty string for GetDeviceAuthorizationURL")
	assert.Empty(t, o.GetTokenURL(), "nil OAuthFlow should return empty string for GetTokenURL")
	assert.Empty(t, o.GetRefreshURL(), "nil OAuthFlow should return empty string for GetRefreshURL")
	assert.Nil(t, o.GetScopes(), "nil OAuthFlow should return nil for GetScopes")
	require.NotNil(t, o.GetExtensions(), "nil OAuthFlow should return empty extensions")
}
