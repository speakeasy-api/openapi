package arazzo_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/stretchr/testify/assert"
)

// arazzo11FeaturesDoc uses every Arazzo 1.1 only feature, it is valid when declared as 1.1.x and invalid when
// declared as 1.0.x.
const arazzo11FeaturesDoc = `arazzo: %s
$self: https://example.com/workflows/pets.arazzo.yaml
info:
  title: Arazzo 1.1 features
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
  - name: events
    url: https://example.com/asyncapi.yaml
    type: asyncapi
workflows:
  - workflowId: main
    inputs:
      type: object
      properties:
        pet:
          type: object
    steps:
      - stepId: search
        operationId: $sourceDescriptions.api.searchPets
        parameters:
          - name: query
            in: querystring
            value: status=available
        requestBody:
          contentType: application/json
          payload: {"pet": {"id": 0}}
          replacements:
            - target: $.pet.id
              targetSelectorType: jsonpath
              value:
                context: $inputs.pet
                selector: /id
                type: jsonpointer
        successCriteria:
          - condition: $statusCode == 200
        onSuccess:
          - name: adopt
            type: goto
            workflowId: adopt
            parameters:
              - name: petId
                value: $response.body#/0/id
        outputs:
          firstPetId:
            context: $response.body
            selector: $[0].id
            type: jsonpath
      - stepId: waitForAdoption
        channelPath: '{$sourceDescriptions.events.url}#/channels/adoptions'
        action: receive
        correlationId: $steps.search.outputs.firstPetId
        timeout: 5000
        dependsOn:
          - search
  - workflowId: adopt
    inputs:
      type: object
      properties:
        petId:
          type: integer
    steps:
      - stepId: adopt
        operationId: $sourceDescriptions.api.adoptPet
        parameters:
          - name: petId
            in: path
            value: $inputs.petId
`

func TestArazzo_Validate_Arazzo11Features_Success(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"1.1.0", "1.1.1", "1.1.0-rc1"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, sprintfDoc(arazzo11FeaturesDoc, version))
			assert.Empty(t, messages, "Arazzo 1.1 features should be valid in a %s document", version)
		})
	}
}

func TestArazzo_Validate_Arazzo11FeaturesIn10Document_Error(t *testing.T) {
	t.Parallel()

	messages := validationMessages(t, sprintfDoc(arazzo11FeaturesDoc, "1.0.1"))

	requires11 := " requires Arazzo 1.1.0 or later, update the `arazzo` version (e.g. with `openapi arazzo upgrade`)"
	expected := []string{
		"`$self`" + requires11,
		"`sourceDescription.type` `asyncapi`" + requires11,
		"`parameter.in` `querystring`" + requires11,
		"`payloadReplacement.targetSelectorType`" + requires11,
		"`successAction.parameters`" + requires11,
		"a selector object" + requires11,
		"`step.channelPath`" + requires11,
		"`step.action`" + requires11,
		"`step.correlationId`" + requires11,
		"`step.timeout`" + requires11,
		"`step.dependsOn`" + requires11,
	}
	assert.ElementsMatch(t, expected, messages, "should report each Arazzo 1.1 feature used in a 1.0 document")
}

func TestArazzo_Validate_Self_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		self     string
		expected string
	}{
		{name: "fragment identifier", self: "https://example.com/pets.arazzo.yaml#/info", expected: "`$self` must not contain a fragment identifier"},
		{name: "invalid uri", self: "'http://[::1'", expected: "`$self` must be a valid URI reference"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := "arazzo: 1.1.0\n$self: " + tt.self + "\ninfo:\n  title: Self\n  version: 1.0.0\nsourceDescriptions:\n  - name: api\n    url: ./api.yaml\nworkflows:\n  - workflowId: main\n    steps:\n      - stepId: first\n        operationId: getPet\n"
			messages := validationMessages(t, doc)
			assertMessagesContain(t, []string{tt.expected}, messages)
		})
	}
}

func TestWorkflow_Validate_InvalidDependsOnExpression_Error(t *testing.T) {
	t.Parallel()

	doc := `arazzo: 1.1.0
info:
  title: Workflows
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: ./api.yaml
workflows:
  - workflowId: main
    dependsOn:
      - $sourceDescriptions.api
    steps:
      - stepId: first
        operationId: getPet
`

	messages := validationMessages(t, doc)
	assertMessagesContain(t, []string{"workflow.dependsOn expression is invalid"}, messages)
}

func TestIsSupportedVersion_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version  string
		expected bool
	}{
		{version: "1.0.0", expected: true},
		{version: "1.0.1", expected: true},
		{version: "1.0.2", expected: true},
		{version: "1.1.0", expected: true},
		{version: "1.1.5", expected: true},
		{version: "1.1.0-rc1", expected: true},
		{version: "1.2.0", expected: false},
		{version: "2.0.0", expected: false},
		{version: "0.9.0", expected: false},
		{version: "1.0", expected: false},
		{version: "not-a-version", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, arazzo.IsSupportedVersion(tt.version), "supported version should match")
		})
	}
}
