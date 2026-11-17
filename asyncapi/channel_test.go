package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/validation"
)

func TestChannel_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "address with a query",
			doc:      minimalDoc("channels:\n  c:\n    address: orders?x=1\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "address parameter is not defined",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{id}\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "repeated undefined address parameter is reported once",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{id}.{id}\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "address with an empty expression",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{}\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "parameter is not used in the address",
			doc:      minimalDoc("channels:\n  c:\n    address: orders\n    parameters:\n      id: {}\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidTarget},
		},
		{
			name:     "parameter name has invalid characters",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{a.b}\n    parameters:\n      a.b: {}\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "parameter location is not a runtime expression",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{id}\n    parameters:\n      id:\n        location: $request.body#/id\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidSyntax},
		},
		{
			name:     "parameter default is not in enum",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{id}\n    parameters:\n      id:\n        enum: ['1', '2']\n        default: '3'\n"),
			expected: []string{"error:" + validation.RuleValidationAllowedValues},
		},
		{
			name:     "server reference without $ref",
			doc:      minimalDoc("channels:\n  c:\n    servers:\n      - $ref: ''\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "null channel reference",
			doc:      minimalDoc("channels:\n  c:\n    $ref: null\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "inline message is validated",
			doc:      minimalDoc("channels:\n  c:\n    messages:\n      m:\n        contentType: 'json;;'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
	})
}

func TestChannel_Validate_Success(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "address parameters defined",
			doc:      minimalDoc("channels:\n  c:\n    address: users.{userId}.{event}\n    parameters:\n      userId:\n        location: $message.payload#/user/id\n      event:\n        $ref: '#/components/parameters/event'\n"),
			expected: []string{},
		},
		{
			name:     "address with an MQTT multi-level wildcard",
			doc:      minimalDoc("channels:\n  c:\n    address: sensors/#\n"),
			expected: []string{},
		},
		{
			name:     "null address",
			doc:      minimalDoc("channels:\n  c:\n    address: null\n"),
			expected: []string{},
		},
		{
			name:     "channel without address",
			doc:      minimalDoc("channels:\n  c: {}\n"),
			expected: []string{},
		},
	})
}
