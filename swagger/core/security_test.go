package core_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecurityRequirement_Success(t *testing.T) {
	t.Parallel()

	secReq := core.NewSecurityRequirement()

	require.NotNil(t, secReq, "NewSecurityRequirement should not return nil")
	require.NotNil(t, secReq.Map, "Map should be initialized")
	assert.Equal(t, 0, secReq.Len(), "new security requirement should be empty")
}
