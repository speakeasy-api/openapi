package openapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReferencedPathItemFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/pathItems/MyPath")
	result := openapi.NewReferencedPathItemFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
	assert.Nil(t, result.GetObject(), "object should be nil for unresolved reference")
}

func TestNewReferencedPathItemFromPathItem_Success(t *testing.T) {
	t.Parallel()

	pathItem := &openapi.PathItem{
		Summary: pointer.From("Test path item"),
	}
	result := openapi.NewReferencedPathItemFromPathItem(pathItem)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, pathItem, result.GetObject(), "object should match")
}

func TestNewReferencedExampleFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/examples/MyExample")
	result := openapi.NewReferencedExampleFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
}

func TestNewReferencedExampleFromExample_Success(t *testing.T) {
	t.Parallel()

	example := &openapi.Example{
		Summary: pointer.From("Test example"),
	}
	result := openapi.NewReferencedExampleFromExample(example)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, example, result.GetObject(), "object should match")
}

func TestNewReferencedParameterFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/parameters/MyParam")
	result := openapi.NewReferencedParameterFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
}

func TestNewReferencedParameterFromParameter_Success(t *testing.T) {
	t.Parallel()

	param := &openapi.Parameter{
		Name: "testParam",
		In:   openapi.ParameterInQuery,
	}
	result := openapi.NewReferencedParameterFromParameter(param)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, param, result.GetObject(), "object should match")
}

func TestNewReferencedHeaderFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/headers/MyHeader")
	result := openapi.NewReferencedHeaderFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
}

func TestNewReferencedHeaderFromHeader_Success(t *testing.T) {
	t.Parallel()

	header := &openapi.Header{
		Description: pointer.From("Test header"),
	}
	result := openapi.NewReferencedHeaderFromHeader(header)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, header, result.GetObject(), "object should match")
}

func TestNewReferencedRequestBodyFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/requestBodies/MyBody")
	result := openapi.NewReferencedRequestBodyFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
}

func TestNewReferencedRequestBodyFromRequestBody_Success(t *testing.T) {
	t.Parallel()

	body := &openapi.RequestBody{
		Description: pointer.From("Test request body"),
	}
	result := openapi.NewReferencedRequestBodyFromRequestBody(body)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, body, result.GetObject(), "object should match")
}

func TestNewReferencedResponseFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/responses/MyResponse")
	result := openapi.NewReferencedResponseFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
}

func TestNewReferencedResponseFromResponse_Success(t *testing.T) {
	t.Parallel()

	response := &openapi.Response{
		Description: "Test response",
	}
	result := openapi.NewReferencedResponseFromResponse(response)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, response, result.GetObject(), "object should match")
}

func TestNewReferencedCallbackFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/callbacks/MyCallback")
	result := openapi.NewReferencedCallbackFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
}

func TestNewReferencedCallbackFromCallback_Success(t *testing.T) {
	t.Parallel()

	callback := &openapi.Callback{}
	result := openapi.NewReferencedCallbackFromCallback(callback)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, callback, result.GetObject(), "object should match")
}

func TestNewReferencedLinkFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/links/MyLink")
	result := openapi.NewReferencedLinkFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
}

func TestNewReferencedLinkFromLink_Success(t *testing.T) {
	t.Parallel()

	link := &openapi.Link{
		Description: pointer.From("Test link"),
	}
	result := openapi.NewReferencedLinkFromLink(link)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, link, result.GetObject(), "object should match")
}

func TestNewReferencedSecuritySchemeFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/components/securitySchemes/MyScheme")
	result := openapi.NewReferencedSecuritySchemeFromRef(ref)

	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.IsReference(), "should be a reference")
	assert.Equal(t, ref, result.GetReference(), "reference should match")
}

func TestNewReferencedSecuritySchemeFromSecurityScheme_Success(t *testing.T) {
	t.Parallel()

	scheme := &openapi.SecurityScheme{
		Type: openapi.SecuritySchemeTypeAPIKey,
	}
	result := openapi.NewReferencedSecuritySchemeFromSecurityScheme(scheme)

	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsReference(), "should not be a reference")
	assert.Equal(t, scheme, result.GetObject(), "object should match")
}
