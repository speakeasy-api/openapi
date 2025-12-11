package swagger_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaths_GetExtensions_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    x-path-ext: path-value
    get:
      responses:
        "200":
          description: Success
  x-custom: value
`
	doc, validationErrs, err := swagger.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should have no validation errors")

	assert.NotNil(t, doc.Paths.GetExtensions(), "GetExtensions should return non-nil")
}

func TestPaths_GetExtensions_Nil(t *testing.T) {
	t.Parallel()

	var paths *swagger.Paths
	ext := paths.GetExtensions()
	assert.NotNil(t, ext, "GetExtensions should return empty extensions for nil paths")
}

func TestPathItem_NewPathItem(t *testing.T) {
	t.Parallel()

	pi := swagger.NewPathItem()
	assert.NotNil(t, pi, "NewPathItem should return non-nil")
}

func TestPathItem_GetRef_Success(t *testing.T) {
	t.Parallel()

	pi := &swagger.PathItem{
		Ref: pointer.From("#/paths/~1other"),
	}

	assert.Equal(t, "#/paths/~1other", pi.GetRef(), "GetRef should return correct value")
}

func TestPathItem_GetRef_Nil(t *testing.T) {
	t.Parallel()

	var pi *swagger.PathItem
	assert.Empty(t, pi.GetRef(), "GetRef should return empty string for nil")

	pi2 := &swagger.PathItem{}
	assert.Empty(t, pi2.GetRef(), "GetRef should return empty string when Ref is nil")
}

func TestPathItem_GetParameters_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{id}:
    parameters:
      - name: id
        in: path
        required: true
        type: string
    get:
      responses:
        "200":
          description: Success
`
	doc, _, err := swagger.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")

	pathItem, ok := doc.Paths.Get("/users/{id}")
	require.True(t, ok, "path should exist")
	assert.NotNil(t, pathItem.GetParameters(), "GetParameters should return non-nil")
	assert.Len(t, pathItem.GetParameters(), 1, "should have one parameter")
}

func TestPathItem_GetParameters_Nil(t *testing.T) {
	t.Parallel()

	var pi *swagger.PathItem
	assert.Nil(t, pi.GetParameters(), "GetParameters should return nil for nil pathitem")
}

func TestPathItem_GetExtensions_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    x-custom: value
    get:
      responses:
        "200":
          description: Success
`
	doc, _, err := swagger.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")

	pathItem, ok := doc.Paths.Get("/users")
	require.True(t, ok, "path should exist")
	assert.NotNil(t, pathItem.GetExtensions(), "GetExtensions should return non-nil")
}

func TestPathItem_GetExtensions_Nil(t *testing.T) {
	t.Parallel()

	var pi *swagger.PathItem
	ext := pi.GetExtensions()
	assert.NotNil(t, ext, "GetExtensions should return empty extensions for nil pathitem")
}

func TestPathItem_Operations_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
    put:
      operationId: updateUsers
      responses:
        "200":
          description: Success
    post:
      operationId: createUsers
      responses:
        "200":
          description: Success
    delete:
      operationId: deleteUsers
      responses:
        "200":
          description: Success
    options:
      operationId: optionsUsers
      responses:
        "200":
          description: Success
    head:
      operationId: headUsers
      responses:
        "200":
          description: Success
    patch:
      operationId: patchUsers
      responses:
        "200":
          description: Success
`
	doc, validationErrs, err := swagger.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should have no validation errors")

	pathItem, ok := doc.Paths.Get("/users")
	require.True(t, ok, "path should exist")

	assert.NotNil(t, pathItem.Get(), "Get should return non-nil")
	assert.Equal(t, "getUsers", pathItem.Get().GetOperationID())
	assert.NotNil(t, pathItem.Put(), "Put should return non-nil")
	assert.Equal(t, "updateUsers", pathItem.Put().GetOperationID())
	assert.NotNil(t, pathItem.Post(), "Post should return non-nil")
	assert.Equal(t, "createUsers", pathItem.Post().GetOperationID())
	assert.NotNil(t, pathItem.Delete(), "Delete should return non-nil")
	assert.Equal(t, "deleteUsers", pathItem.Delete().GetOperationID())
	assert.NotNil(t, pathItem.Options(), "Options should return non-nil")
	assert.Equal(t, "optionsUsers", pathItem.Options().GetOperationID())
	assert.NotNil(t, pathItem.Head(), "Head should return non-nil")
	assert.Equal(t, "headUsers", pathItem.Head().GetOperationID())
	assert.NotNil(t, pathItem.Patch(), "Patch should return non-nil")
	assert.Equal(t, "patchUsers", pathItem.Patch().GetOperationID())
}

func TestPathItem_Operations_Nil(t *testing.T) {
	t.Parallel()

	var pi *swagger.PathItem
	assert.Nil(t, pi.Get(), "Get should return nil for nil pathitem")
	assert.Nil(t, pi.Put(), "Put should return nil for nil pathitem")
	assert.Nil(t, pi.Post(), "Post should return nil for nil pathitem")
	assert.Nil(t, pi.Delete(), "Delete should return nil for nil pathitem")
	assert.Nil(t, pi.Options(), "Options should return nil for nil pathitem")
	assert.Nil(t, pi.Head(), "Head should return nil for nil pathitem")
	assert.Nil(t, pi.Patch(), "Patch should return nil for nil pathitem")
}
