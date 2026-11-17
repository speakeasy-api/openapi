package arazzoexplore_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/arazzoexplore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const exploreDoc = `arazzo: 1.1.0
info:
  title: Explore
  version: 1.0.0
sourceDescriptions:
  - name: petStore
    url: ./petstore.yaml
    type: openapi
  - name: events
    url: ./events.yaml
    type: asyncapi
workflows:
  - workflowId: adoptPet
    summary: Adopt a pet
    description: Finds a pet and adopts it
    dependsOn:
      - registerOwner
    inputs:
      type: object
      properties:
        species:
          type: string
          description: The species
        owner:
          $ref: "#/components/inputs/owner"
        anything: {}
      required:
        - species
    parameters:
      - reference: $components.parameters.apiVersion
    steps:
      - stepId: findPet
        description: Find a pet
        operationId: listPets
        parameters:
          - name: species
            in: query
            value: $inputs.species
          - reference: $components.parameters.apiVersion
            value: "2026"
        successCriteria:
          - condition: $statusCode == 200
          - context: $response.body
            condition: $[0]
            type:
              type: jsonpath
              version: rfc9535
        onSuccess:
          - name: noPets
            type: end
            criteria:
              - condition: $response.body#/0 == null
          - name: next
            type: goto
            stepId: adopt
        outputs:
          petId: $response.body#/0/id
          owner:
            context: $response.body
            selector: $.owner
            type: jsonpath
      - stepId: adopt
        operationPath: "{$sourceDescriptions.petStore.url}#/paths/~1pets~1{petId}/post"
        dependsOn:
          - findPet
        requestBody:
          contentType: application/json
          payload:
            petId: $steps.findPet.outputs.petId
            tags: [new, adopted]
          replacements:
            - target: /owner
              value: $inputs.species
        onFailure:
          - name: retryBusy
            type: retry
            retryAfter: 2.5
            stepId: findPet
          - name: fallback
            type: goto
            workflowId: registerOwner
      - stepId: waitForAdoption
        channelPath: "{$sourceDescriptions.events.url}#/channels/adopted"
        action: receive
        correlationId: $inputs.species
        timeout: 5000
      - stepId: register
        workflowId: registerOwner
        parameters:
          - name: name
            value: Jane
    successActions:
      - reference: $components.successActions.done
    failureActions:
      - name: retryBusy
        type: retry
        retryAfter: 1
        retryLimit: 3
        criteria:
          - condition: $statusCode == 503
      - reference: $components.failureActions.missing
    outputs:
      petId: $steps.findPet.outputs.petId
  - workflowId: registerOwner
    steps:
      - stepId: create
        operationId: createOwner
      - stepId: send
        channelPath: "{$sourceDescriptions.events.url}#/channels/owners"
        action: send
      - stepId: untargeted
components:
  inputs:
    owner:
      type: string
  parameters:
    apiVersion:
      name: X-API-Version
      in: header
      value: "2025"
  successActions:
    done:
      name: done
      type: end
`

func TestCollect_Success(t *testing.T) {
	t.Parallel()

	doc, _, err := arazzo.Unmarshal(t.Context(), strings.NewReader(exploreDoc), arazzo.WithSkipValidation())
	require.NoError(t, err, "document should parse")

	expected := []arazzoexplore.Item{
		{
			Kind:    arazzoexplore.KindWorkflow,
			Label:   "adoptPet",
			Summary: "Adopt a pet",
			Details: []string{
				"Summary: Adopt a pet",
				"Description: Finds a pet and adopts it",
				"Depends on:",
				"  - registerOwner",
				"Inputs:",
				"  - species (string, required): The species",
				"  - owner (#/components/inputs/owner)",
				"  - anything",
				"Parameters:",
				"  - X-API-Version (header) = 2025 (from $components.parameters.apiVersion)",
				"Success actions:",
				"  - done: end the workflow (from $components.successActions.done)",
				"Failure actions:",
				"  - retryBusy: retry after 1s up to 3 times when $statusCode == 503",
				"  - $components.failureActions.missing (unresolved)",
				"Outputs:",
				"  - petId: $steps.findPet.outputs.petId",
				"Steps: 4",
			},
		},
		{
			Kind:    arazzoexplore.KindStep,
			Label:   "findPet",
			Badge:   arazzoexplore.BadgeOperation,
			Summary: "listPets",
			Details: []string{
				"Description: Find a pet",
				"Operation: listPets",
				"Parameters:",
				"  - species (query) = $inputs.species",
				"  - X-API-Version (header) = 2026 (from $components.parameters.apiVersion)",
				"Success criteria:",
				"  - $statusCode == 200",
				"  - $[0] (jsonpath rfc9535, context: $response.body)",
				"On success:",
				"  - noPets: end the workflow when $response.body#/0 == null",
				"  - next: go to step adopt",
				"  - done: end the workflow (from $components.successActions.done) (from workflow)",
				"On failure:",
				"  - retryBusy: retry after 1s up to 3 times when $statusCode == 503 (from workflow)",
				"  - $components.failureActions.missing (unresolved) (from workflow)",
				"Outputs:",
				"  - petId: $response.body#/0/id",
				"  - owner: $.owner from $response.body (jsonpath)",
			},
		},
		{
			Kind:    arazzoexplore.KindStep,
			Label:   "adopt",
			Badge:   arazzoexplore.BadgeOperationPath,
			Summary: "{$sourceDescriptions.petStore.url}#/paths/~1pets~1{petId}/post",
			Details: []string{
				"Operation path: {$sourceDescriptions.petStore.url}#/paths/~1pets~1{petId}/post",
				"Depends on:",
				"  - findPet",
				"Request body:",
				"  - content type: application/json",
				"  - payload: {petId: $steps.findPet.outputs.petId, tags: [new, adopted]}",
				"  - replace /owner = $inputs.species",
				"On success:",
				"  - done: end the workflow (from $components.successActions.done) (from workflow)",
				"On failure:",
				"  - retryBusy: retry after 2.5s via step findPet",
				"  - fallback: go to workflow registerOwner",
				"  - $components.failureActions.missing (unresolved) (from workflow)",
			},
		},
		{
			Kind:    arazzoexplore.KindStep,
			Label:   "waitForAdoption",
			Badge:   arazzoexplore.BadgeReceive,
			Summary: "{$sourceDescriptions.events.url}#/channels/adopted",
			Details: []string{
				"Channel (receive): {$sourceDescriptions.events.url}#/channels/adopted",
				"Correlation ID: $inputs.species",
				"Timeout: 5000ms",
				"On success:",
				"  - done: end the workflow (from $components.successActions.done) (from workflow)",
				"On failure:",
				"  - retryBusy: retry after 1s up to 3 times when $statusCode == 503 (from workflow)",
				"  - $components.failureActions.missing (unresolved) (from workflow)",
			},
		},
		{
			Kind:    arazzoexplore.KindStep,
			Label:   "register",
			Badge:   arazzoexplore.BadgeWorkflow,
			Summary: "registerOwner",
			Details: []string{
				"Workflow: registerOwner",
				"Parameters:",
				"  - name = Jane",
				"On success:",
				"  - done: end the workflow (from $components.successActions.done) (from workflow)",
				"On failure:",
				"  - retryBusy: retry after 1s up to 3 times when $statusCode == 503 (from workflow)",
				"  - $components.failureActions.missing (unresolved) (from workflow)",
			},
		},
		{
			Kind:    arazzoexplore.KindWorkflow,
			Label:   "registerOwner",
			Details: []string{"Steps: 3"},
		},
		{
			Kind:    arazzoexplore.KindStep,
			Label:   "create",
			Badge:   arazzoexplore.BadgeOperation,
			Summary: "createOwner",
			Details: []string{"Operation: createOwner"},
		},
		{
			Kind:    arazzoexplore.KindStep,
			Label:   "send",
			Badge:   arazzoexplore.BadgeSend,
			Summary: "{$sourceDescriptions.events.url}#/channels/owners",
			Details: []string{"Channel (send): {$sourceDescriptions.events.url}#/channels/owners"},
		},
		{
			Kind:  arazzoexplore.KindStep,
			Label: "untargeted",
			Badge: arazzoexplore.BadgeNone,
		},
	}

	assert.Equal(t, expected, arazzoexplore.Collect(doc), "items should describe the workflows and steps")
}

func TestCollect_Edges_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		doc      string
		expected []arazzoexplore.Item
	}{
		{
			name: "references without components",
			doc: `arazzo: 1.1.0
info: {title: Edges, version: 1.0.0}
sourceDescriptions: []
workflows:
  - workflowId: refs
    inputs:
      $ref: "#/components/inputs/all"
    steps:
      - stepId: only
        channelPath: "{$sourceDescriptions.events.url}#/channels/x"
        parameters:
          - reference: $components.parameters.missing
        onSuccess:
          - reference: $components.successActions.missing
        onFailure:
          - reference: $components.failureActions.missing
`,
			expected: []arazzoexplore.Item{
				{
					Kind:    arazzoexplore.KindWorkflow,
					Label:   "refs",
					Details: []string{"Inputs:", "  - #/components/inputs/all", "Steps: 1"},
				},
				{
					Kind:    arazzoexplore.KindStep,
					Label:   "only",
					Badge:   arazzoexplore.BadgeChannel,
					Summary: "{$sourceDescriptions.events.url}#/channels/x",
					Details: []string{
						"Channel: {$sourceDescriptions.events.url}#/channels/x",
						"Parameters:",
						"  - $components.parameters.missing (unresolved)",
						"On success:",
						"  - $components.successActions.missing (unresolved)",
						"On failure:",
						"  - $components.failureActions.missing (unresolved)",
					},
				},
			},
		},
		{
			name:     "no workflows",
			doc:      "arazzo: 1.1.0\ninfo: {title: Empty, version: 1.0.0}\nsourceDescriptions: []\nworkflows: []\n",
			expected: []arazzoexplore.Item{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, _, err := arazzo.Unmarshal(t.Context(), strings.NewReader(tt.doc), arazzo.WithSkipValidation())
			require.NoError(t, err, "document should parse")
			assert.Equal(t, tt.expected, arazzoexplore.Collect(doc), "items should match")
		})
	}
}

func TestCollect_NilDocument_Success(t *testing.T) {
	t.Parallel()

	assert.Nil(t, arazzoexplore.Collect(nil), "a nil document should have no items")
}
