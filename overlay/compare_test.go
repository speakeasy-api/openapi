package overlay_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
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

func TestCompare_ArrayAppendMultiple(t *testing.T) {
	t.Parallel()

	original := `items:
  - name: a
  - name: b
`
	target := `items:
  - name: a
  - name: b
  - name: c
  - name: d
`
	var origNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(original), &origNode))
	var targetNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(target), &targetNode))

	o, err := overlay.Compare("test", &origNode, targetNode)
	require.NoError(t, err)

	// Emits a single append update for the new tail elements
	require.Len(t, o.Actions, 1)
	assert.Equal(t, `$["items"]`, o.Actions[0].Target)
	assert.False(t, o.Actions[0].Remove)
	require.Len(t, o.Actions[0].Update.Content, 2)

	// Round-trip: applying the overlay should produce the target
	err = o.ApplyTo(&origNode)
	require.NoError(t, err)

	var expected yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(target), &expected))

	actualYAML, err := yaml.Marshal(&origNode)
	require.NoError(t, err)
	expectedYAML, err := yaml.Marshal(&expected)
	require.NoError(t, err)
	assert.Equal(t, string(expectedYAML), string(actualYAML))
}

func TestCompare_OpenAPIScalarTypes_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, beforeType, afterType, beforeValue, afterValue string
	}{
		{"boolean to string", "boolean", "string", "false", `"false"`},
		{"integer to string", "integer", "string", "0", `"0"`},
		{"null to string", `"null"`, "string", "null", `"null"`},
		{"string to boolean", "string", "boolean", `"enabled"`, "true"},
		{"integer to different string", "integer", "string", "0", `"1"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			const document = `openapi: 3.1.0
info: {title: Settings API, version: 1.0.0}
paths:
  /settings:
    get:
      responses:
        '200':
          description: Current setting
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Setting'
components:
  schemas:
    Setting:
      type: %s
      default: %s
      enum: [%s]
`
			before, err := loader.LoadSpecificationFromReader(strings.NewReader(fmt.Sprintf(document, tt.beforeType, tt.beforeValue, tt.beforeValue)))
			require.NoError(t, err, "before OpenAPI document should load")
			after, err := loader.LoadSpecificationFromReader(strings.NewReader(fmt.Sprintf(document, tt.afterType, tt.afterValue, tt.afterValue)))
			require.NoError(t, err, "after OpenAPI document should load")
			change, err := overlay.Compare("Migrate setting type", before, *after)
			require.NoError(t, err, "comparison should succeed")
			serialized, err := change.ToString()
			require.NoError(t, err, "generated overlay should serialize")
			parsed, err := overlay.ParseReader(strings.NewReader(serialized))
			require.NoError(t, err, "generated overlay should parse")
			require.NoError(t, parsed.ApplyTo(before), "generated overlay should apply")
			encoded, err := yaml.Marshal(before)
			require.NoError(t, err, "transformed OpenAPI document should serialize")
			var actual, expected any
			require.NoError(t, yaml.Unmarshal(encoded, &actual), "transformed OpenAPI document should decode")
			require.NoError(t, after.Decode(&expected), "target OpenAPI document should decode")
			assert.Equal(t, expected, actual, "default and enum must retain the target JSON types")
		})
	}
}
