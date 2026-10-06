package overlay_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/overlay.yaml")
	require.NoError(t, err, "overlay fixture should load")
	path := filepath.Join(t.TempDir(), "overlay.yaml")
	require.NoError(t, os.WriteFile(path, fixture, 0o600), "private overlay fixture should be created")

	err = overlay.Format(path)
	require.NoError(t, err)
	o, err := overlay.Parse(path)
	require.NoError(t, err)
	assert.NotNil(t, o)
	expect, err := os.ReadFile(path)
	require.NoError(t, err)

	actual, err := o.ToString()
	require.NoError(t, err)
	assert.Equal(t, string(expect), actual)

}
