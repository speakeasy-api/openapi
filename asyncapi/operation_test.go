package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
)

func TestOperation_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "invalid action",
			doc:      minimalDoc("operations:\n  o:\n    action: publish\n    channel:\n      $ref: '#/channels/c'\n"),
			expected: []string{"error:" + validation.RuleValidationAllowedValues},
		},
		{
			name:     "missing action and channel",
			doc:      minimalDoc("operations:\n  o:\n    title: t\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField, "error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "reply address location is not a runtime expression",
			doc:      minimalDoc("operations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    reply:\n      address:\n        location: replies\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidSyntax},
		},
		{
			name:     "reply address without location",
			doc:      minimalDoc("operations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    reply:\n      address:\n        description: d\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "inline trait is validated",
			doc:      minimalDoc("operations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    traits:\n      - tags:\n          - name: a\n          - name: a\n"),
			expected: []string{"error:" + validation.RuleValidationDuplicateKey},
		},
		{
			name:     "empty channel reference object",
			doc:      minimalDoc("operations:\n  o:\n    action: send\n    channel: {}\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "operation trait sets forbidden fields",
			doc:      minimalDoc("components:\n  operationTraits:\n    t:\n      action: send\n      channel:\n        $ref: '#/channels/c'\n      messages: []\n      traits: []\n"),
			expected: []string{"error:" + validation.RuleValidationUnknownProperties, "error:" + validation.RuleValidationUnknownProperties, "error:" + validation.RuleValidationUnknownProperties, "error:" + validation.RuleValidationUnknownProperties},
		},
		{
			name:     "unknown operation property",
			doc:      minimalDoc("operations:\n  o:\n    action: send\n    operationId: legacy\n    channel:\n      $ref: '#/channels/c'\n"),
			expected: []string{"warning:" + validation.RuleValidationUnknownProperties},
		},
	})
}

func TestOperation_Validate_Success(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "operation trait with reply",
			doc:      minimalDoc("components:\n  operationTraits:\n    t:\n      reply:\n        channel:\n          $ref: '#/channels/replies'\n"),
			expected: []string{},
		},
		{
			name:     "request reply operation",
			doc:      minimalDoc("operations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    reply:\n      $ref: '#/components/replies/r'\n"),
			expected: []string{},
		},
	})
}

func TestOperation_Validate_BuiltInCode_Error(t *testing.T) {
	t.Parallel()

	operation := &asyncapi.Operation{}
	errs := operation.Validate(t.Context())
	assert.Equal(t, []string{"error:" + validation.RuleValidationRequiredField, "error:" + validation.RuleValidationRequiredField}, diagnostics(errs), "action and channel should be required")
	assert.False(t, operation.Valid, "operation should be invalid")

	operation.Action = asyncapi.OperationActionReceive
	operation.Channel = *asyncapi.NewReferenceObject("#/channels/c")
	assert.Empty(t, operation.Validate(t.Context()), "complete operation should be valid")
	assert.True(t, operation.Valid, "complete operation should be valid")
}
