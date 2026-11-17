package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindings_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bindings string
		expected []string
	}{
		{
			name:     "known protocol is preserved but not validated",
			bindings: "      kafka:\n        topic: orders\n        bindingVersion: 0.5.0\n",
			expected: []string{"hint:" + validation.RuleValidationUnsupportedFormat},
		},
		{
			name:     "unknown protocol",
			bindings: "      carrier-pigeon:\n        coop: north\n",
			expected: []string{"warning:" + validation.RuleValidationUnknownProperties},
		},
		{
			name:     "extensions are not protocols",
			bindings: "      x-internal: true\n",
			expected: []string{},
		},
		{
			name:     "binding reference",
			bindings: "      $ref: '#/components/channelBindings/kafka'\n",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, validationErrs := unmarshal(t, minimalDoc("channels:\n  c:\n    bindings:\n"+tt.bindings))
			assert.Equal(t, tt.expected, diagnostics(validationErrs), "binding diagnostics should match")
			assert.True(t, a.Valid, "document should be valid")
		})
	}
}

func TestBindings_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "binding is not an object",
			doc:      minimalDoc("channels:\n  c:\n    bindings:\n      kafka: orders\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "binding version is not a string",
			doc:      minimalDoc("channels:\n  c:\n    bindings:\n      kafka:\n        bindingVersion:\n          major: 0\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "binding version is a number",
			doc:      minimalDoc("channels:\n  c:\n    bindings:\n      kafka:\n        bindingVersion: 1\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "binding version is a boolean",
			doc:      minimalDoc("channels:\n  c:\n    bindings:\n      kafka:\n        bindingVersion: true\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
	})
}

func TestBindings_GetBindingVersion_Alias_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, minimalDoc("x-versions:\n  kafka: &bv 0.5.0\nchannels:\n  c:\n    bindings:\n      kafka:\n        bindingVersion: *bv\n"))
	bindings := a.GetChannels().GetOrZero("c").GetObject().GetBindings().GetObject()
	require.NotNil(t, bindings, "bindings should be parsed")
	assert.Equal(t, "0.5.0", bindings.GetBindingVersion("kafka"), "aliased binding version should be resolved")
}

func TestBindings_GetBindingVersion_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, minimalDoc("channels:\n  c:\n    bindings:\n      kafka:\n        bindingVersion: 0.5.0\n      ws: {}\n"))
	bindings := a.GetChannels().GetOrZero("c").GetObject().GetBindings().GetObject()
	require.NotNil(t, bindings, "bindings should be parsed")
	assert.Equal(t, 2, bindings.Len(), "both protocol bindings should be parsed")
	assert.Equal(t, "0.5.0", bindings.GetBindingVersion("kafka"), "binding version should be returned")
	assert.Empty(t, bindings.GetBindingVersion("ws"), "missing binding version should be empty")
	assert.Empty(t, bindings.GetBindingVersion("mqtt"), "missing binding should be empty")
	assert.Empty(t, (*asyncapi.Bindings)(nil).GetBindingVersion("kafka"), "nil bindings should be empty")
}

func TestBindings_Mutate_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, minimalDoc("channels:\n  c:\n    bindings:\n      kafka:\n        topic: orders\n"))
	bindings := a.GetChannels().GetOrZero("c").GetObject().GetBindings().GetObject()
	bindings.Set("amqp", yml.CreateMapNode(t.Context(), nil))

	out := marshal(t, a)
	assert.Contains(t, out, "      kafka:\n        topic: orders\n      amqp: {}\n", "new binding should be added after existing bindings")
}
