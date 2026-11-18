package asyncapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// OperationAction is the action an application performs for an operation.
type OperationAction string

const (
	// OperationActionSend means the application sends messages to the channel.
	OperationActionSend OperationAction = "send"
	// OperationActionReceive means the application receives messages from the channel.
	OperationActionReceive OperationAction = "receive"
)

// Operation describes a specific operation. Operations are identified by their key in the operations map; there is
// no separate operation ID field.
type Operation struct {
	marshaller.Model[core.Operation]

	// Action is whether the application sends or receives messages. Required.
	Action OperationAction
	// Channel is a reference to the channel in the root channels map this operation is performed on. Required.
	Channel ReferenceObject
	// Title is a human-friendly title for the operation.
	Title *string
	// Summary is a short summary of the operation.
	Summary *string
	// Description is a description of the operation. May contain CommonMark syntax.
	Description *string
	// Security are the security schemes associated with this operation. Only one needs to be satisfied.
	Security []*ReferencedSecurityScheme
	// Tags are used for logical grouping and categorization of operations.
	Tags []*Tag
	// ExternalDocs is additional external documentation for this operation.
	ExternalDocs *ExternalDocumentation
	// Bindings are protocol-specific definitions for the operation.
	Bindings *ReferencedBindings
	// Traits are applied to the operation, in order, using the traits merge mechanism.
	Traits []*ReferencedOperationTrait
	// Messages are references to the channel's messages this operation supports. When nil (omitted) all of the
	// channel's messages are supported; when empty no messages are.
	Messages []*ReferenceObject
	// Reply is the definition of the reply in a request-reply operation.
	Reply *ReferencedOperationReply
	// Extensions provides a list of extensions to the Operation Object.
	Extensions *extensions.Extensions

	// effective is set by the last call to AsyncAPI.ResolveAllReferences, guarded by resolutionMu.
	effective *effectiveView[Operation]
}

var _ interfaces.Model[core.Operation] = (*Operation)(nil)

// GetEffective returns the operation with its traits applied, as computed by the last call to
// AsyncAPI.ResolveAllReferences. Returns nil if the operation has traits that haven't been applied.
//
// For an operation with traits the result is a separate snapshot: editing it has no effect on the document. An
// operation without traits is its own effective view, so GetEffective returns the operation itself and editing the
// result edits the document.
func (o *Operation) GetEffective() *Operation {
	if o == nil {
		return nil
	}
	if len(o.Traits) == 0 {
		return o
	}
	return o.effectiveView().get()
}

// GetTraitSource reports where a top-level field of the effective operation came from: the index in Traits of the
// trait that supplied it, or -1 if the operation sets it itself. ok is false if the field isn't in the effective
// operation or traits haven't been applied. For an operation without traits, fields written in the document report -1.
func (o *Operation) GetTraitSource(field string) (index int, ok bool) {
	if o == nil {
		return 0, false
	}
	if len(o.Traits) == 0 {
		return authoredSource(o.GetCore().GetRootNode(), field)
	}
	return o.effectiveView().source(field)
}

func (o *Operation) effectiveView() *effectiveView[Operation] {
	if o == nil {
		return nil
	}
	resolutionMu.RLock()
	defer resolutionMu.RUnlock()
	return o.effective
}

func (o *Operation) setEffective(view *effectiveView[Operation]) {
	resolutionMu.Lock()
	defer resolutionMu.Unlock()
	o.effective = view
}

// GetAction returns the action, or an empty action if not set.
func (o *Operation) GetAction() OperationAction {
	if o == nil {
		return ""
	}
	return o.Action
}

// GetChannel returns the channel reference, or nil if the operation is nil.
func (o *Operation) GetChannel() *ReferenceObject {
	if o == nil {
		return nil
	}
	return &o.Channel
}

// GetTitle returns the title, or an empty string if not set.
func (o *Operation) GetTitle() string {
	if o == nil || o.Title == nil {
		return ""
	}
	return *o.Title
}

// GetSummary returns the summary, or an empty string if not set.
func (o *Operation) GetSummary() string {
	if o == nil || o.Summary == nil {
		return ""
	}
	return *o.Summary
}

// GetDescription returns the description, or an empty string if not set.
func (o *Operation) GetDescription() string {
	if o == nil || o.Description == nil {
		return ""
	}
	return *o.Description
}

// GetSecurity returns the security schemes, or nil if not set.
func (o *Operation) GetSecurity() []*ReferencedSecurityScheme {
	if o == nil {
		return nil
	}
	return o.Security
}

// GetTags returns the tags, or nil if not set.
func (o *Operation) GetTags() []*Tag {
	if o == nil {
		return nil
	}
	return o.Tags
}

// GetExternalDocs returns the external documentation, or nil if not set.
func (o *Operation) GetExternalDocs() *ExternalDocumentation {
	if o == nil {
		return nil
	}
	return o.ExternalDocs
}

// GetBindings returns the operation bindings, or nil if not set.
func (o *Operation) GetBindings() *ReferencedBindings {
	if o == nil {
		return nil
	}
	return o.Bindings
}

// GetTraits returns the operation traits, or nil if not set.
func (o *Operation) GetTraits() []*ReferencedOperationTrait {
	if o == nil {
		return nil
	}
	return o.Traits
}

// GetMessages returns the message references. A nil result means messages were omitted, so all of the channel's
// messages apply; an empty result means no messages apply.
func (o *Operation) GetMessages() []*ReferenceObject {
	if o == nil {
		return nil
	}
	return o.Messages
}

// GetReply returns the reply, or nil if not set.
func (o *Operation) GetReply() *ReferencedOperationReply {
	if o == nil {
		return nil
	}
	return o.Reply
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (o *Operation) GetExtensions() *extensions.Extensions {
	if o == nil || o.Extensions == nil {
		return extensions.New()
	}
	return o.Extensions
}

// Validate will validate the Operation Object against the AsyncAPI Specification.
func (o *Operation) Validate(ctx context.Context, opts ...validation.Option) []error {
	if o == nil {
		return nil
	}

	c := o.GetCore()
	errs := []error{}

	switch o.Action {
	case OperationActionSend, OperationActionReceive:
	case "":
		errs = append(errs, requiredString(string(o.Action), c.Action.Present, c, c.Action, "operation.action")...)
	default:
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("operation.action must be one of [`%s`, `%s`], got `%s`", OperationActionSend, OperationActionReceive, o.Action), c, c.Action))
	}

	if o.Channel.Reference == "" && c.GetRootNode() == nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`operation.channel` is required"), c, c.Channel))
	} else {
		errs = append(errs, o.Channel.Validate(ctx, opts...)...)
	}

	errs = append(errs, validateOperationFields(ctx, o.Security, o.Tags, o.ExternalDocs, o.Bindings, o.Reply, c, c.Tags, "operation", opts...)...)

	for _, trait := range o.Traits {
		errs = append(errs, trait.Validate(ctx, opts...)...)
	}
	for _, message := range o.Messages {
		errs = append(errs, message.Validate(ctx, opts...)...)
	}

	errs = append(errs, unknownPropertyErrors(c, "operation")...)

	o.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// validateOperationFields validates the fields shared by Operation and Operation Trait Objects.
func validateOperationFields(ctx context.Context, security []*ReferencedSecurityScheme, tags []*Tag, externalDocs *ExternalDocumentation, bindings *ReferencedBindings, reply *ReferencedOperationReply, c validation.CoreModeler, tagsNode validation.SliceNodeGetter, object string, opts ...validation.Option) []error {
	errs := []error{}
	for _, scheme := range security {
		errs = append(errs, scheme.Validate(ctx, opts...)...)
	}
	errs = append(errs, validateTags(ctx, tags, c, tagsNode, object+".tags", opts...)...)
	if externalDocs != nil {
		errs = append(errs, externalDocs.Validate(ctx, opts...)...)
	}
	if bindings != nil {
		errs = append(errs, bindings.Validate(ctx, opts...)...)
	}
	if reply != nil {
		errs = append(errs, reply.Validate(ctx, opts...)...)
	}
	return errs
}

// OperationTrait describes a trait that may be applied to an Operation Object. It can contain any operation property
// except action, channel, messages and traits.
type OperationTrait struct {
	marshaller.Model[core.OperationTrait]

	// Title is a human-friendly title for the operation.
	Title *string
	// Summary is a short summary of the operation.
	Summary *string
	// Description is a description of the operation. May contain CommonMark syntax.
	Description *string
	// Security are the security schemes associated with the operation.
	Security []*ReferencedSecurityScheme
	// Tags are used for logical grouping and categorization of operations.
	Tags []*Tag
	// ExternalDocs is additional external documentation for the operation.
	ExternalDocs *ExternalDocumentation
	// Bindings are protocol-specific definitions for the operation.
	Bindings *ReferencedBindings
	// Reply is the definition of the reply in a request-reply operation.
	Reply *ReferencedOperationReply
	// Extensions provides a list of extensions to the Operation Trait Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.OperationTrait] = (*OperationTrait)(nil)

// GetTitle returns the title, or an empty string if not set.
func (t *OperationTrait) GetTitle() string {
	if t == nil || t.Title == nil {
		return ""
	}
	return *t.Title
}

// GetSummary returns the summary, or an empty string if not set.
func (t *OperationTrait) GetSummary() string {
	if t == nil || t.Summary == nil {
		return ""
	}
	return *t.Summary
}

// GetDescription returns the description, or an empty string if not set.
func (t *OperationTrait) GetDescription() string {
	if t == nil || t.Description == nil {
		return ""
	}
	return *t.Description
}

// GetSecurity returns the security schemes, or nil if not set.
func (t *OperationTrait) GetSecurity() []*ReferencedSecurityScheme {
	if t == nil {
		return nil
	}
	return t.Security
}

// GetTags returns the tags, or nil if not set.
func (t *OperationTrait) GetTags() []*Tag {
	if t == nil {
		return nil
	}
	return t.Tags
}

// GetExternalDocs returns the external documentation, or nil if not set.
func (t *OperationTrait) GetExternalDocs() *ExternalDocumentation {
	if t == nil {
		return nil
	}
	return t.ExternalDocs
}

// GetBindings returns the operation bindings, or nil if not set.
func (t *OperationTrait) GetBindings() *ReferencedBindings {
	if t == nil {
		return nil
	}
	return t.Bindings
}

// GetReply returns the reply, or nil if not set.
func (t *OperationTrait) GetReply() *ReferencedOperationReply {
	if t == nil {
		return nil
	}
	return t.Reply
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (t *OperationTrait) GetExtensions() *extensions.Extensions {
	if t == nil || t.Extensions == nil {
		return extensions.New()
	}
	return t.Extensions
}

// Validate will validate the Operation Trait Object against the AsyncAPI Specification.
func (t *OperationTrait) Validate(ctx context.Context, opts ...validation.Option) []error {
	if t == nil {
		return nil
	}

	c := t.GetCore()
	errs := validateOperationFields(ctx, t.Security, t.Tags, t.ExternalDocs, t.Bindings, t.Reply, c, c.Tags, "operationTrait", opts...)
	errs = append(errs, unknownPropertyErrors(c, "operationTrait", "action", "channel", "messages", "traits")...)

	t.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// OperationReply describes the reply part of a request-reply operation.
type OperationReply struct {
	marshaller.Model[core.OperationReply]

	// Address is where the reply should be sent, determined at runtime.
	Address *ReferencedOperationReplyAddress
	// Channel is a reference to the channel used for the reply.
	Channel *ReferenceObject
	// Messages are references to the channel's messages that can be used as the reply.
	Messages []*ReferenceObject
	// Extensions provides a list of extensions to the Operation Reply Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.OperationReply] = (*OperationReply)(nil)

// GetAddress returns the reply address, or nil if not set.
func (r *OperationReply) GetAddress() *ReferencedOperationReplyAddress {
	if r == nil {
		return nil
	}
	return r.Address
}

// GetChannel returns the reply channel reference, or nil if not set.
func (r *OperationReply) GetChannel() *ReferenceObject {
	if r == nil {
		return nil
	}
	return r.Channel
}

// GetMessages returns the reply message references, or nil if not set.
func (r *OperationReply) GetMessages() []*ReferenceObject {
	if r == nil {
		return nil
	}
	return r.Messages
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (r *OperationReply) GetExtensions() *extensions.Extensions {
	if r == nil || r.Extensions == nil {
		return extensions.New()
	}
	return r.Extensions
}

// Validate will validate the Operation Reply Object against the AsyncAPI Specification.
func (r *OperationReply) Validate(ctx context.Context, opts ...validation.Option) []error {
	if r == nil {
		return nil
	}

	c := r.GetCore()
	errs := []error{}

	if r.Address != nil {
		errs = append(errs, r.Address.Validate(ctx, opts...)...)
	}
	if r.Channel != nil {
		errs = append(errs, r.Channel.Validate(ctx, opts...)...)
	}
	for _, message := range r.Messages {
		errs = append(errs, message.Validate(ctx, opts...)...)
	}
	errs = append(errs, unknownPropertyErrors(c, "operationReply")...)

	r.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// OperationReplyAddress is a runtime expression identifying where a reply should be sent.
type OperationReplyAddress struct {
	marshaller.Model[core.OperationReplyAddress]

	// Description is a description of the address. May contain CommonMark syntax.
	Description *string
	// Location is a runtime expression specifying the location of the reply address, e.g. `$message.header#/replyTo`. Required.
	Location string
	// Extensions provides a list of extensions to the Operation Reply Address Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.OperationReplyAddress] = (*OperationReplyAddress)(nil)

// GetDescription returns the description, or an empty string if not set.
func (a *OperationReplyAddress) GetDescription() string {
	if a == nil || a.Description == nil {
		return ""
	}
	return *a.Description
}

// GetLocation returns the location runtime expression, or an empty string if not set.
func (a *OperationReplyAddress) GetLocation() string {
	if a == nil {
		return ""
	}
	return a.Location
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (a *OperationReplyAddress) GetExtensions() *extensions.Extensions {
	if a == nil || a.Extensions == nil {
		return extensions.New()
	}
	return a.Extensions
}

// Validate will validate the Operation Reply Address Object against the AsyncAPI Specification.
func (a *OperationReplyAddress) Validate(_ context.Context, _ ...validation.Option) []error {
	if a == nil {
		return nil
	}

	c := a.GetCore()
	errs := validateLocation(a.Location, c.Location.Present, c, c.Location, "operationReplyAddress.location")
	errs = append(errs, unknownPropertyErrors(c, "operationReplyAddress")...)

	a.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// validateLocation validates a required runtime expression location.
func validateLocation(location string, present bool, c validation.CoreModeler, node validation.ValueNodeGetter, field string) []error {
	if location == "" {
		return requiredString(location, present, c, node, field)
	}
	if err := validateRuntimeExpression(location); err != nil {
		return []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("%s: %w", field, err), c, node)}
	}
	return nil
}
