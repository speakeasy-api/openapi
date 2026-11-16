package arazzo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// actionTestDoc is an Arazzo 1.1 document, the %s verb is replaced by the onSuccess/onFailure actions of the
// `first` step.
const actionTestDoc = `arazzo: 1.1.0
info:
  title: Actions
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
components:
  parameters:
    petId:
      name: petId
      value: 1
    petIdPath:
      name: petId
      in: path
      value: 1
workflows:
  - workflowId: main
    steps:
      - stepId: first
        operationId: getPet
%s
      - stepId: second
        operationId: listPets
  - workflowId: adopt
    steps:
      - stepId: adopt
        operationId: adoptPet
`

func TestActions_Validate_Parameters_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		actions string
	}{
		{
			name: "success action with inline and reusable parameters",
			actions: `        onSuccess:
          - name: adopt
            type: goto
            workflowId: adopt
            parameters:
              - name: name
                value: $response.body#/name
              - reference: $components.parameters.petId`,
		},
		{
			name: "failure action with parameters",
			actions: `        onFailure:
          - name: adopt
            type: retry
            workflowId: adopt
            retryLimit: 3
            parameters:
              - name: name
                value: fallback`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, sprintfDoc(actionTestDoc, tt.actions))
			assert.Empty(t, messages, "actions should be valid")
		})
	}
}

func TestActions_Validate_Parameters_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		actions  string
		expected []string
	}{
		{
			name: "success action parameters require workflowId",
			actions: `        onSuccess:
          - name: next
            type: goto
            stepId: second
            parameters:
              - name: name
                value: rex`,
			expected: []string{"successAction.workflowId is required when parameters are set"},
		},
		{
			name: "failure action parameters require workflowId",
			actions: `        onFailure:
          - name: retry
            type: retry
            parameters:
              - name: name
                value: rex`,
			expected: []string{"failureAction.workflowId is required when parameters are set"},
		},
		{
			name: "action parameters must not use in",
			actions: `        onSuccess:
          - name: adopt
            type: goto
            workflowId: adopt
            parameters:
              - name: name
                in: query
                value: rex`,
			expected: []string{"parameter.in must not be used for successAction parameters"},
		},
		{
			name: "referenced component parameter with in",
			actions: `        onFailure:
          - name: adopt
            type: goto
            workflowId: adopt
            parameters:
              - reference: $components.parameters.petIdPath`,
			expected: []string{"failureAction.parameters reference `$components.parameters.petIdPath` resolves to a parameter with `in`, which must not be used for action parameters"},
		},
		{
			name: "duplicate action parameters",
			actions: `        onSuccess:
          - name: adopt
            type: goto
            workflowId: adopt
            parameters:
              - name: name
                value: rex
              - name: name
                value: fido
              - reference: $components.parameters.petId
              - reference: $components.parameters.petId`,
			expected: []string{
				"successAction.parameters duplicate parameter found with name `name`",
				"successAction.parameters duplicate parameter found with reference `$components.parameters.petId`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, sprintfDoc(actionTestDoc, tt.actions))
			assertMessagesContain(t, tt.expected, messages)
		})
	}
}
