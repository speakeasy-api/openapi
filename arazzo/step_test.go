package arazzo_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stepTestDoc is an Arazzo 1.1 document with OpenAPI and AsyncAPI source descriptions, the %s verb is replaced by
// additional steps in the `main` workflow.
const stepTestDoc = `arazzo: 1.1.0
info:
  title: Steps
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
  - name: events
    url: https://example.com/asyncapi.yaml
    type: asyncapi
components:
  parameters:
    limit:
      name: limit
      in: query
      value: 10
workflows:
  - workflowId: main
    steps:
      - stepId: other
        operationId: $sourceDescriptions.api.listPets
%s
  - workflowId: second
    steps:
      - stepId: remote
        operationId: $sourceDescriptions.api.getPet
        parameters:
          - name: petId
            in: path
            value: 1
`

func TestStep_Validate_Arazzo11_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		steps string
	}{
		{
			name: "asyncapi send via operationId",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.events.publishPet
        action: send`,
		},
		{
			name: "asyncapi receive via channelPath with correlationId and timeout",
			steps: `      - stepId: first
        channelPath: '{$sourceDescriptions.events.url}#/channels/pets'
        action: receive
        correlationId: $message.header.correlationId
        timeout: 1000`,
		},
		{
			name: "dependsOn local and cross workflow steps",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        dependsOn:
          - other
          - $workflows.second.steps.remote`,
		},
		{
			name: "querystring parameter",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.searchPets
        parameters:
          - name: query
            in: querystring
            value: status=available`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, sprintfDoc(stepTestDoc, tt.steps))
			assert.Empty(t, messages, "step should be valid")
		})
	}
}

func TestStep_Validate_Arazzo11_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		steps    string
		expected []string
	}{
		{
			name: "channelPath without action",
			steps: `      - stepId: first
        channelPath: '{$sourceDescriptions.events.url}#/channels/pets'`,
			expected: []string{"step.action is required when channelPath is set"},
		},
		{
			name: "asyncapi operationId without action",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.events.publishPet`,
			expected: []string{"step.action is required for asyncapi steps, operationId references asyncapi sourceDescription `events`"},
		},
		{
			name: "action on openapi step",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        action: send`,
			expected: []string{"step.action is only applicable to asyncapi steps, sourceDescription `api` is of type `openapi`"},
		},
		{
			name: "action on workflow step",
			steps: `      - stepId: first
        workflowId: second
        action: send`,
			expected: []string{"step.action is only applicable to asyncapi steps using operationId or channelPath"},
		},
		{
			name: "invalid action",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.events.publishPet
        action: publish`,
			expected: []string{"step.action must be one of [`send`, `receive`] but was `publish`"},
		},
		{
			name: "correlationId with send action",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.events.publishPet
        action: send
        correlationId: abc`,
			expected: []string{"step.correlationId is only applicable to asyncapi steps with action `receive`"},
		},
		{
			name: "channelPath referencing openapi source description",
			steps: `      - stepId: first
        channelPath: '{$sourceDescriptions.api.url}#/channels/pets'
        action: send`,
			expected: []string{"step.channelPath must reference an `asyncapi` sourceDescription, `api` is of type `openapi`"},
		},
		{
			name: "operationId and channelPath are mutually exclusive",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.events.publishPet
        channelPath: '{$sourceDescriptions.events.url}#/channels/pets'
        action: send`,
			expected: []string{"step only one of operationId, operationPath, channelPath or workflowId can be set"},
		},
		{
			name: "negative timeout",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        timeout: -1`,
			expected: []string{"step.timeout must be greater than or equal to 0"},
		},
		{
			name: "dependsOn invalid references",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        dependsOn:
          - missing
          - first
          - other
          - other
          - $steps.other
          - $workflows.second.steps.missing
          - $workflows.missing.steps.remote
          - $sourceDescriptions.api.wf.steps.remote`,
			expected: []string{
				"step.dependsOn stepId `missing` not found in workflow `main`",
				"step.dependsOn must not reference the step itself `first`",
				"step.dependsOn duplicate entry `other`",
				"step.dependsOn must be a stepId, `$workflows.<workflowId>.steps.<stepId>` or `$sourceDescriptions.<name>.<workflowId>.steps.<stepId>`, got `$steps.other`",
				"step.dependsOn stepId `missing` not found in workflow `second`",
				"step.dependsOn workflowId `missing` not found",
				"step.dependsOn sourceDescription `api` must be of type `arazzo`",
			},
		},
		{
			name: "dependsOn qualified reference to the step itself",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        dependsOn:
          - $workflows.main.steps.first`,
			expected: []string{"step.dependsOn must not reference the step itself `$workflows.main.steps.first`"},
		},
		{
			name: "bare operationId with action and multiple source descriptions",
			steps: `      - stepId: first
        operationId: publishPet
        action: send`,
			expected: []string{"step.operationId must be a valid expression if there are multiple OpenAPI or AsyncAPI source descriptions"},
		},
		{
			name: "invalid operationId expression",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.`,
			expected: []string{"step.operationId expression is invalid"},
		},
		{
			name: "invalid workflowId expression",
			steps: `      - stepId: first
        workflowId: $sourceDescriptions.api.`,
			expected: []string{"step.workflowId expression is invalid"},
		},
		{
			name: "invalid action workflowId expression",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        onSuccess:
          - name: next
            type: goto
            workflowId: $sourceDescriptions.api.`,
			expected: []string{"`successAction`.workflowId expression is invalid"},
		},
		{
			name: "dependsOn unknown source description",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        dependsOn:
          - $sourceDescriptions.missing.wf.steps.remote`,
			expected: []string{"step.dependsOn sourceDescription `missing` not found"},
		},
		{
			name: "invalid correlationId expression",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.events.receivePet
        action: receive
        correlationId: $message.payload.id`,
			expected: []string{"step.correlationId is invalid"},
		},
		{
			name: "duplicate parameter references",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        parameters:
          - reference: $components.parameters.limit
          - reference: $components.parameters.limit`,
			expected: []string{"step.parameters duplicate parameter found with reference `$components.parameters.limit`"},
		},
		{
			name: "empty dependsOn",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        dependsOn: []`,
			expected: []string{"step.dependsOn must contain at least one entry if set"},
		},
		{
			name: "empty successCriteria",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        successCriteria: []`,
			expected: []string{"step.successCriteria must contain at least one criterion if set"},
		},
		{
			name: "operation step parameter without in",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        parameters:
          - name: petId
            value: 1`,
			expected: []string{"parameter.in is required within a step when workflowId is not set"},
		},
		{
			name: "duplicate parameters",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        parameters:
          - name: limit
            in: query
            value: 1
          - name: limit
            in: query
            value: 2`,
			expected: []string{"step.parameters duplicate parameter found with name `limit` and in `query`"},
		},
		{
			name: "channel parameter location is not supported",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.api.getPet
        parameters:
          - name: petId
            in: channel
            value: 1`,
			expected: []string{"parameter.in must be one of [`path, query, querystring, header, cookie`] but was `channel`"},
		},
		{
			name: "dotted message payload expression",
			steps: `      - stepId: first
        operationId: $sourceDescriptions.events.receivePet
        action: receive
        outputs:
          petId: $message.payload.id`,
			expected: []string{"only json pointers are allowed after $message.payload"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, sprintfDoc(stepTestDoc, tt.steps))
			assertMessagesContain(t, tt.expected, messages)
		})
	}
}

// singleSourceStepTestDoc is an Arazzo 1.1 document with a single source description, the first %s verb is replaced
// by the source description type and the second by the step's operationId and action.
const singleSourceStepTestDoc = `arazzo: 1.1.0
info:
  title: Steps
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/api.yaml
%s
workflows:
  - workflowId: main
    steps:
      - stepId: first
%s
`

func TestStep_Validate_BareOperationIDSingleSource_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sourceType string
		step       string
	}{
		{
			name:       "asyncapi source with action",
			sourceType: "    type: asyncapi",
			step:       "        operationId: publishPet\n        action: send",
		},
		{
			name:       "openapi source without action",
			sourceType: "    type: openapi",
			step:       "        operationId: getPet",
		},
		{
			name:       "untyped source with action",
			sourceType: "",
			step:       "        operationId: publishPet\n        action: send",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, fmt.Sprintf(singleSourceStepTestDoc, tt.sourceType, tt.step))
			assert.Empty(t, messages, "step should be valid")
		})
	}
}

func TestStep_Validate_BareOperationIDSingleSource_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sourceType string
		step       string
		expected   []string
	}{
		{
			name:       "asyncapi source without action",
			sourceType: "    type: asyncapi",
			step:       "        operationId: publishPet",
			expected:   []string{"step.action is required for asyncapi steps, operationId references asyncapi sourceDescription `api`"},
		},
		{
			name:       "openapi source with action",
			sourceType: "    type: openapi",
			step:       "        operationId: getPet\n        action: send",
			expected:   []string{"step.action is only applicable to asyncapi steps, sourceDescription `api` is of type `openapi`"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, fmt.Sprintf(singleSourceStepTestDoc, tt.sourceType, tt.step))
			assertMessagesContain(t, tt.expected, messages)
		})
	}
}
