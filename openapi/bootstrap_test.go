package openapi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestBootstrap_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create bootstrap document
	doc := openapi.Bootstrap()

	// Marshal to YAML
	var buf bytes.Buffer
	ctx = yml.ContextWithConfig(ctx, &yml.Config{
		ValueStringStyle: yaml.DoubleQuotedStyle,
		Indentation:      2,
		OutputFormat:     yml.OutputFormatYAML,
	})
	err := openapi.Marshal(ctx, doc, &buf)
	require.NoError(t, err, "should marshal without error")

	// Read expected output
	expectedBytes, err := os.ReadFile("testdata/bootstrap_expected.yaml")
	require.NoError(t, err, "should read expected file")

	// Compare outputs
	expected := string(expectedBytes)
	actual := buf.String()

	assert.Equal(t, expected, actual, "marshaled output should match expected YAML")
}
