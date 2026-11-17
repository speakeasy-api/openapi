package arazzo_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	arazzoLinter "github.com/speakeasy-api/openapi/arazzo/linter"
	baseLinter "github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bootstrapFixture = "testdata/bootstrap/petstore.openapi.yaml"

// loadBootstrapOpenAPI parses an OpenAPI document and resolves its references, ignoring references that can't be
// resolved.
func loadBootstrapOpenAPI(t *testing.T, data []byte, location string) *openapi.OpenAPI {
	t.Helper()

	doc, _, err := openapi.Unmarshal(t.Context(), bytes.NewReader(data), openapi.WithSkipValidation())
	require.NoError(t, err, "OpenAPI document should parse")
	_, _ = doc.ResolveAllReferences(t.Context(), openapi.ResolveAllOptions{OpenAPILocation: location})
	return doc
}

// marshalArazzo marshals an Arazzo document to YAML.
func marshalArazzo(t *testing.T, a *arazzo.Arazzo) string {
	t.Helper()

	var out bytes.Buffer
	require.NoError(t, arazzo.Marshal(t.Context(), a, &out), "document should marshal")
	return out.String()
}

// requireValidAndClean checks a generated document validates, including against its sources when location is set,
// and has no lint findings.
func requireValidAndClean(t *testing.T, data, location string) {
	t.Helper()
	ctx := t.Context()

	a, errs, err := arazzo.Unmarshal(ctx, strings.NewReader(data))
	require.NoError(t, err, "generated document should parse")
	assert.Empty(t, errs, "generated document should be valid")

	if location != "" {
		sources, loadErrs := arazzo.LoadSourceDescriptions(ctx, a, arazzo.LoadSourcesOptions{DocumentLocation: location})
		assert.Empty(t, loadErrs, "sources should load")
		assert.Empty(t, a.ValidateSources(ctx, sources), "generated document should be valid against its sources")
	}

	output, err := arazzoLinter.NewLinter(nil).Lint(ctx, baseLinter.NewDocumentInfo(a, location), nil, &baseLinter.LintOptions{})
	require.NoError(t, err, "lint should succeed")
	assert.Empty(t, output.Results, "generated document should have no lint findings")
}

func TestBootstrapFromOpenAPI_Success(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(bootstrapFixture)
	require.NoError(t, err, "fixture should open")

	expected, err := os.ReadFile("testdata/bootstrap/petstore.arazzo.yaml")
	require.NoError(t, err, "golden file should open")

	a, err := arazzo.BootstrapFromOpenAPI(loadBootstrapOpenAPI(t, data, bootstrapFixture), arazzo.BootstrapOptions{SourceURL: "./petstore.openapi.yaml"})
	require.NoError(t, err, "bootstrap should succeed")

	actual := marshalArazzo(t, a)
	assert.Equal(t, string(expected), actual, "generated document should match the golden file")
	requireValidAndClean(t, actual, "testdata/bootstrap/petstore.arazzo.yaml")
}

func TestBootstrapFromOpenAPI_Tags_Success(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(bootstrapFixture)
	require.NoError(t, err, "fixture should open")

	tests := []struct {
		name     string
		tags     []string
		expected []string
	}{
		{name: "single tag", tags: []string{"adoption"}, expected: []string{"adoption"}},
		{name: "untagged", tags: []string{arazzo.UntaggedWorkflowID}, expected: []string{"default"}},
		{name: "document order", tags: []string{"default", "pets"}, expected: []string{"pets", "default"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := arazzo.BootstrapFromOpenAPI(loadBootstrapOpenAPI(t, data, bootstrapFixture), arazzo.BootstrapOptions{SourceURL: "./petstore.openapi.yaml", Tags: tt.tags})
			require.NoError(t, err, "bootstrap should succeed")

			ids := make([]string, 0, len(a.Workflows))
			for _, w := range a.Workflows {
				ids = append(ids, w.WorkflowID)
			}
			assert.Equal(t, tt.expected, ids, "workflows should match the tags")
		})
	}
}

func TestBootstrapFromOpenAPI_Operations_Success(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(bootstrapFixture)
	require.NoError(t, err, "fixture should open")

	a, err := arazzo.BootstrapFromOpenAPI(loadBootstrapOpenAPI(t, data, bootstrapFixture), arazzo.BootstrapOptions{
		SourceURL:    "./petstore.openapi.yaml",
		SourceName:   "pets",
		OperationIDs: []string{"createPet", "getPet", "getPet"},
		WorkflowID:   "fetchTwice",
	})
	require.NoError(t, err, "bootstrap should succeed")

	actual := marshalArazzo(t, a)
	assert.Equal(t, `arazzo: 1.1.0
info:
  title: Pet Store API workflows
  description: Starter workflows generated from Pet Store API 2.0.0. Review each step's parameters, request body and success criteria, then add outputs or chain further steps as needed.
  version: 1.0.0
sourceDescriptions:
  - name: pets
    url: ./petstore.openapi.yaml
    type: openapi
workflows:
  - workflowId: fetchTwice
    summary: Calls createPet, getPet, getPet in order
    inputs:
      type: object
      properties:
        createPetBody:
          type: object
          description: The pet to create
        petId:
          type: string
          description: The pet's ID
      required:
        - createPetBody
        - petId
    steps:
      - stepId: createPet
        description: Create a pet
        operationId: createPet
        requestBody:
          contentType: application/json
          payload: $inputs.createPetBody
        successCriteria:
          - condition: $statusCode == 201
      - stepId: getPet
        description: Get a pet
        operationId: getPet
        parameters:
          - name: petId
            in: path
            value: $inputs.petId
        successCriteria:
          - context: $statusCode
            condition: ^2\d\d$
            type: regex
      - stepId: getPet2
        description: Get a pet
        operationId: getPet
        parameters:
          - name: petId
            in: path
            value: $inputs.petId
        successCriteria:
          - context: $statusCode
            condition: ^2\d\d$
            type: regex
`, actual, "generated document should call the operations in order")
	requireValidAndClean(t, actual, "testdata/bootstrap/petstore.arazzo.yaml")
}

func TestBootstrapFromOpenAPI_EdgeCases_Success(t *testing.T) {
	t.Parallel()

	source := `openapi: 3.1.0
info:
  title: ""
  version: 1.0.0
tags:
  - name: ""
paths:
  /missing:
    $ref: "#/components/pathItems/Missing"
  /things/{id}:
    parameters:
      - $ref: "#/components/parameters/Missing"
      - name: id
        in: path
        required: true
        schema:
          type: integer
    get:
      operationId: get.thing
      tags: [Thing Store, default, Thing Store]
      parameters:
        - name: id
          in: path
          required: true
          schema: true
        - name: X-Trace
          in: header
          required: true
          content:
            application/json:
              schema:
                type: string
                format: uuid
      responses:
        "200":
          description: The thing
          links:
            unusable:
              operationId: put.thing
              parameters:
                X-Trace: $response.body.id
                id: $inputs.id
              requestBody:
                payload: $response.body#/thing
            external:
              operationRef: "./other.yaml#/paths/~1things~1{id}/put"
              parameters:
                id: $response.header.X-Id
            update:
              operationId: put.thing
              parameters:
                cookie.id: $response.body#/ignored
                path.id: $response.body#/id
                other: $response.body#/other
                x-trace:
                  trace: abc
                price: $5
              requestBody:
                payload: fixed
        "202":
          description: Accepted
    put:
      operationId: put.thing
      tags: [Thing Store]
      parameters:
        - name: X-Trace
          in: header
          required: true
          schema:
            type: string
        - name: id
          in: query
          required: true
          schema:
            type: string
        - name: price
          in: query
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          text/plain: {}
      responses:
        "201":
          description: Updated
  /ping:
    get:
      operationId: ping
      responses:
        "204":
          description: Pong
`

	a, err := arazzo.BootstrapFromOpenAPI(loadBootstrapOpenAPI(t, []byte(source), "edge.openapi.yaml"), arazzo.BootstrapOptions{SourceURL: "https://example.com/edge.openapi.yaml"})
	require.NoError(t, err, "bootstrap should succeed")

	actual := marshalArazzo(t, a)
	assert.Equal(t, `arazzo: 1.1.0
info:
  title: API workflows
  description: Starter workflows generated from API 1.0.0. Review each step's parameters, request body and success criteria, then add outputs or chain further steps as needed.
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/edge.openapi.yaml
    type: openapi
workflows:
  - workflowId: thingStore
    summary: Calls the `+"`Thing Store`"+` operations in order
    inputs:
      type: object
      properties:
        id: {}
        xTrace:
          type: string
          format: uuid
        id2:
          type: string
      required:
        - id
        - xTrace
        - id2
    steps:
      - stepId: getThing
        operationId: get.thing
        parameters:
          - name: id
            in: path
            value: $inputs.id
          - name: X-Trace
            in: header
            value: $inputs.xTrace
        successCriteria:
          - context: $statusCode
            condition: ^(200|202)$
            type: regex
        outputs:
          id: $response.body#/id
      - stepId: putThing
        operationId: put.thing
        parameters:
          - name: id
            in: path
            value: $steps.getThing.outputs.id
          - name: X-Trace
            in: header
            value:
              trace: abc
          - name: id
            in: query
            value: $inputs.id2
          - name: price
            in: query
            value: $5
        requestBody:
          contentType: text/plain
          payload:
            payload: fixed
        successCriteria:
          - condition: $statusCode == 201
  - workflowId: default
    summary: Calls the `+"`default`"+` operations in order
    inputs:
      type: object
      properties:
        id: {}
        xTrace:
          type: string
          format: uuid
      required:
        - id
        - xTrace
    steps:
      - stepId: getThing
        operationId: get.thing
        parameters:
          - name: id
            in: path
            value: $inputs.id
          - name: X-Trace
            in: header
            value: $inputs.xTrace
        successCriteria:
          - context: $statusCode
            condition: ^(200|202)$
            type: regex
      - stepId: ping
        operationId: ping
        successCriteria:
          - condition: $statusCode == 204
`, actual, "generated document should handle the edge cases")
	requireValidAndClean(t, actual, "")
}

func TestBootstrapFromOpenAPI_Error(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(bootstrapFixture)
	require.NoError(t, err, "fixture should open")

	tests := []struct {
		name     string
		source   string
		opts     arazzo.BootstrapOptions
		expected string
	}{
		{name: "no source URL", opts: arazzo.BootstrapOptions{}, expected: "a source URL for the OpenAPI document is required"},
		{name: "tags and operations", opts: arazzo.BootstrapOptions{SourceURL: "a.yaml", Tags: []string{"pets"}, OperationIDs: []string{"getPet"}}, expected: "tags and operationIds can't be used together"},
		{name: "invalid source name", opts: arazzo.BootstrapOptions{SourceURL: "a.yaml", SourceName: "my api"}, expected: "source name `my api` must match ^[A-Za-z0-9_\\-]+$"},
		{name: "unknown tag", opts: arazzo.BootstrapOptions{SourceURL: "a.yaml", Tags: []string{"unused"}}, expected: "no operations found for tag `unused`"},
		{name: "unknown operation", opts: arazzo.BootstrapOptions{SourceURL: "a.yaml", OperationIDs: []string{"getPet", "deletePet"}}, expected: "operationId `deletePet` not found in the OpenAPI document"},
		{name: "invalid workflowId", opts: arazzo.BootstrapOptions{SourceURL: "a.yaml", OperationIDs: []string{"getPet"}, WorkflowID: "my workflow"}, expected: "workflowId `my workflow` must match ^[A-Za-z0-9_\\-]+$"},
		{name: "no paths", source: "openapi: 3.1.0\ninfo:\n  title: Empty\n  version: 1.0.0\n", opts: arazzo.BootstrapOptions{SourceURL: "a.yaml"}, expected: "the OpenAPI document has no operations"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := data
			if tt.source != "" {
				source = []byte(tt.source)
			}

			_, err := arazzo.BootstrapFromOpenAPI(loadBootstrapOpenAPI(t, source, bootstrapFixture), tt.opts)
			require.EqualError(t, err, tt.expected, "bootstrap should fail")
		})
	}
}

func TestBootstrapFromOpenAPI_NilDocument_Error(t *testing.T) {
	t.Parallel()

	_, err := arazzo.BootstrapFromOpenAPI(nil, arazzo.BootstrapOptions{SourceURL: "a.yaml"})
	require.EqualError(t, err, "an OpenAPI document is required", "bootstrap should fail")
}

func TestBootstrap_Success(t *testing.T) {
	t.Parallel()

	expected, err := os.ReadFile("testdata/bootstrap/sample.arazzo.yaml")
	require.NoError(t, err, "golden file should open")

	actual := marshalArazzo(t, arazzo.Bootstrap())
	assert.Equal(t, string(expected), actual, "sample document should match the golden file")
	requireValidAndClean(t, actual, "")
}
