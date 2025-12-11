package swagger

import (
	"testing"

	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
)

func TestNewReferencedParameterFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/parameters/MyParam")
	result := NewReferencedParameterFromRef(ref)

	assert.NotNil(t, result, "NewReferencedParameterFromRef should return non-nil")
	assert.NotNil(t, result.Reference, "Reference field should be set")
	assert.Equal(t, ref, *result.Reference, "Reference should match")
	assert.Nil(t, result.Object, "Object should be nil for reference")
}

func TestNewReferencedParameterFromParameter_Success(t *testing.T) {
	t.Parallel()

	param := &Parameter{}
	result := NewReferencedParameterFromParameter(param)

	assert.NotNil(t, result, "NewReferencedParameterFromParameter should return non-nil")
	assert.Nil(t, result.Reference, "Reference should be nil for inline object")
	assert.Equal(t, param, result.Object, "Object should match")
}

func TestNewReferencedResponseFromRef_Success(t *testing.T) {
	t.Parallel()

	ref := references.Reference("#/responses/MyResponse")
	result := NewReferencedResponseFromRef(ref)

	assert.NotNil(t, result, "NewReferencedResponseFromRef should return non-nil")
	assert.NotNil(t, result.Reference, "Reference field should be set")
	assert.Equal(t, ref, *result.Reference, "Reference should match")
	assert.Nil(t, result.Object, "Object should be nil for reference")
}

func TestNewReferencedResponseFromResponse_Success(t *testing.T) {
	t.Parallel()

	resp := &Response{}
	result := NewReferencedResponseFromResponse(resp)

	assert.NotNil(t, result, "NewReferencedResponseFromResponse should return non-nil")
	assert.Nil(t, result.Reference, "Reference should be nil for inline object")
	assert.Equal(t, resp, result.Object, "Object should match")
}
