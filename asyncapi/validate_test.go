package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/validation"
)

func TestValidate_AbsoluteURL_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "url cannot be parsed",
			doc:      securityDoc("      type: openIdConnect\n      openIdConnectUrl: 'http://[::1'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "url is relative",
			doc:      securityDoc("      type: openIdConnect\n      openIdConnectUrl: /.well-known/openid-configuration\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
	})
}
