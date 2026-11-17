package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/validation"
)

func TestComponents_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "component key has invalid characters",
			doc:      minimalDoc("components:\n  messages:\n    order placed:\n      name: OrderPlaced\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "component schema is validated",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      type: integer\n      default: one\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "foreign schema format is preserved but not validated",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/vnd.apache.avro;version=1.9.0\n      schema:\n        type: record\n"),
			expected: []string{"warning:" + validation.RuleValidationUnsupportedFormat},
		},
		{
			name:     "component tag is validated",
			doc:      minimalDoc("components:\n  tags:\n    t:\n      description: d\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "component external docs are validated",
			doc:      minimalDoc("components:\n  externalDocs:\n    d:\n      description: d\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "unknown component type",
			doc:      minimalDoc("components:\n  requestBodies: {}\n"),
			expected: []string{"warning:" + validation.RuleValidationUnknownProperties},
		},
	})
}
