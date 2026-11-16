package oas3

import (
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestDialectValidationErrors_NoError_Success(t *testing.T) {
	t.Parallel()

	errs := dialectValidationErrors(nil, &core.Schema{})
	assert.Nil(t, errs, "should return no errors when the validator succeeds")
}

func TestDialectValidationErrors_NonValidationError_Error(t *testing.T) {
	t.Parallel()

	coreSchema := &core.Schema{}
	coreSchema.RootNode = &yaml.Node{Kind: yaml.MappingNode, Line: 2, Column: 1}

	errs := dialectValidationErrors(errors.New("unexpected failure"), coreSchema)
	require.Len(t, errs, 1, "should report a single error")
	assert.Equal(t, "[2:1] error validation-invalid-schema schema invalid: unexpected failure", errs[0].Error(), "should report the error against the schema root")
}
