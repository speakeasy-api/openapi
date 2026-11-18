package asyncapi_test

import (
	"context"
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingPopulateObject is an inline object whose population always fails.
type failingPopulateObject struct{}

var errPopulateFailed = errors.New("populate failed")

func (*failingPopulateObject) Populate(any) error { return errPopulateFailed }

func (*failingPopulateObject) Validate(context.Context, ...validation.Option) []error { return nil }

func TestReference_NilGetters_Success(t *testing.T) {
	t.Parallel()

	var ref *asyncapi.ReferencedServer
	assert.False(t, ref.IsReference(), "nil reference should not be a reference")
	assert.Equal(t, references.Reference(""), ref.GetReference(), "nil reference should have an empty reference")
	assert.Nil(t, ref.GetObject(), "nil reference should have no object")

	var refObj *asyncapi.ReferenceObject
	assert.Equal(t, references.Reference(""), refObj.GetReference(), "nil reference object should have an empty reference")
	assert.Nil(t, refObj.Validate(t.Context()), "nil reference object should have no diagnostics")
	assert.Nil(t, refObj.GetResolvedObject(), "nil reference object should have no resolved object")
	assert.Equal(t, references.Reference(""), refObj.GetResolvedReference(), "nil reference object should have no resolved reference")
	assert.False(t, refObj.IsResolved(), "nil reference object should not be resolved")
	assert.Nil(t, refObj.GetChannel(), "nil reference object should have no channel")
	assert.Nil(t, refObj.GetMessage(), "nil reference object should have no message")
	assert.Nil(t, refObj.GetServer(), "nil reference object should have no server")
}

func TestReference_Unresolved_Success(t *testing.T) {
	t.Parallel()

	refObj := asyncapi.NewReferenceObject("#/channels/c")
	assert.False(t, refObj.IsResolved(), "unresolved reference object should not be resolved")
	assert.Equal(t, references.Reference(""), refObj.GetResolvedReference(), "unresolved reference object should have no resolved reference")
	assert.Nil(t, refObj.GetChannel(), "unresolved reference object should have no channel")

	ref := &asyncapi.ReferencedServer{Reference: pointer.From(references.Reference("#/components/servers/s"))}
	assert.False(t, ref.IsResolved(), "unresolved reference should not be resolved")
	_, err := ref.GetNavigableNode()
	require.Error(t, err, "unresolved reference should not be navigable")
	assert.Equal(t, "reference `#/components/servers/s` is not resolved", err.Error(), "navigation error should name the reference")
}

func TestReference_Resolved_Success(t *testing.T) {
	t.Parallel()

	a, validationErrs := unmarshal(t, minimalDoc("channels:\n  c:\n    address: c\noperations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n"))
	require.Empty(t, validationErrs, "document should be valid")
	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")
	require.Empty(t, errs, "resolution should report nothing")

	channel := &a.GetOperations().GetOrZero("o").GetObject().Channel
	assert.True(t, channel.IsResolved(), "resolved reference object should be resolved")
	assert.Same(t, a.GetChannels().GetOrZero("c").GetObject(), channel.GetResolvedObject(), "resolved object should be the channel")
	assert.Nil(t, channel.GetMessage(), "a channel reference should have no message")
	assert.Nil(t, channel.GetServer(), "a channel reference should have no server")

	inline := a.GetChannels().GetOrZero("c")
	node, err := inline.GetNavigableNode()
	require.NoError(t, err, "inline object should be navigable")
	assert.Same(t, inline.GetObject(), node, "navigable node should be the inline object")
}

func TestReference_Getters_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, minimalDoc("servers:\n  inline:\n    host: example.com\n    protocol: kafka\n  ref:\n    $ref: '#/components/servers/s'\n"))

	inline := a.GetServers().GetOrZero("inline")
	assert.False(t, inline.IsReference(), "inline server should not be a reference")
	assert.Equal(t, references.Reference(""), inline.GetReference(), "inline server should have an empty reference")
	require.NotNil(t, inline.GetObject(), "inline server should have an object")
	assert.Equal(t, "example.com", inline.GetObject().GetHost(), "inline server should be populated")

	ref := a.GetServers().GetOrZero("ref")
	assert.True(t, ref.IsReference(), "reference should be a reference")
	assert.Equal(t, references.Reference("#/components/servers/s"), ref.GetReference(), "reference should be returned")
	assert.Nil(t, ref.GetObject(), "reference should have no object")

	assert.Equal(t, references.Reference("#/channels/c"), asyncapi.NewReferenceObject("#/channels/c").GetReference(), "reference object should return its reference")
}

func TestReference_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "reference has an invalid JSON pointer",
			doc:      minimalDoc("servers:\n  s:\n    $ref: '#'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
		{
			name:     "reference object has an invalid JSON pointer",
			doc:      minimalDoc("channels:\n  c:\n    address: c\noperations:\n  o:\n    action: send\n    channel:\n      $ref: '#'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
	})
}

func TestReference_ValidateNil_Error(t *testing.T) {
	t.Parallel()

	var ref *asyncapi.ReferencedServer
	assert.Equal(t, []string{"reference is nil"}, errorStrings(ref.Validate(t.Context())), "nil reference should report an error")
}

func TestReference_Populate_Success(t *testing.T) {
	t.Parallel()

	ref := &asyncapi.ReferencedServer{}
	source := core.Reference[*core.Server]{Reference: marshaller.Node[*string]{Present: true, Value: pointer.From("#/components/servers/s")}}
	require.NoError(t, ref.Populate(source), "populating from a core reference value should succeed")
	assert.Equal(t, references.Reference("#/components/servers/s"), ref.GetReference(), "reference should be populated")
}

func TestReference_Populate_Error(t *testing.T) {
	t.Parallel()

	marshaller.RegisterType(func() *failingPopulateObject { return &failingPopulateObject{} })

	tests := []struct {
		name     string
		ref      interface{ Populate(source any) error }
		source   any
		expected string
	}{
		{
			name:     "unexpected source type",
			ref:      &asyncapi.ReferencedServer{},
			source:   "server",
			expected: "expected *core.Reference[C] or core.Reference[C], got string",
		},
		{
			name:     "inline object fails to populate",
			ref:      &asyncapi.Reference[failingPopulateObject, *failingPopulateObject, *core.Server]{},
			source:   &core.Reference[*core.Server]{Object: &core.Server{}},
			expected: errPopulateFailed.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ref.Populate(tt.source)
			require.Error(t, err, "populate should fail")
			assert.Equal(t, tt.expected, err.Error(), "populate error should match")
		})
	}
}
