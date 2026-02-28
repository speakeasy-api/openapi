package arazzo_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestRequestBody_Validate_JSONPathInPayload_Success(t *testing.T) {
	t.Parallel()
	// This test reproduces the bug where JSONPath-like expressions in payload
	// are incorrectly validated as Arazzo runtime expressions
	ctx := t.Context()

	// Create a payload that contains JSONPath-like expressions ($.Id, $.time, etc.)
	// but these should NOT be validated as Arazzo expressions since they're just payload data
	payloadNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "dataSource"},
			{Kind: yaml.ScalarNode, Value: "/query?query=select * from Account"},
			{Kind: yaml.ScalarNode, Value: "keyBy"},
			{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "$.Id"},
			}},
			{Kind: yaml.ScalarNode, Value: "requiredData"},
			{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "Currentbal"},
				{Kind: yaml.ScalarNode, Value: "$.CurrentBalance"},
				{Kind: yaml.ScalarNode, Value: "SubAcc"},
				{Kind: yaml.ScalarNode, Value: "$.SubAccount"},
				{Kind: yaml.ScalarNode, Value: "id"},
				{Kind: yaml.ScalarNode, Value: "$.Id"},
			}},
			{Kind: yaml.ScalarNode, Value: "sourceModifiedDate"},
			{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "$.time"},
			}},
		},
	}

	requestBody := &arazzo.RequestBody{
		ContentType: pointer.From("application/json"),
		Payload:     payloadNode,
		Model: marshaller.Model[core.RequestBody]{
			Valid: true,
		},
	}

	// This should NOT return validation errors for the JSONPath expressions in the payload
	errs := requestBody.Validate(ctx)

	// The bug causes validation errors like:
	// "expression is not valid, must begin with one of [url, method, statusCode, request, response, inputs, outputs, steps, workflows, sourceDescriptions, components]: $.time"
	assert.Empty(t, errs, "JSONPath-like expressions in payload should not be validated as Arazzo expressions")
}

func TestRequestBody_Validate_AnyPayloadData_Success(t *testing.T) {
	t.Parallel()
	// This test ensures that ANY data in payloads is allowed, including invalid expressions
	// since payloads are arbitrary user data and should not be validated as Arazzo expressions
	ctx := t.Context()

	// Create a payload with various expression-like data that should all be ignored
	payloadNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "invalidExpression"},
			{Kind: yaml.ScalarNode, Value: "$invalidExpression"}, // Would be invalid if validated
			{Kind: yaml.ScalarNode, Value: "jsonPath"},
			{Kind: yaml.ScalarNode, Value: "$.field.subfield"}, // JSONPath expression
			{Kind: yaml.ScalarNode, Value: "template"},
			{Kind: yaml.ScalarNode, Value: "${variable}"}, // Template-like expression
		},
	}

	requestBody := &arazzo.RequestBody{
		ContentType: pointer.From("application/json"),
		Payload:     payloadNode,
		Model: marshaller.Model[core.RequestBody]{
			Valid: true,
		},
	}

	errs := requestBody.Validate(ctx)

	// All payload data should be allowed without validation errors
	assert.Empty(t, errs, "Payload data should not be validated as Arazzo expressions")
}

func TestRequestBody_Validate_TopLevelExpression_ValidatesCorrectly(t *testing.T) {
	t.Parallel()
	// Test that top-level Arazzo expressions are properly validated

	// Test valid top-level expression
	validRequestBody := &arazzo.RequestBody{}

	validPayloadNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "$inputs.userId",
	}
	validRequestBody.Payload = validPayloadNode
	validRequestBody.Model = marshaller.Model[core.RequestBody]{
		Valid: true,
	}

	validationErrors := validRequestBody.Validate(t.Context())
	assert.Empty(t, validationErrors, "Valid top-level expression should not produce validation errors")

	// Test invalid top-level expression (valid type but invalid format)
	invalidRequestBody := &arazzo.RequestBody{}

	invalidPayloadNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "$inputs", // Missing required name after $inputs
	}
	invalidRequestBody.Payload = invalidPayloadNode
	invalidRequestBody.Model = marshaller.Model[core.RequestBody]{
		Valid: true,
	}

	validationErrors = invalidRequestBody.Validate(t.Context())
	assert.NotEmpty(t, validationErrors, "Invalid top-level expression should produce validation errors")
	assert.Contains(t, validationErrors[0].Error(), "payload expression is not valid")
}
