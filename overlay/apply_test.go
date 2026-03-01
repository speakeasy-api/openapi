package overlay_test

import (
	"bytes"
	"os"
	"strconv"
	"testing"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// NodeMatchesFile is a test that marshals the YAML file from the given node,
// then compares those bytes to those found in the expected file.
func NodeMatchesFile(
	t *testing.T,
	actual *yaml.Node,
	expectedFile string,
	msgAndArgs ...any,
) {
	t.Helper()
	variadoc := func(pre ...any) []any { return append(msgAndArgs, pre...) }

	var actualBuf bytes.Buffer
	enc := yaml.NewEncoder(&actualBuf)
	enc.SetIndent(2)
	err := enc.Encode(actual)
	require.NoError(t, err, variadoc("failed to marshal node: ")...)

	expectedBytes, err := os.ReadFile(expectedFile)
	require.NoError(t, err, variadoc("failed to read expected file: ")...)

	// lazy redo snapshot
	// os.WriteFile(expectedFile, actualBuf.Bytes(), 0644)

	// t.Log("### EXPECT START ###\n" + string(expectedBytes) + "\n### EXPECT END ###\n")
	// t.Log("### ACTUAL START ###\n" + actualBuf.string() + "\n### ACTUAL END ###\n")

	assert.Equal(t, string(expectedBytes), actualBuf.String(), variadoc("node does not match expected file: ")...)
}

func TestApplyTo(t *testing.T) {
	t.Parallel()

	node, err := loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)

	o, err := loader.LoadOverlay("testdata/overlay.yaml")
	require.NoError(t, err)

	err = o.ApplyTo(node)
	require.NoError(t, err)

	NodeMatchesFile(t, node, "testdata/openapi-overlayed.yaml")
}

func TestApplyToStrict(t *testing.T) {
	t.Parallel()

	node, err := loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)

	o, err := loader.LoadOverlay("testdata/overlay-mismatched.yaml")
	require.NoError(t, err)

	warnings, err := o.ApplyToStrict(node)
	require.Error(t, err, "error applying overlay (strict): selector \"$.unknown-attribute\" did not match any targets")
	assert.Len(t, warnings, 2)
	o.Actions = o.Actions[1:]
	node, err = loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)

	warnings, err = o.ApplyToStrict(node)
	require.NoError(t, err)
	assert.Len(t, warnings, 1)
	assert.Equal(t, "update action (2 / 2) target=$.info.title: does nothing", warnings[0])
	NodeMatchesFile(t, node, "testdata/openapi-strict-onechange.yaml")

	node, err = loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)

	o, err = loader.LoadOverlay("testdata/overlay.yaml")
	require.NoError(t, err)

	err = o.ApplyTo(node)
	require.NoError(t, err)

	NodeMatchesFile(t, node, "testdata/openapi-overlayed.yaml")

}

func BenchmarkApplyToStrict(b *testing.B) {
	openAPIBytes, err := os.ReadFile("testdata/openapi.yaml")
	require.NoError(b, err)
	overlayBytes, err := os.ReadFile("testdata/overlay-zero-change.yaml")
	require.NoError(b, err)

	var specNode yaml.Node
	err = yaml.NewDecoder(bytes.NewReader(openAPIBytes)).Decode(&specNode)
	require.NoError(b, err)

	// Load overlay from bytes
	var o overlay.Overlay
	err = yaml.NewDecoder(bytes.NewReader(overlayBytes)).Decode(&o)
	require.NoError(b, err)

	// Apply overlay to spec
	for b.Loop() {
		_, _ = o.ApplyToStrict(&specNode)
	}
}

func BenchmarkApplyToStrictBySize(b *testing.B) {
	// Read the base OpenAPI spec
	openAPIBytes, err := os.ReadFile("testdata/openapi.yaml")
	require.NoError(b, err)

	// Read the overlay spec
	overlayBytes, err := os.ReadFile("testdata/overlay-zero-change.yaml")
	require.NoError(b, err)

	// Decode the base spec
	var baseSpec yaml.Node
	err = yaml.NewDecoder(bytes.NewReader(openAPIBytes)).Decode(&baseSpec)
	require.NoError(b, err)

	// Find the paths node and a path to duplicate
	pathsNode := findPathsNode(&baseSpec)
	require.NotNil(b, pathsNode)

	// Get the first path item to use as template
	var templatePath *yaml.Node
	var templateKey string
	for i := 0; i < len(pathsNode.Content); i += 2 {
		if pathsNode.Content[i].Kind == yaml.ScalarNode && pathsNode.Content[i].Value[0] == '/' {
			templateKey = pathsNode.Content[i].Value
			templatePath = pathsNode.Content[i+1]
			break
		}
	}
	require.NotNil(b, templatePath)

	// Target sizes: 2KB, 20KB, 200KB, 2MB, 20MB
	targetSizes := []struct {
		size int
		name string
	}{
		{2 * 1024, "2KB"},
		{20 * 1024, "20KB"},
		{200 * 1024, "200KB"},
		{2000 * 1024, "2M"},
	}

	// Calculate the base document size
	var baseBuf bytes.Buffer
	enc := yaml.NewEncoder(&baseBuf)
	err = enc.Encode(&baseSpec)
	require.NoError(b, err)
	baseSize := baseBuf.Len()

	// Calculate the size of a single path item by encoding it
	var pathBuf bytes.Buffer
	pathEnc := yaml.NewEncoder(&pathBuf)
	tempNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: templateKey + "-test"},
			cloneNode(templatePath),
		},
	}
	err = pathEnc.Encode(tempNode)
	require.NoError(b, err)
	// Approximate size contribution of one path (accounting for YAML structure)
	pathItemSize := pathBuf.Len() - 10 // Subtract some overhead

	for _, target := range targetSizes {
		b.Run(target.name, func(b *testing.B) {
			// Create a copy of the base spec
			specCopy := cloneNode(&baseSpec)
			pathsNodeCopy := findPathsNode(specCopy)

			// Calculate how many paths we need to add
			bytesNeeded := target.size - baseSize
			pathsToAdd := 0
			if bytesNeeded > 0 {
				pathsToAdd = bytesNeeded / pathItemSize
				// Add a few extra to ensure we exceed the target
				pathsToAdd += 5
			}

			// Add the calculated number of path duplicates
			for i := 0; i < pathsToAdd; i++ {
				newPathKey := yaml.Node{Kind: yaml.ScalarNode, Value: templateKey + "-duplicate-" + strconv.Itoa(i)}
				newPathValue := cloneNode(templatePath)
				pathsNodeCopy.Content = append(pathsNodeCopy.Content, &newPathKey, newPathValue)
			}

			// Verify final size
			var finalBuf bytes.Buffer
			finalEnc := yaml.NewEncoder(&finalBuf)
			err = finalEnc.Encode(specCopy)
			require.NoError(b, err)
			actualSize := finalBuf.Len()
			b.Logf("OpenAPI size: %d bytes (target: %d, paths added: %d)", actualSize, target.size, pathsToAdd)

			// Load overlay
			var o overlay.Overlay
			err = yaml.NewDecoder(bytes.NewReader(overlayBytes)).Decode(&o)
			require.NoError(b, err)

			specForTest := cloneNode(specCopy)
			// Run the benchmark
			b.ResetTimer()
			for b.Loop() {
				_, _ = o.ApplyToStrict(specForTest)
			}
		})
	}
}

// Helper function to find the paths node in the OpenAPI spec
func findPathsNode(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	if node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == "paths" {
			return node.Content[i+1]
		}
	}
	return nil
}

// Helper function to deep clone a YAML node
func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	clone := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		Alias:       node.Alias,
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}

	if node.Content != nil {
		clone.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			clone.Content[i] = cloneNode(child)
		}
	}

	return clone
}

func TestApplyTo_CopyVersionToHeader(t *testing.T) {
	t.Parallel()

	node, err := loader.LoadSpecification("testdata/openapi-version-header.yaml")
	require.NoError(t, err)

	o, err := loader.LoadOverlay("testdata/overlay-version-header.yaml")
	require.NoError(t, err)

	err = o.ApplyTo(node)
	require.NoError(t, err)

	NodeMatchesFile(t, node, "testdata/openapi-version-header-expected.yaml")
}

// applyOverlay is a test helper that unmarshals the input/overlay YAML,
// applies the overlay (lax or strict), and returns the result.
func applyOverlay(t *testing.T, inputYAML, overlayYAML string) *yaml.Node {
	t.Helper()
	var specNode yaml.Node
	err := yaml.Unmarshal([]byte(inputYAML), &specNode)
	require.NoError(t, err, "unmarshal input spec should succeed")

	var o overlay.Overlay
	err = yaml.Unmarshal([]byte(overlayYAML), &o)
	require.NoError(t, err, "unmarshal overlay should succeed")

	err = o.ApplyTo(&specNode)
	require.NoError(t, err, "apply overlay should succeed")
	return &specNode
}

// applyOverlayStrict is a test helper that applies in strict mode.
func applyOverlayStrict(t *testing.T, inputYAML, overlayYAML string) ([]string, error) {
	t.Helper()
	var specNode yaml.Node
	err := yaml.Unmarshal([]byte(inputYAML), &specNode)
	require.NoError(t, err, "unmarshal input spec should succeed")

	var o overlay.Overlay
	err = yaml.Unmarshal([]byte(overlayYAML), &o)
	require.NoError(t, err, "unmarshal overlay should succeed")

	return o.ApplyToStrict(&specNode)
}

// marshalNode is a test helper that marshals a YAML node to a string.
func marshalNode(t *testing.T, node *yaml.Node) string {
	t.Helper()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(node)
	require.NoError(t, err, "encode result should succeed")
	return buf.String()
}

// --- 1.0.0 Tests: Preserve existing behavior ---

func TestApplyTo_V100_TypeMismatchReplace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputYAML    string
		overlayYAML  string
		expectedYAML string
	}{
		{
			name: "object replaces scalar",
			inputYAML: `root:
  key: value
`,
			overlayYAML: `overlay: 1.0.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root.key"
    update:
      nested: object
`,
			expectedYAML: `root:
  key:
    nested: object
`,
		},
		{
			name: "scalar replaces object",
			inputYAML: `root:
  key:
    nested: object
`,
			overlayYAML: `overlay: 1.0.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root.key"
    update: simple
`,
			expectedYAML: `root:
  key: simple
`,
		},
		{
			name: "array replaces object",
			inputYAML: `root:
  key:
    a: 1
`,
			overlayYAML: `overlay: 1.0.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root.key"
    update:
      - item1
      - item2
`,
			expectedYAML: `root:
  key:
    - item1
    - item2
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyOverlay(t, tt.inputYAML, tt.overlayYAML)
			assert.Equal(t, tt.expectedYAML, marshalNode(t, result), "output should match expected YAML")
		})
	}
}

func TestApplyTo_V100_ArrayConcatenation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputYAML    string
		overlayYAML  string
		expectedYAML string
	}{
		{
			name: "top-level array target with array update concatenates",
			inputYAML: `items:
  - a
  - b
`,
			overlayYAML: `overlay: 1.0.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.items"
    update:
      - c
      - d
`,
			expectedYAML: `items:
  - a
  - b
  - c
  - d
`,
		},
		{
			name: "nested array in object merge concatenates",
			inputYAML: `root:
  list:
    - existing
`,
			overlayYAML: `overlay: 1.0.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root"
    update:
      list:
        - new
`,
			expectedYAML: `root:
  list:
    - existing
    - new
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyOverlay(t, tt.inputYAML, tt.overlayYAML)
			assert.Equal(t, tt.expectedYAML, marshalNode(t, result), "output should match expected YAML")
		})
	}
}

// --- 1.1.0 Tests: New spec-compliant behavior ---

func TestApplyTo_V110_TopLevelArrayAppend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputYAML    string
		overlayYAML  string
		expectedYAML string
	}{
		{
			name: "object appended to array as single element",
			inputYAML: `tags:
  - name: existing
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.tags"
    update:
      name: newTag
      description: appended
`,
			expectedYAML: `tags:
  - name: existing
  - name: newTag
    description: appended
`,
		},
		{
			name: "scalar appended to array as single element",
			inputYAML: `tags:
  - pets
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.tags"
    update: admin
`,
			expectedYAML: `tags:
  - pets
  - admin
`,
		},
		{
			name: "array update concatenated with array target",
			inputYAML: `tags:
  - name: existing
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.tags"
    update:
      - name: new1
      - name: new2
`,
			expectedYAML: `tags:
  - name: existing
  - name: new1
  - name: new2
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyOverlay(t, tt.inputYAML, tt.overlayYAML)
			assert.Equal(t, tt.expectedYAML, marshalNode(t, result), "output should match expected YAML")
		})
	}
}

func TestApplyTo_V110_TopLevelObjectMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputYAML    string
		overlayYAML  string
		expectedYAML string
	}{
		{
			name: "new keys added to object",
			inputYAML: `info:
  title: Original
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.info"
    update:
      description: Added
`,
			expectedYAML: `info:
  title: Original
  description: Added
`,
		},
		{
			name: "existing key recursively merged",
			inputYAML: `info:
  title: Original
  contact:
    name: Old
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.info"
    update:
      contact:
        email: new@example.com
`,
			expectedYAML: `info:
  title: Original
  contact:
    name: Old
    email: new@example.com
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyOverlay(t, tt.inputYAML, tt.overlayYAML)
			assert.Equal(t, tt.expectedYAML, marshalNode(t, result), "output should match expected YAML")
		})
	}
}

func TestApplyTo_V110_TopLevelPrimitiveReplace(t *testing.T) {
	t.Parallel()

	inputYAML := `info:
  title: Original
`
	overlayYAML := `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.info.title"
    update: Replaced
`
	expectedYAML := `info:
  title: Replaced
`
	result := applyOverlay(t, inputYAML, overlayYAML)
	assert.Equal(t, expectedYAML, marshalNode(t, result), "scalar should be replaced")
}

func TestApplyTo_V110_RecursiveArrayConcat(t *testing.T) {
	t.Parallel()

	inputYAML := `paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
`
	overlayYAML := `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.paths['/pets'].get"
    update:
      parameters:
        - name: offset
          in: query
`
	expectedYAML := `paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
        - name: offset
          in: query
`
	result := applyOverlay(t, inputYAML, overlayYAML)
	assert.Equal(t, expectedYAML, marshalNode(t, result), "nested arrays should be concatenated during object merge")
}

func TestApplyTo_V110_RecursiveTypeMismatch_Strict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		inputYAML   string
		overlayYAML string
		errContains string
	}{
		{
			name: "array target with object update within recursive merge",
			inputYAML: `root:
  key:
    - item1
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root"
    update:
      key:
        nested: value
`,
			errContains: `key "key": type mismatch: target is array but update is object`,
		},
		{
			name: "scalar target with array update within recursive merge",
			inputYAML: `root:
  key: scalar
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root"
    update:
      key:
        - item1
`,
			errContains: `key "key": type mismatch: target is scalar but update is array`,
		},
		{
			name: "scalar target with object update within recursive merge",
			inputYAML: `root:
  key: scalar
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root"
    update:
      key:
        nested: value
`,
			errContains: `key "key": type mismatch: target is scalar but update is object`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := applyOverlayStrict(t, tt.inputYAML, tt.overlayYAML)
			require.Error(t, err, "strict mode should return error on recursive type mismatch")
			assert.Contains(t, err.Error(), tt.errContains, "error should describe the mismatch")
		})
	}
}

func TestApplyTo_V110_RecursiveTypeMismatch_Lax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputYAML    string
		overlayYAML  string
		expectedYAML string
	}{
		{
			name: "array target with object update replaces gracefully",
			inputYAML: `root:
  key:
    - item1
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root"
    update:
      key:
        nested: value
`,
			expectedYAML: `root:
  key:
    nested: value
`,
		},
		{
			name: "scalar target with object update replaces gracefully",
			inputYAML: `root:
  key: scalar
`,
			overlayYAML: `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root"
    update:
      key:
        nested: value
`,
			expectedYAML: `root:
  key:
    nested: value
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyOverlay(t, tt.inputYAML, tt.overlayYAML)
			assert.Equal(t, tt.expectedYAML, marshalNode(t, result), "lax mode should replace gracefully")
		})
	}
}

func TestApplyTo_V110_CopyArrayAppend(t *testing.T) {
	t.Parallel()

	inputYAML := `source:
  name: copied
  description: from source
targets:
  - name: existing
`
	overlayYAML := `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.targets"
    copy: "$.source"
`
	result := applyOverlay(t, inputYAML, overlayYAML)
	expectedYAML := `source:
  name: copied
  description: from source
targets:
  - name: existing
  - name: copied
    description: from source
`
	assert.Equal(t, expectedYAML, marshalNode(t, result), "copy into array should append the object")
}

func TestApplyTo_V110_HomogeneityCheck_Strict(t *testing.T) {
	t.Parallel()

	// This test uses a wildcard that selects nodes of different types.
	// In 1.1.0 strict mode, this should error.
	inputYAML := `root:
  arrayKey:
    - item1
  objectKey:
    nested: value
`
	overlayYAML := `overlay: 1.1.0
info:
  title: test
  version: 1.0.0
actions:
  - target: "$.root.*"
    update:
      newKey: newValue
`
	_, err := applyOverlayStrict(t, inputYAML, overlayYAML)
	require.Error(t, err, "strict mode should error on mixed node types")
	assert.Contains(t, err.Error(), "mixed node types", "error should mention mixed types")
}

func TestApplyTo_V100_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	node, err := loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)

	o, err := loader.LoadOverlay("testdata/overlay.yaml")
	require.NoError(t, err)

	err = o.ApplyTo(node)
	require.NoError(t, err)

	NodeMatchesFile(t, node, "testdata/openapi-overlayed.yaml", "1.0.0 overlay should produce identical output")
}

func TestApplyTo_V100_BackwardCompatibility_Strict(t *testing.T) {
	t.Parallel()

	node, err := loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)

	o, err := loader.LoadOverlay("testdata/overlay.yaml")
	require.NoError(t, err)

	_, err = o.ApplyToStrict(node)
	require.NoError(t, err)

	NodeMatchesFile(t, node, "testdata/openapi-overlayed.yaml", "1.0.0 strict overlay should produce identical output")
}

func TestApplyToOld(t *testing.T) {
	t.Parallel()

	nodeOld, err := loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)

	nodeNew, err := loader.LoadSpecification("testdata/openapi.yaml")
	require.NoError(t, err)

	o, err := loader.LoadOverlay("testdata/overlay-old.yaml")
	require.NoError(t, err)

	warnings, err := o.ApplyToStrict(nodeOld)
	require.NoError(t, err)
	require.Len(t, warnings, 2)
	require.Contains(t, warnings[0], "invalid rfc9535 jsonpath")
	require.Contains(t, warnings[1], "x-speakeasy-jsonpath: rfc9535")

	path, err := jsonpath.NewPath(`$.paths["/anything/selectGlobalServer"]`)
	require.NoError(t, err)
	result := path.Query(nodeOld)
	require.NoError(t, err)
	require.Empty(t, result)
	o.JSONPathVersion = "rfc9535"
	_, err = o.ApplyToStrict(nodeNew)
	require.ErrorContains(t, err, "unexpected token") // should error out: invalid nodepath
	// now lets fix it.
	o.Actions[0].Target = "$.paths.*[?(@[\"x-my-ignore\"])]"
	_, err = o.ApplyToStrict(nodeNew)
	require.ErrorContains(t, err, "did not match any targets")
	// Now lets fix it.
	o.Actions[0].Target = "$.paths[?(@[\"x-my-ignore\"])]" // @ should always refer to the child node in RFC 9535..
	_, err = o.ApplyToStrict(nodeNew)
	require.NoError(t, err)
	result = path.Query(nodeNew)
	require.NoError(t, err)
	require.Empty(t, result)
}
