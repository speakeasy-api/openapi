package arazzo_test

import (
	"bytes"
	"os"
	"testing"
	"testing/fstest"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateSourcesFixture loads the fixture and its sources and returns the source validation messages.
func validateSourcesFixture(t *testing.T, path string) []string {
	t.Helper()
	ctx := t.Context()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "fixture should load")

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewReader(data))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "fixture should be structurally valid")

	sources, loadErrs := arazzo.LoadSourceDescriptions(ctx, a, arazzo.LoadSourcesOptions{DocumentLocation: path})
	require.Empty(t, loadErrs, "sources should load")

	return severityMessages(a.ValidateSources(ctx, sources))
}

func TestArazzo_ValidateSources_Success(t *testing.T) {
	t.Parallel()

	assert.Empty(t, validateSourcesFixture(t, "testdata/sources/valid.arazzo.yaml"), "document should be valid against its sources")
}

func TestArazzo_ValidateSources_Error(t *testing.T) {
	t.Parallel()

	expected := []string{
		"error: workflow.dependsOn references workflowId `missingWorkflow` which doesn't exist in sourceDescription `shared`",
		"error: step.operationId `missingOperation` not found in sourceDescription `petstore`",
		"error: step.operationPath `{$sourceDescriptions.petstore.url}#/paths/~1missing/get` doesn't reference an operation in sourceDescription `petstore`",
		"error: step.operationId references sourceDescription `shared` which is an arazzo document, not an OpenAPI document",
		"error: step.parameters `id` in `path` is not a parameter of operation `deletePet` (DELETE)",
		"warning: step.parameters `verbose` in `query` is not a parameter of operation `deletePet` (DELETE)",
		"error: step is missing required parameter `petId` in `path` of operation `deletePet` (DELETE)",
		"warning: step is missing required parameter `X-Api-Key` in `header` of operation `deletePet` (DELETE)",
		"error: step is missing required parameter `reason` in `query` of operation `deletePet` (DELETE)",
		"error: step.requestBody is required by operation `createPet` (POST)",
		"error: step.parameters `unknown` is not an input of workflow `adoptPet`",
		"error: step is missing required input `petId` of workflow `adoptPet`",
		"error: step.dependsOn stepId `missingStep` not found in workflow `adoptPet` of sourceDescription `shared`",
		"error: step.dependsOn workflowId `missingWorkflow` not found in sourceDescription `shared`",
		"error: failureAction.workflowId references workflowId `missingWorkflow` which doesn't exist in sourceDescription `shared`",
		"error: step.workflowId references sourceDescription `untypedPetstore` which is an openapi document, not an Arazzo document",
		"error: step.dependsOn references sourceDescription `untypedPetstore` which is an openapi document, not an Arazzo document",
		"error: step.operationPath references sourceDescription `shared` which is an arazzo document, not an OpenAPI document",
		"error: step.channelPath references sourceDescription `untypedPetstore` which is an openapi document, not an AsyncAPI document",
		"error: successAction.parameters `unknown` is not an input of workflow `adoptPet`",
		"error: successAction is missing required input `petId` of workflow `adoptPet`",
	}

	assert.Equal(t, expected, validateSourcesFixture(t, "testdata/sources/invalid.arazzo.yaml"), "source validation errors should match")
}

const sourceValidationEdgeCasesOpenAPI = `openapi: 3.2.0
info:
  title: Pets
  version: 1.0.0
paths:
  /broken:
    $ref: "#/components/pathItems/Missing"
  /pets:
    get:
      operationId: listPets
      parameters:
        - $ref: "#/components/parameters/Missing"
      responses:
        "200":
          description: OK
webhooks:
  petAdded:
    post:
      operationId: petAddedWebhook
      responses:
        "200":
          description: OK
`

func TestArazzo_ValidateSources_EdgeCases_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		steps string
	}{
		{
			name: "unknown parameters aren't reported when the operation has unresolved parameters",
			steps: `      - stepId: list
        operationId: listPets
        parameters:
          - name: anything
            in: query
            value: 1`,
		},
		{
			name: "operationPath that isn't a source url expression",
			steps: `      - stepId: list
        operationPath: "{$sourceDescriptions.api.url}"`,
		},
		{
			name: "operationPath to a source that wasn't loaded",
			steps: `      - stepId: list
        operationPath: "{$sourceDescriptions.events.url}#/paths/~1pets/get"
        action: send`,
		},
		{
			name: "operationId of a webhook",
			steps: `      - stepId: list
        operationId: $sourceDescriptions.api.petAddedWebhook`,
		},
		{
			name: "workflowId to a source that wasn't loaded",
			steps: `      - stepId: list
        workflowId: $sourceDescriptions.events.flow`,
		},
		{
			name: "workflowId expression that isn't a source description",
			steps: `      - stepId: list
        workflowId: $steps.first.outputs.workflow`,
		},
		{
			name: "workflowId to a workflow in the document",
			steps: `      - stepId: list
        workflowId: other`,
		},
		{
			name: "operationId targeting a source that wasn't loaded",
			steps: `      - stepId: list
        operationId: $sourceDescriptions.events.publishPet
        action: send`,
		},
		{
			name: "dependsOn a step in this document",
			steps: `      - stepId: list
        operationId: $sourceDescriptions.api.listPets
      - stepId: other
        operationId: $sourceDescriptions.api.listPets
        dependsOn:
          - list
          - $sourceDescriptions.events.flow.steps.step`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc := `arazzo: 1.1.0
info:
  title: Edge cases
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: petstore.yaml
  - name: events
    url: events.yaml
    type: asyncapi
workflows:
  - workflowId: main
    steps:
` + tt.steps + `
  - workflowId: other
    steps:
      - stepId: list
        operationId: $sourceDescriptions.api.listPets
`

			a, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(doc), arazzo.WithSkipValidation())
			require.NoError(t, err, "unmarshal should succeed")

			sources, _ := arazzo.LoadSourceDescriptions(ctx, a, arazzo.LoadSourcesOptions{
				VirtualFS: fstest.MapFS{"petstore.yaml": {Data: []byte(sourceValidationEdgeCasesOpenAPI)}},
			})
			assert.Empty(t, a.ValidateSources(ctx, sources), "nothing should be reported")
		})
	}
}

func TestArazzo_ValidateSources_OperationPaths_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		operationPath string
	}{
		{name: "pointer without a leading slash", operationPath: "{$sourceDescriptions.api.url}#paths"},
		{name: "pointer too short", operationPath: "{$sourceDescriptions.api.url}#/paths/~1pets"},
		{name: "unknown root", operationPath: "{$sourceDescriptions.api.url}#/components/~1pets/get"},
		{name: "missing method", operationPath: "{$sourceDescriptions.api.url}#/paths/~1pets/put"},
		{name: "missing webhook", operationPath: "{$sourceDescriptions.api.url}#/webhooks/missing/post"},
		{name: "missing additional operation", operationPath: "{$sourceDescriptions.api.url}#/paths/~1pets/additionalOperations/COPY"},
		{name: "unsupported pointer shape", operationPath: "{$sourceDescriptions.api.url}#/paths/~1pets/get/responses"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc := "arazzo: 1.1.0\ninfo:\n  title: Paths\n  version: 1.0.0\nsourceDescriptions:\n  - name: api\n    url: petstore.yaml\nworkflows:\n  - workflowId: main\n    steps:\n      - stepId: first\n        operationPath: \"" + tt.operationPath + "\"\n"

			a, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(doc), arazzo.WithSkipValidation())
			require.NoError(t, err, "unmarshal should succeed")

			sources, _ := arazzo.LoadSourceDescriptions(ctx, a, arazzo.LoadSourcesOptions{
				VirtualFS: fstest.MapFS{"petstore.yaml": {Data: []byte(sourceValidationEdgeCasesOpenAPI)}},
			})
			assert.Equal(t, []string{"error: step.operationPath `" + tt.operationPath + "` doesn't reference an operation in sourceDescription `api`"}, severityMessages(a.ValidateSources(ctx, sources)), "should report the missing operation")
		})
	}
}

func TestArazzo_ValidateSources_Nil_Success(t *testing.T) {
	t.Parallel()

	var a *arazzo.Arazzo
	assert.Nil(t, a.ValidateSources(t.Context(), &arazzo.SourceDocuments{}), "nil document should be nil safe")
	assert.Nil(t, (&arazzo.Arazzo{}).ValidateSources(t.Context(), nil), "nil sources should be nil safe")
}

func TestArazzo_ValidateSources_NilEntries_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	data, err := os.ReadFile("testdata/sources/valid.arazzo.yaml")
	require.NoError(t, err, "fixture should load")
	a, _, err := arazzo.Unmarshal(ctx, bytes.NewReader(data))
	require.NoError(t, err, "unmarshal should succeed")

	sources, loadErrs := arazzo.LoadSourceDescriptions(ctx, a, arazzo.LoadSourcesOptions{DocumentLocation: "testdata/sources/valid.arazzo.yaml"})
	require.Empty(t, loadErrs, "sources should load")

	a.Workflows = append(a.Workflows, nil)
	w := a.Workflows[0]
	w.Steps = append(w.Steps, nil)
	w.Parameters = append(w.Parameters, nil)
	w.SuccessActions = []*arazzo.ReusableSuccessAction{nil, {Reference: nil}}
	w.FailureActions = []*arazzo.ReusableFailureAction{nil, {Reference: nil}}
	a.Components = &arazzo.Components{}
	steps := w.Steps
	steps[1].Parameters = append(steps[1].Parameters, nil)

	assert.Empty(t, a.ValidateSources(ctx, sources), "nil entries should be skipped")
}
