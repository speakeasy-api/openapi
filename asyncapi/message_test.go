package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/validation"
)

func TestMessage_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "headers are not an object schema",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      headers:\n        type: string\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "invalid content type",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      contentType: 'json;;'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "correlation ID location is not a runtime expression",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      correlationId:\n        location: $message.body#/id\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidSyntax},
		},
		{
			name:     "correlation ID location has an invalid pointer",
			doc:      minimalDoc("components:\n  correlationIds:\n    c:\n      location: '$message.payload#id'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidSyntax},
		},
		{
			name:     "example without headers or payload",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      examples:\n        - name: empty\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "example headers are not an object",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      examples:\n        - headers: [1]\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "payload schema is validated",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      payload:\n        type: string\n        default: 1\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "message trait headers are not an object schema",
			doc:      minimalDoc("components:\n  messageTraits:\n    t:\n      headers:\n        type: array\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "message trait cannot set a payload",
			doc:      minimalDoc("components:\n  messageTraits:\n    t:\n      payload:\n        type: object\n"),
			expected: []string{"warning:" + validation.RuleValidationUnknownProperties},
		},
	})
}

func TestMessage_Validate_Success(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "multi format headers and payload",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      headers:\n        schemaFormat: application/schema+json;version=draft-07\n        schema:\n          type: object\n      payload:\n        schemaFormat: application/vnd.aai.asyncapi+json;version=3.1.0\n        schema:\n          type: string\n"),
			expected: []string{},
		},
		{
			name:     "example with payload only",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      examples:\n        - payload: 42\n"),
			expected: []string{},
		},
		{
			name:     "correlation ID for the whole header",
			doc:      minimalDoc("components:\n  correlationIds:\n    c:\n      location: $message.header\n"),
			expected: []string{},
		},
	})
}
