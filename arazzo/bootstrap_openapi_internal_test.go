package arazzo

import (
	"slices"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapStep_LinkValue_Outputs_Success(t *testing.T) {
	t.Parallel()

	bs := &bootstrapStep{step: &Step{StepID: "first"}, outputs: map[string]string{}}

	assert.Equal(t, "$steps.first.outputs.id", bs.linkValue(yml.CreateStringNode("$response.body#/id"), "id").Value, "first expression should get an output named after the parameter")
	assert.Equal(t, "$steps.first.outputs.id", bs.linkValue(yml.CreateStringNode("$response.body#/id"), "other").Value, "the same expression should reuse its output")
	assert.Equal(t, "$steps.first.outputs.id2", bs.linkValue(yml.CreateStringNode("$response.header.X-Id"), "id").Value, "a different expression should get a unique output name")
	assert.Equal(t, "$steps.first.outputs.value", bs.linkValue(yml.CreateStringNode("$statusCode"), "!!").Value, "names without usable characters should fall back")
	assert.Nil(t, bs.linkValue(nil, "id"), "nil values should be skipped")

	require.NotNil(t, bs.step.Outputs, "outputs should be set")
	assert.Equal(t, []string{"id", "id2", "value"}, slices.Collect(bs.step.Outputs.Keys()), "outputs should be declared once each")
}

func TestBootstrapFromOpenAPI_DefaultWorkflowID_Success(t *testing.T) {
	t.Parallel()

	doc := &openapi.OpenAPI{Paths: openapi.NewPaths()}
	pathItem := openapi.NewPathItem()
	pathItem.Set(openapi.HTTPMethodGet, &openapi.Operation{OperationID: pointer.From("getThing")})
	doc.Paths.Set("/thing", &openapi.ReferencedPathItem{Object: pathItem})

	a, err := BootstrapFromOpenAPI(doc, BootstrapOptions{SourceURL: "a.yaml", OperationIDs: []string{"getThing"}})
	require.NoError(t, err, "bootstrap should succeed")
	require.Len(t, a.Workflows, 1, "a single workflow should be generated")
	assert.Equal(t, DefaultBootstrapWorkflowID, a.Workflows[0].WorkflowID, "the default workflowId should be used")
	assert.Equal(t, "api", a.SourceDescriptions[0].Name, "the source name should fall back when there's no title")
}

func TestToIdentifier_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value    string
		expected string
	}{
		{value: "listPets", expected: "listPets"},
		{value: "list_pets-v2", expected: "list_pets-v2"},
		{value: "Pet Store API", expected: "petStoreAPI"},
		{value: "PURGE /health", expected: "purgeHealth"},
		{value: "get /pets/{petId}", expected: "getPetsPetId"},
		{value: "pets.list", expected: "petsList"},
		{value: "{}", expected: "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, toIdentifier(tt.value, "fallback"), "identifier should match")
		})
	}
}
