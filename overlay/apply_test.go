package overlay_test

import (
	"bytes"
	"os"
	"strconv"
	"testing"
	"unsafe"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
	yamlv3 "gopkg.in/yaml.v3"
)

// queryV4 runs a jsonpath query using a v4 node by casting to v3 and back.
func queryV4(path *jsonpath.JSONPath, node *yaml.Node) []*yaml.Node {
	v3result := path.Query((*yamlv3.Node)(unsafe.Pointer(node)))
	return *(*[]*yaml.Node)(unsafe.Pointer(&v3result))
}

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
	result := queryV4(path, nodeOld)
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
	result = queryV4(path, nodeNew)
	require.NoError(t, err)
	require.Empty(t, result)
}
