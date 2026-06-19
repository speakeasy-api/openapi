package oq

import (
	"testing"

	"github.com/speakeasy-api/openapi/oq/expr"
	"github.com/stretchr/testify/assert"
)

func TestToonValue_ArrayEscapesSemicolonElements(t *testing.T) {
	t.Parallel()

	value := expr.ArrayVal([]string{"v1;deprecated", "v2;current"})

	encoded := toonValue(value)

	assert.Equal(t, `"v1;deprecated";"v2;current"`, encoded, "array elements containing the delimiter should be quoted individually")
}
