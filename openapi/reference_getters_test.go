package openapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
)

func TestReference_GetResolvedObject_Nil_Success(t *testing.T) {
	t.Parallel()

	var ref *openapi.ReferencedPathItem
	result := ref.GetResolvedObject()

	assert.Nil(t, result, "GetResolvedObject on nil should return nil")
}

func TestReference_GetResolvedObject_InlineObject_Success(t *testing.T) {
	t.Parallel()

	pathItem := &openapi.PathItem{
		Summary: pointer.From("Test path item"),
	}
	ref := openapi.NewReferencedPathItemFromPathItem(pathItem)
	result := ref.GetResolvedObject()

	assert.Equal(t, pathItem, result, "GetResolvedObject should return the inline object")
}

func TestReference_GetResolvedObject_UnresolvedRef_Success(t *testing.T) {
	t.Parallel()

	refStr := references.Reference("#/components/pathItems/UnresolvedPath")
	ref := openapi.NewReferencedPathItemFromRef(refStr)
	result := ref.GetResolvedObject()

	assert.Nil(t, result, "GetResolvedObject on unresolved ref should return nil")
}
