package overlay_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompare(t *testing.T) {
	t.Parallel()

	node, err := loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)
	node2, err := loader.LoadSpecification("testdata/openapi-overlayed.yaml")
	require.NoError(t, err)

	o, err := loader.LoadOverlay("testdata/overlay-generated.yaml")
	require.NoError(t, err)

	o2, err := overlay.Compare("Drinks Overlay", node, *node2)
	require.NoError(t, err)

	o1s, err := o.ToString()
	require.NoError(t, err)
	o2s, err := o2.ToString()
	require.NoError(t, err)

	// Uncomment this if we've improved the output
	// os.WriteFile("testdata/overlay-generated.yaml", []byte(o2s), 0644)
	assert.Equal(t, o1s, o2s)

	// round trip it
	err = o.ApplyTo(node)
	require.NoError(t, err)
	NodeMatchesFile(t, node, "testdata/openapi-overlayed.yaml")

}
