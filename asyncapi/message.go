package asyncapi

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

// Message describes a message received on a given channel and operation.
type Message struct {
	marshaller.Model[core.Message]

	// Headers is the schema of the application headers. It must be a schema of type object.
	Headers *jsonschema.SchemaUnion
	// Payload is the schema of the message payload.
	Payload *jsonschema.SchemaUnion
	// CorrelationID is the definition of the correlation ID used for message tracing or matching.
	CorrelationID *ReferencedCorrelationID
	// ContentType is the content type to use when encoding/decoding the payload, e.g. `application/json`.
	ContentType *string
	// Name is a machine-friendly name for the message.
	Name *string
	// Title is a human-friendly title for the message.
	Title *string
	// Summary is a short summary of what the message is about.
	Summary *string
	// Description is a description of the message. May contain CommonMark syntax.
	Description *string
	// Tags are used for logical grouping and categorization of messages.
	Tags []*Tag
	// ExternalDocs is additional external documentation for this message.
	ExternalDocs *ExternalDocumentation
	// Bindings are protocol-specific definitions for the message.
	Bindings *ReferencedBindings
	// Examples are examples of the message.
	Examples []*MessageExample
	// Traits are applied to the message, in order, using the traits merge mechanism.
	Traits []*ReferencedMessageTrait
	// Extensions provides a list of extensions to the Message Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Message] = (*Message)(nil)

// GetHeaders returns the headers schema, or nil if not set.
func (m *Message) GetHeaders() *jsonschema.SchemaUnion {
	if m == nil {
		return nil
	}
	return m.Headers
}

// GetPayload returns the payload schema, or nil if not set.
func (m *Message) GetPayload() *jsonschema.SchemaUnion {
	if m == nil {
		return nil
	}
	return m.Payload
}

// GetCorrelationID returns the correlation ID, or nil if not set.
func (m *Message) GetCorrelationID() *ReferencedCorrelationID {
	if m == nil {
		return nil
	}
	return m.CorrelationID
}

// GetContentType returns the content type, or an empty string if not set.
func (m *Message) GetContentType() string {
	if m == nil || m.ContentType == nil {
		return ""
	}
	return *m.ContentType
}

// GetName returns the name, or an empty string if not set.
func (m *Message) GetName() string {
	if m == nil || m.Name == nil {
		return ""
	}
	return *m.Name
}

// GetTitle returns the title, or an empty string if not set.
func (m *Message) GetTitle() string {
	if m == nil || m.Title == nil {
		return ""
	}
	return *m.Title
}

// GetSummary returns the summary, or an empty string if not set.
func (m *Message) GetSummary() string {
	if m == nil || m.Summary == nil {
		return ""
	}
	return *m.Summary
}

// GetDescription returns the description, or an empty string if not set.
func (m *Message) GetDescription() string {
	if m == nil || m.Description == nil {
		return ""
	}
	return *m.Description
}

// GetTags returns the tags, or nil if not set.
func (m *Message) GetTags() []*Tag {
	if m == nil {
		return nil
	}
	return m.Tags
}

// GetExternalDocs returns the external documentation, or nil if not set.
func (m *Message) GetExternalDocs() *ExternalDocumentation {
	if m == nil {
		return nil
	}
	return m.ExternalDocs
}

// GetBindings returns the message bindings, or nil if not set.
func (m *Message) GetBindings() *ReferencedBindings {
	if m == nil {
		return nil
	}
	return m.Bindings
}

// GetExamples returns the examples, or nil if not set.
func (m *Message) GetExamples() []*MessageExample {
	if m == nil {
		return nil
	}
	return m.Examples
}

// GetTraits returns the message traits, or nil if not set.
func (m *Message) GetTraits() []*ReferencedMessageTrait {
	if m == nil {
		return nil
	}
	return m.Traits
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (m *Message) GetExtensions() *extensions.Extensions {
	if m == nil || m.Extensions == nil {
		return extensions.New()
	}
	return m.Extensions
}

// Validate will validate the Message Object against the AsyncAPI Specification.
func (m *Message) Validate(ctx context.Context, opts ...validation.Option) []error {
	if m == nil {
		return nil
	}

	c := m.GetCore()
	errs := validateMessageFields(ctx, messageFields{
		headers:       m.Headers,
		correlationID: m.CorrelationID,
		contentType:   m.ContentType,
		tags:          m.Tags,
		externalDocs:  m.ExternalDocs,
		bindings:      m.Bindings,
		examples:      m.Examples,
	}, c, c.Headers, c.ContentType, c.Tags, "message", opts...)

	if m.Payload != nil {
		errs = append(errs, m.Payload.Validate(ctx, opts...)...)
	}
	for _, trait := range m.Traits {
		errs = append(errs, trait.Validate(ctx, opts...)...)
	}

	errs = append(errs, unknownPropertyErrors(c, "message")...)

	m.Valid = noErrors(errs) && coreValid(c)

	return errs
}

type messageFields struct {
	headers       *jsonschema.SchemaUnion
	correlationID *ReferencedCorrelationID
	contentType   *string
	tags          []*Tag
	externalDocs  *ExternalDocumentation
	bindings      *ReferencedBindings
	examples      []*MessageExample
}

// validateMessageFields validates the fields shared by Message and Message Trait Objects.
func validateMessageFields(ctx context.Context, f messageFields, c validation.CoreModeler, headersNode, contentTypeNode validation.ValueNodeGetter, tagsNode validation.SliceNodeGetter, object string, opts ...validation.Option) []error {
	errs := []error{}

	if f.headers != nil {
		if types := f.headers.GetSchema().GetSchema().GetType(); len(types) > 0 && !slices.Contains(types, "object") {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationTypeMismatch, fmt.Errorf("%s.headers must be a schema of type `object`", object), c, headersNode))
		}
		errs = append(errs, f.headers.Validate(ctx, opts...)...)
	}
	if f.correlationID != nil {
		errs = append(errs, f.correlationID.Validate(ctx, opts...)...)
	}
	if f.contentType != nil {
		if err := validateMediaType(*f.contentType, object+".contentType"); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, err, c, contentTypeNode))
		}
	}
	errs = append(errs, validateTags(ctx, f.tags, c, tagsNode, object+".tags", opts...)...)
	if f.externalDocs != nil {
		errs = append(errs, f.externalDocs.Validate(ctx, opts...)...)
	}
	if f.bindings != nil {
		errs = append(errs, f.bindings.Validate(ctx, opts...)...)
	}
	for _, example := range f.examples {
		errs = append(errs, example.Validate(ctx, opts...)...)
	}

	return errs
}

// MessageTrait describes a trait that may be applied to a Message Object. It can contain any message property
// except payload and traits.
type MessageTrait struct {
	marshaller.Model[core.MessageTrait]

	// Headers is the schema of the application headers. It must be a schema of type object.
	Headers *jsonschema.SchemaUnion
	// CorrelationID is the definition of the correlation ID used for message tracing or matching.
	CorrelationID *ReferencedCorrelationID
	// ContentType is the content type to use when encoding/decoding the payload.
	ContentType *string
	// Name is a machine-friendly name for the message.
	Name *string
	// Title is a human-friendly title for the message.
	Title *string
	// Summary is a short summary of what the message is about.
	Summary *string
	// Description is a description of the message. May contain CommonMark syntax.
	Description *string
	// Tags are used for logical grouping and categorization of messages.
	Tags []*Tag
	// ExternalDocs is additional external documentation for the message.
	ExternalDocs *ExternalDocumentation
	// Bindings are protocol-specific definitions for the message.
	Bindings *ReferencedBindings
	// Examples are examples of the message.
	Examples []*MessageExample
	// Extensions provides a list of extensions to the Message Trait Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.MessageTrait] = (*MessageTrait)(nil)

// GetHeaders returns the headers schema, or nil if not set.
func (t *MessageTrait) GetHeaders() *jsonschema.SchemaUnion {
	if t == nil {
		return nil
	}
	return t.Headers
}

// GetCorrelationID returns the correlation ID, or nil if not set.
func (t *MessageTrait) GetCorrelationID() *ReferencedCorrelationID {
	if t == nil {
		return nil
	}
	return t.CorrelationID
}

// GetContentType returns the content type, or an empty string if not set.
func (t *MessageTrait) GetContentType() string {
	if t == nil || t.ContentType == nil {
		return ""
	}
	return *t.ContentType
}

// GetName returns the name, or an empty string if not set.
func (t *MessageTrait) GetName() string {
	if t == nil || t.Name == nil {
		return ""
	}
	return *t.Name
}

// GetTitle returns the title, or an empty string if not set.
func (t *MessageTrait) GetTitle() string {
	if t == nil || t.Title == nil {
		return ""
	}
	return *t.Title
}

// GetSummary returns the summary, or an empty string if not set.
func (t *MessageTrait) GetSummary() string {
	if t == nil || t.Summary == nil {
		return ""
	}
	return *t.Summary
}

// GetDescription returns the description, or an empty string if not set.
func (t *MessageTrait) GetDescription() string {
	if t == nil || t.Description == nil {
		return ""
	}
	return *t.Description
}

// GetTags returns the tags, or nil if not set.
func (t *MessageTrait) GetTags() []*Tag {
	if t == nil {
		return nil
	}
	return t.Tags
}

// GetExternalDocs returns the external documentation, or nil if not set.
func (t *MessageTrait) GetExternalDocs() *ExternalDocumentation {
	if t == nil {
		return nil
	}
	return t.ExternalDocs
}

// GetBindings returns the message bindings, or nil if not set.
func (t *MessageTrait) GetBindings() *ReferencedBindings {
	if t == nil {
		return nil
	}
	return t.Bindings
}

// GetExamples returns the examples, or nil if not set.
func (t *MessageTrait) GetExamples() []*MessageExample {
	if t == nil {
		return nil
	}
	return t.Examples
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (t *MessageTrait) GetExtensions() *extensions.Extensions {
	if t == nil || t.Extensions == nil {
		return extensions.New()
	}
	return t.Extensions
}

// Validate will validate the Message Trait Object against the AsyncAPI Specification.
func (t *MessageTrait) Validate(ctx context.Context, opts ...validation.Option) []error {
	if t == nil {
		return nil
	}

	c := t.GetCore()
	errs := validateMessageFields(ctx, messageFields{
		headers:       t.Headers,
		correlationID: t.CorrelationID,
		contentType:   t.ContentType,
		tags:          t.Tags,
		externalDocs:  t.ExternalDocs,
		bindings:      t.Bindings,
		examples:      t.Examples,
	}, c, c.Headers, c.ContentType, c.Tags, "messageTrait", opts...)
	errs = append(errs, unknownPropertyErrors(c, "messageTrait")...)

	t.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// MessageExample is an example of a message's headers and/or payload.
type MessageExample struct {
	marshaller.Model[core.MessageExample]

	// Headers is the example value of the message headers. It must be an object.
	Headers values.Value
	// Payload is the example value of the message payload.
	Payload values.Value
	// Name is a machine-friendly name for the example.
	Name *string
	// Summary is a short summary of the example.
	Summary *string
	// Extensions provides a list of extensions to the Message Example Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.MessageExample] = (*MessageExample)(nil)

// GetHeaders returns the example headers, or nil if not set.
func (e *MessageExample) GetHeaders() values.Value {
	if e == nil {
		return nil
	}
	return e.Headers
}

// GetPayload returns the example payload, or nil if not set.
func (e *MessageExample) GetPayload() values.Value {
	if e == nil {
		return nil
	}
	return e.Payload
}

// GetName returns the name, or an empty string if not set.
func (e *MessageExample) GetName() string {
	if e == nil || e.Name == nil {
		return ""
	}
	return *e.Name
}

// GetSummary returns the summary, or an empty string if not set.
func (e *MessageExample) GetSummary() string {
	if e == nil || e.Summary == nil {
		return ""
	}
	return *e.Summary
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (e *MessageExample) GetExtensions() *extensions.Extensions {
	if e == nil || e.Extensions == nil {
		return extensions.New()
	}
	return e.Extensions
}

// Validate will validate the Message Example Object against the AsyncAPI Specification.
// Example values are not validated against the message schemas.
func (e *MessageExample) Validate(_ context.Context, _ ...validation.Option) []error {
	if e == nil {
		return nil
	}

	c := e.GetCore()
	errs := []error{}

	if e.Headers == nil && e.Payload == nil {
		node := c.GetRootNode()
		errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("messageExample requires at least one of `headers` or `payload`"), node))
	}
	if h := yml.ResolveAlias(e.Headers); h != nil && h.Kind != yaml.MappingNode {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError("messageExample.headers", "messageExample.headers expected `object`, got %s", yml.NodeKindToString(h.Kind)), c, c.Headers))
	}
	errs = append(errs, unknownPropertyErrors(c, "messageExample")...)

	e.Valid = noErrors(errs) && coreValid(c)

	return errs
}

// CorrelationID specifies an identifier at design time that can be used for message tracing and correlation.
type CorrelationID struct {
	marshaller.Model[core.CorrelationID]

	// Description is a description of the identifier. May contain CommonMark syntax.
	Description *string
	// Location is a runtime expression specifying the location of the correlation ID, e.g. `$message.header#/correlationId`. Required.
	Location string
	// Extensions provides a list of extensions to the Correlation ID Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.CorrelationID] = (*CorrelationID)(nil)

// GetDescription returns the description, or an empty string if not set.
func (c *CorrelationID) GetDescription() string {
	if c == nil || c.Description == nil {
		return ""
	}
	return *c.Description
}

// GetLocation returns the location runtime expression, or an empty string if not set.
func (c *CorrelationID) GetLocation() string {
	if c == nil {
		return ""
	}
	return c.Location
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (c *CorrelationID) GetExtensions() *extensions.Extensions {
	if c == nil || c.Extensions == nil {
		return extensions.New()
	}
	return c.Extensions
}

// Validate will validate the Correlation ID Object against the AsyncAPI Specification.
func (c *CorrelationID) Validate(_ context.Context, _ ...validation.Option) []error {
	if c == nil {
		return nil
	}

	cc := c.GetCore()
	errs := validateLocation(c.Location, cc.Location.Present, cc, cc.Location, "correlationId.location")
	errs = append(errs, unknownPropertyErrors(cc, "correlationId")...)

	c.Valid = noErrors(errs) && coreValid(cc)

	return errs
}
