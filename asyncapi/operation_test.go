package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
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

func TestOperation_Validate_Nil_Success(t *testing.T) {
	t.Parallel()

	runNilValidateCases(t, []nilValidateCase{
		{name: "operation", model: (*asyncapi.Operation)(nil)},
		{name: "operation trait", model: (*asyncapi.OperationTrait)(nil)},
		{name: "operation reply", model: (*asyncapi.OperationReply)(nil)},
		{name: "operation reply address", model: (*asyncapi.OperationReplyAddress)(nil)},
	})
}

func TestOperation_GetEffective_Success(t *testing.T) {
	t.Parallel()

	untraited := &asyncapi.Operation{Action: asyncapi.OperationActionSend}
	assert.Same(t, untraited, untraited.GetEffective(), "an operation without traits should be its own effective view")
	assert.Nil(t, (*asyncapi.Operation)(nil).GetEffective(), "a nil operation should have no effective view")

	unapplied := &asyncapi.Operation{Traits: []*asyncapi.ReferencedOperationTrait{{Object: &asyncapi.OperationTrait{}}}}
	assert.Nil(t, unapplied.GetEffective(), "an operation whose traits haven't been applied should have no effective view")
}

func TestOperation_GetTraitSource_NotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation *asyncapi.Operation
	}{
		{
			name:      "nil operation",
			operation: nil,
		},
		{
			name:      "traits not applied",
			operation: &asyncapi.Operation{Summary: pointer.From("s"), Traits: []*asyncapi.ReferencedOperationTrait{{Object: &asyncapi.OperationTrait{}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			index, ok := tt.operation.GetTraitSource("summary")
			assert.False(t, ok, "field should have no source")
			assert.Zero(t, index, "index should be zero without a source")
		})
	}
}

func TestOperation_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.Operation{
		Action:       asyncapi.OperationActionSend,
		Channel:      *asyncapi.NewReferenceObject("#/channels/orders"),
		Title:        pointer.From("Place order"),
		Summary:      pointer.From("Places an order."),
		Description:  pointer.From("Sends an order to the broker."),
		Security:     []*asyncapi.ReferencedSecurityScheme{{Object: &asyncapi.SecurityScheme{Type: asyncapi.SecuritySchemeTypePlain}}},
		Tags:         []*asyncapi.Tag{{Name: pointer.From("orders")}},
		ExternalDocs: &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com/docs")},
		Bindings:     &asyncapi.ReferencedBindings{Object: &asyncapi.Bindings{}},
		Traits:       []*asyncapi.ReferencedOperationTrait{{Object: &asyncapi.OperationTrait{}}},
		Messages:     []*asyncapi.ReferenceObject{asyncapi.NewReferenceObject("#/channels/orders/messages/placed")},
		Reply:        &asyncapi.ReferencedOperationReply{Object: &asyncapi.OperationReply{}},
		Extensions:   exts,
	}
	unset := &asyncapi.Operation{}

	type getters struct {
		action       asyncapi.OperationAction
		channel      *asyncapi.ReferenceObject
		title        string
		summary      string
		description  string
		security     []*asyncapi.ReferencedSecurityScheme
		tags         []*asyncapi.Tag
		externalDocs *asyncapi.ExternalDocumentation
		bindings     *asyncapi.ReferencedBindings
		traits       []*asyncapi.ReferencedOperationTrait
		messages     []*asyncapi.ReferenceObject
		reply        *asyncapi.ReferencedOperationReply
		extensions   *extensions.Extensions
	}

	tests := []struct {
		name      string
		operation *asyncapi.Operation
		expected  getters
	}{
		{
			name:      "nil operation returns defaults",
			operation: nil,
			expected:  getters{extensions: extensions.New()},
		},
		{
			name:      "unset fields return defaults",
			operation: unset,
			expected:  getters{channel: &unset.Channel, extensions: extensions.New()},
		},
		{
			name:      "populated fields are returned",
			operation: populated,
			expected: getters{
				action:       asyncapi.OperationActionSend,
				channel:      &populated.Channel,
				title:        "Place order",
				summary:      "Places an order.",
				description:  "Sends an order to the broker.",
				security:     populated.Security,
				tags:         populated.Tags,
				externalDocs: populated.ExternalDocs,
				bindings:     populated.Bindings,
				traits:       populated.Traits,
				messages:     populated.Messages,
				reply:        populated.Reply,
				extensions:   exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				action:       tt.operation.GetAction(),
				channel:      tt.operation.GetChannel(),
				title:        tt.operation.GetTitle(),
				summary:      tt.operation.GetSummary(),
				description:  tt.operation.GetDescription(),
				security:     tt.operation.GetSecurity(),
				tags:         tt.operation.GetTags(),
				externalDocs: tt.operation.GetExternalDocs(),
				bindings:     tt.operation.GetBindings(),
				traits:       tt.operation.GetTraits(),
				messages:     tt.operation.GetMessages(),
				reply:        tt.operation.GetReply(),
				extensions:   tt.operation.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestOperationTrait_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.OperationTrait{
		Title:        pointer.From("Traced"),
		Summary:      pointer.From("Adds tracing."),
		Description:  pointer.From("Applied to traced operations."),
		Security:     []*asyncapi.ReferencedSecurityScheme{{Object: &asyncapi.SecurityScheme{Type: asyncapi.SecuritySchemeTypePlain}}},
		Tags:         []*asyncapi.Tag{{Name: pointer.From("tracing")}},
		ExternalDocs: &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com/docs")},
		Bindings:     &asyncapi.ReferencedBindings{Object: &asyncapi.Bindings{}},
		Reply:        &asyncapi.ReferencedOperationReply{Object: &asyncapi.OperationReply{}},
		Extensions:   exts,
	}

	type getters struct {
		title        string
		summary      string
		description  string
		security     []*asyncapi.ReferencedSecurityScheme
		tags         []*asyncapi.Tag
		externalDocs *asyncapi.ExternalDocumentation
		bindings     *asyncapi.ReferencedBindings
		reply        *asyncapi.ReferencedOperationReply
		extensions   *extensions.Extensions
	}

	tests := []struct {
		name     string
		trait    *asyncapi.OperationTrait
		expected getters
	}{
		{
			name:     "nil trait returns defaults",
			trait:    nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			trait:    &asyncapi.OperationTrait{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:  "populated fields are returned",
			trait: populated,
			expected: getters{
				title:        "Traced",
				summary:      "Adds tracing.",
				description:  "Applied to traced operations.",
				security:     populated.Security,
				tags:         populated.Tags,
				externalDocs: populated.ExternalDocs,
				bindings:     populated.Bindings,
				reply:        populated.Reply,
				extensions:   exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				title:        tt.trait.GetTitle(),
				summary:      tt.trait.GetSummary(),
				description:  tt.trait.GetDescription(),
				security:     tt.trait.GetSecurity(),
				tags:         tt.trait.GetTags(),
				externalDocs: tt.trait.GetExternalDocs(),
				bindings:     tt.trait.GetBindings(),
				reply:        tt.trait.GetReply(),
				extensions:   tt.trait.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestOperationReply_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.OperationReply{
		Address:    &asyncapi.ReferencedOperationReplyAddress{Object: &asyncapi.OperationReplyAddress{Location: "$message.header#/replyTo"}},
		Channel:    asyncapi.NewReferenceObject("#/channels/replies"),
		Messages:   []*asyncapi.ReferenceObject{asyncapi.NewReferenceObject("#/channels/replies/messages/ack")},
		Extensions: exts,
	}

	type getters struct {
		address    *asyncapi.ReferencedOperationReplyAddress
		channel    *asyncapi.ReferenceObject
		messages   []*asyncapi.ReferenceObject
		extensions *extensions.Extensions
	}

	tests := []struct {
		name     string
		reply    *asyncapi.OperationReply
		expected getters
	}{
		{
			name:     "nil reply returns defaults",
			reply:    nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			reply:    &asyncapi.OperationReply{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:  "populated fields are returned",
			reply: populated,
			expected: getters{
				address:    populated.Address,
				channel:    populated.Channel,
				messages:   populated.Messages,
				extensions: exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				address:    tt.reply.GetAddress(),
				channel:    tt.reply.GetChannel(),
				messages:   tt.reply.GetMessages(),
				extensions: tt.reply.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestOperationReplyAddress_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))

	type getters struct {
		description string
		location    string
		extensions  *extensions.Extensions
	}

	tests := []struct {
		name     string
		address  *asyncapi.OperationReplyAddress
		expected getters
	}{
		{
			name:     "nil address returns defaults",
			address:  nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			address:  &asyncapi.OperationReplyAddress{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			address: &asyncapi.OperationReplyAddress{
				Description: pointer.From("Reply topic."),
				Location:    "$message.header#/replyTo",
				Extensions:  exts,
			},
			expected: getters{
				description: "Reply topic.",
				location:    "$message.header#/replyTo",
				extensions:  exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				description: tt.address.GetDescription(),
				location:    tt.address.GetLocation(),
				extensions:  tt.address.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}
