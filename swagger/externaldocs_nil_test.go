package swagger_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalDocumentation_GetDescription_Nil(t *testing.T) {
	t.Parallel()

	var extDoc *swagger.ExternalDocumentation
	assert.Empty(t, extDoc.GetDescription(), "nil ExternalDocumentation should return empty string for GetDescription")
}

func TestExternalDocumentation_GetURL_Nil(t *testing.T) {
	t.Parallel()

	var extDoc *swagger.ExternalDocumentation
	assert.Empty(t, extDoc.GetURL(), "nil ExternalDocumentation should return empty string for GetURL")
}

func TestExternalDocumentation_GetExtensions_Nil(t *testing.T) {
	t.Parallel()

	var extDoc *swagger.ExternalDocumentation
	exts := extDoc.GetExtensions()
	require.NotNil(t, exts, "nil ExternalDocumentation should return empty extensions for GetExtensions")
}

func TestInfo_GetTitle_Nil(t *testing.T) {
	t.Parallel()

	var info *swagger.Info
	assert.Empty(t, info.GetTitle(), "nil Info should return empty string for GetTitle")
}

func TestInfo_GetDescription_Nil(t *testing.T) {
	t.Parallel()

	var info *swagger.Info
	assert.Empty(t, info.GetDescription(), "nil Info should return empty string for GetDescription")
}
