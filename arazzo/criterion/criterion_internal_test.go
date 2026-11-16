package criterion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowedVersions_UnknownType_Success(t *testing.T) {
	t.Parallel()

	assert.Nil(t, allowedVersions("unknown", true), "unknown types should have no allowed versions")
}
