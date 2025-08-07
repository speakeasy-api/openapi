package openapi_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAllReferences_Success(t *testing.T) {
	t.Parallel()

	absPath, err := filepath.Abs("testdata/resolve_test/main.yaml")
	require.NoError(t, err)

	f, err := os.Open(absPath)
	require.NoError(t, err)

	ctx := context.Background()

	o, vErrs, err := openapi.Unmarshal(ctx, f)
	require.NoError(t, err)
	require.Empty(t, vErrs)
	require.NotNil(t, o)

	validationErrs, errs := o.ResolveAllReferences(ctx, openapi.ResolveAllOptions{
		OpenAPILocation: absPath,
	})
	require.Empty(t, errs)
	require.Empty(t, validationErrs)

	// Assert that we can get the objects which should be already resolved references
	getUserOp := o.Paths.GetOrZero("/users/{userId}").MustGetObject().Get()
	require.NotNil(t, getUserOp)

	assert.True(t, getUserOp.Parameters[0].IsReference())
	getUserOpParam0 := getUserOp.Parameters[0].GetObject()
	require.NotNil(t, getUserOpParam0)
	assert.Equal(t, "userId", getUserOpParam0.GetName())

	assert.True(t, getUserOp.GetResponses().GetOrZero("200").IsReference())
	getUserOp200Resp := getUserOp.GetResponses().GetOrZero("200").GetObject()
	require.NotNil(t, getUserOp200Resp)
	assert.Equal(t, "User response", getUserOp200Resp.GetDescription())

	createUserOp := o.Paths.GetOrZero("/users").MustGetObject().Post()
	require.NotNil(t, createUserOp)

	assert.True(t, createUserOp.GetRequestBody().IsReference())
	createUserOpReqBody := createUserOp.GetRequestBody().GetObject()
	require.NotNil(t, createUserOpReqBody)
	assert.Equal(t, "User data", createUserOpReqBody.GetDescription())

	assert.True(t, createUserOp.GetResponses().GetOrZero("201").MustGetObject().GetContent().GetOrZero("application/json").GetSchema().IsReference())
	createUserOp201RespSchema := createUserOp.GetResponses().GetOrZero("201").MustGetObject().GetContent().GetOrZero("application/json").GetSchema().GetResolvedSchema()
	require.NotNil(t, createUserOp201RespSchema)
	require.True(t, createUserOp201RespSchema.IsLeft())
	assert.NotNil(t, createUserOp201RespSchema.GetLeft().GetProperties().GetOrZero("id"))
}

func TestResolveAllReferences_Error(t *testing.T) {
	t.Parallel()

	absPath, err := filepath.Abs("testdata/resolve_test/circular.yaml")
	require.NoError(t, err)

	f, err := os.Open(absPath)
	require.NoError(t, err)

	ctx := context.Background()

	o, vErrs, err := openapi.Unmarshal(ctx, f)
	require.NoError(t, err)
	require.Empty(t, vErrs)
	require.NotNil(t, o)

	validationErrs, err := o.ResolveAllReferences(ctx, openapi.ResolveAllOptions{
		OpenAPILocation: absPath,
	})
	require.Empty(t, validationErrs)
	require.Error(t, err)
	require.Regexp(t, `circular reference detected: .*circular\.yaml#/components/schemas/CircularSchema -> .*circular\.yaml#/components/schemas/IntermediateSchema -> .*circular\.yaml#/components/schemas/CircularSchema`, err.Error())
}
