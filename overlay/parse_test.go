package overlay_test

import (
	"os"
	"testing"

	"github.com/speakeasy-api/jsonpath/pkg/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	err := overlay.Format("testdata/overlay.yaml")
	require.NoError(t, err)
	o, err := overlay.Parse("testdata/overlay.yaml")
	require.NoError(t, err)
	assert.NotNil(t, o)
	expect, err := os.ReadFile("testdata/overlay.yaml")
	require.NoError(t, err)

	actual, err := o.ToString()
	require.NoError(t, err)
	assert.Equal(t, string(expect), actual)

}
