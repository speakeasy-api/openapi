package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/validation"
)

func TestServer_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "missing host and protocol",
			doc:      minimalDoc("servers:\n  prod:\n    description: d\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField, "error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "host includes a scheme",
			doc:      minimalDoc("servers:\n  prod:\n    host: mqtt://broker.example.com\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "host variable is not defined",
			doc:      minimalDoc("servers:\n  prod:\n    host: '{region}.example.com'\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "pathname variable is not defined",
			doc:      minimalDoc("servers:\n  prod:\n    host: example.com\n    pathname: /{tenant}\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "host with an empty expression",
			doc:      minimalDoc("servers:\n  prod:\n    host: '{}.example.com'\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "variable default is not in enum",
			doc:      minimalDoc("servers:\n  prod:\n    host: '{region}.example.com'\n    protocol: mqtt\n    variables:\n      region:\n        enum: [eu, us]\n        default: au\n"),
			expected: []string{"error:" + validation.RuleValidationAllowedValues},
		},
		{
			name:     "server name has invalid characters",
			doc:      minimalDoc("servers:\n  prod server:\n    host: example.com\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "duplicate server tags",
			doc:      minimalDoc("servers:\n  prod:\n    host: example.com\n    protocol: mqtt\n    tags:\n      - name: a\n      - name: a\n"),
			expected: []string{"error:" + validation.RuleValidationDuplicateKey},
		},
	})
}

func TestServer_Validate_Success(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "variables defined in host and pathname",
			doc:      minimalDoc("servers:\n  prod:\n    host: '{region}.example.com:{port}'\n    pathname: /{tenant}\n    protocol: mqtt\n    variables:\n      region:\n        default: eu\n      port:\n        $ref: '#/components/serverVariables/port'\n      tenant: {}\n"),
			expected: []string{},
		},
		{
			name:     "server reference",
			doc:      minimalDoc("servers:\n  prod:\n    $ref: '#/components/servers/prod'\n"),
			expected: []string{},
		},
	})
}
