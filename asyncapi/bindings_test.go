package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
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

func TestBindings_Getters_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bindings *asyncapi.Bindings
	}{
		{name: "nil bindings", bindings: nil},
		{name: "bindings without a map", bindings: &asyncapi.Bindings{}},
		{name: "new bindings", bindings: asyncapi.NewBindings()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			protocols := []string{}
			for protocol := range tt.bindings.All() {
				protocols = append(protocols, protocol)
			}
			assert.Equal(t, 0, tt.bindings.Len(), "bindings should be empty")
			assert.Empty(t, protocols, "bindings should have no protocols")
			assert.Equal(t, 0, tt.bindings.GetExtensions().Len(), "bindings should have empty extensions")
		})
	}
}

func TestBindings_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	ext := extensions.New(extensions.NewElem("x-owner", yml.CreateStringNode("platform")))
	bindings := &asyncapi.Bindings{Extensions: ext}
	assert.Same(t, ext, bindings.GetExtensions(), "set extensions should be returned")
}

func TestBindings_ValidateNil_Success(t *testing.T) {
	t.Parallel()

	var bindings *asyncapi.Bindings
	assert.Nil(t, bindings.Validate(t.Context()), "nil bindings should have no diagnostics")
}

func TestBindings_ValidateWithoutDocumentVersion_Success(t *testing.T) {
	t.Parallel()

	bindings := asyncapi.NewBindings()
	bindings.Set("kafka", values.Value(nil))
	bindings.Set("ros2", yml.CreateMapNode(t.Context(), nil))

	assert.Equal(t, []string{
		"hint:" + validation.RuleValidationUnsupportedFormat,
		"hint:" + validation.RuleValidationUnsupportedFormat,
	}, diagnostics(bindings.Validate(t.Context())), "bindings built in code should be validated as the latest version")
	assert.True(t, bindings.Valid, "bindings should be valid")
}

func TestBindings_Mutate_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, minimalDoc("channels:\n  c:\n    bindings:\n      kafka:\n        topic: orders\n"))
	bindings := a.GetChannels().GetOrZero("c").GetObject().GetBindings().GetObject()
	bindings.Set("amqp", yml.CreateMapNode(t.Context(), nil))

	out := marshal(t, a)
	assert.Contains(t, out, "      kafka:\n        topic: orders\n      amqp: {}\n", "new binding should be added after existing bindings")
}
