package asyncapi

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Components holds a set of reusable objects. Objects defined here have no effect unless they are referenced from
// outside the Components Object.
type Components struct {
	marshaller.Model[core.Components]

	// Schemas are reusable schemas, either native AsyncAPI schemas or Multi Format Schema Objects.
	Schemas *sequencedmap.Map[string, *jsonschema.SchemaUnion]
	// Servers are reusable Server Objects.
	Servers *sequencedmap.Map[string, *ReferencedServer]
	// Channels are reusable Channel Objects.
	Channels *sequencedmap.Map[string, *ReferencedChannel]
	// Operations are reusable Operation Objects.
	Operations *sequencedmap.Map[string, *ReferencedOperation]
	// Messages are reusable Message Objects.
	Messages *sequencedmap.Map[string, *ReferencedMessage]
	// SecuritySchemes are reusable Security Scheme Objects.
	SecuritySchemes *sequencedmap.Map[string, *ReferencedSecurityScheme]
	// ServerVariables are reusable Server Variable Objects.
	ServerVariables *sequencedmap.Map[string, *ReferencedServerVariable]
	// Parameters are reusable Parameter Objects.
	Parameters *sequencedmap.Map[string, *ReferencedParameter]
	// CorrelationIDs are reusable Correlation ID Objects.
	CorrelationIDs *sequencedmap.Map[string, *ReferencedCorrelationID]
	// Replies are reusable Operation Reply Objects.
	Replies *sequencedmap.Map[string, *ReferencedOperationReply]
	// ReplyAddresses are reusable Operation Reply Address Objects.
	ReplyAddresses *sequencedmap.Map[string, *ReferencedOperationReplyAddress]
	// ExternalDocs are reusable External Documentation Objects.
	ExternalDocs *sequencedmap.Map[string, *ExternalDocumentation]
	// Tags are reusable Tag Objects.
	Tags *sequencedmap.Map[string, *Tag]
	// OperationTraits are reusable Operation Trait Objects.
	OperationTraits *sequencedmap.Map[string, *ReferencedOperationTrait]
	// MessageTraits are reusable Message Trait Objects.
	MessageTraits *sequencedmap.Map[string, *ReferencedMessageTrait]
	// ServerBindings are reusable Server Bindings Objects.
	ServerBindings *sequencedmap.Map[string, *ReferencedBindings]
	// ChannelBindings are reusable Channel Bindings Objects.
	ChannelBindings *sequencedmap.Map[string, *ReferencedBindings]
	// OperationBindings are reusable Operation Bindings Objects.
	OperationBindings *sequencedmap.Map[string, *ReferencedBindings]
	// MessageBindings are reusable Message Bindings Objects.
	MessageBindings *sequencedmap.Map[string, *ReferencedBindings]
	// Extensions provides a list of extensions to the Components Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Components] = (*Components)(nil)

// GetSchemas returns the reusable schemas, or nil if not set.
func (c *Components) GetSchemas() *sequencedmap.Map[string, *jsonschema.SchemaUnion] {
	if c == nil {
		return nil
	}
	return c.Schemas
}

// GetServers returns the reusable servers, or nil if not set.
func (c *Components) GetServers() *sequencedmap.Map[string, *ReferencedServer] {
	if c == nil {
		return nil
	}
	return c.Servers
}

// GetChannels returns the reusable channels, or nil if not set.
func (c *Components) GetChannels() *sequencedmap.Map[string, *ReferencedChannel] {
	if c == nil {
		return nil
	}
	return c.Channels
}

// GetOperations returns the reusable operations, or nil if not set.
func (c *Components) GetOperations() *sequencedmap.Map[string, *ReferencedOperation] {
	if c == nil {
		return nil
	}
	return c.Operations
}

// GetMessages returns the reusable messages, or nil if not set.
func (c *Components) GetMessages() *sequencedmap.Map[string, *ReferencedMessage] {
	if c == nil {
		return nil
	}
	return c.Messages
}

// GetSecuritySchemes returns the reusable security schemes, or nil if not set.
func (c *Components) GetSecuritySchemes() *sequencedmap.Map[string, *ReferencedSecurityScheme] {
	if c == nil {
		return nil
	}
	return c.SecuritySchemes
}

// GetServerVariables returns the reusable server variables, or nil if not set.
func (c *Components) GetServerVariables() *sequencedmap.Map[string, *ReferencedServerVariable] {
	if c == nil {
		return nil
	}
	return c.ServerVariables
}

// GetParameters returns the reusable parameters, or nil if not set.
func (c *Components) GetParameters() *sequencedmap.Map[string, *ReferencedParameter] {
	if c == nil {
		return nil
	}
	return c.Parameters
}

// GetCorrelationIDs returns the reusable correlation IDs, or nil if not set.
func (c *Components) GetCorrelationIDs() *sequencedmap.Map[string, *ReferencedCorrelationID] {
	if c == nil {
		return nil
	}
	return c.CorrelationIDs
}

// GetReplies returns the reusable replies, or nil if not set.
func (c *Components) GetReplies() *sequencedmap.Map[string, *ReferencedOperationReply] {
	if c == nil {
		return nil
	}
	return c.Replies
}

// GetReplyAddresses returns the reusable reply addresses, or nil if not set.
func (c *Components) GetReplyAddresses() *sequencedmap.Map[string, *ReferencedOperationReplyAddress] {
	if c == nil {
		return nil
	}
	return c.ReplyAddresses
}

// GetExternalDocs returns the reusable external documentation, or nil if not set.
func (c *Components) GetExternalDocs() *sequencedmap.Map[string, *ExternalDocumentation] {
	if c == nil {
		return nil
	}
	return c.ExternalDocs
}

// GetTags returns the reusable tags, or nil if not set.
func (c *Components) GetTags() *sequencedmap.Map[string, *Tag] {
	if c == nil {
		return nil
	}
	return c.Tags
}

// GetOperationTraits returns the reusable operation traits, or nil if not set.
func (c *Components) GetOperationTraits() *sequencedmap.Map[string, *ReferencedOperationTrait] {
	if c == nil {
		return nil
	}
	return c.OperationTraits
}

// GetMessageTraits returns the reusable message traits, or nil if not set.
func (c *Components) GetMessageTraits() *sequencedmap.Map[string, *ReferencedMessageTrait] {
	if c == nil {
		return nil
	}
	return c.MessageTraits
}

// GetServerBindings returns the reusable server bindings, or nil if not set.
func (c *Components) GetServerBindings() *sequencedmap.Map[string, *ReferencedBindings] {
	if c == nil {
		return nil
	}
	return c.ServerBindings
}

// GetChannelBindings returns the reusable channel bindings, or nil if not set.
func (c *Components) GetChannelBindings() *sequencedmap.Map[string, *ReferencedBindings] {
	if c == nil {
		return nil
	}
	return c.ChannelBindings
}

// GetOperationBindings returns the reusable operation bindings, or nil if not set.
func (c *Components) GetOperationBindings() *sequencedmap.Map[string, *ReferencedBindings] {
	if c == nil {
		return nil
	}
	return c.OperationBindings
}

// GetMessageBindings returns the reusable message bindings, or nil if not set.
func (c *Components) GetMessageBindings() *sequencedmap.Map[string, *ReferencedBindings] {
	if c == nil {
		return nil
	}
	return c.MessageBindings
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (c *Components) GetExtensions() *extensions.Extensions {
	if c == nil || c.Extensions == nil {
		return extensions.New()
	}
	return c.Extensions
}

// Validate will validate the Components Object against the AsyncAPI Specification.
func (c *Components) Validate(ctx context.Context, opts ...validation.Option) []error {
	if c == nil {
		return nil
	}

	cc := c.GetCore()
	errs := []error{}

	errs = append(errs, validateComponentMap(ctx, c.Schemas, cc, cc.Schemas, "schemas", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.Servers, cc, cc.Servers, "servers", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.Channels, cc, cc.Channels, "channels", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.Operations, cc, cc.Operations, "operations", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.Messages, cc, cc.Messages, "messages", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.SecuritySchemes, cc, cc.SecuritySchemes, "securitySchemes", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.ServerVariables, cc, cc.ServerVariables, "serverVariables", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.Parameters, cc, cc.Parameters, "parameters", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.CorrelationIDs, cc, cc.CorrelationIDs, "correlationIds", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.Replies, cc, cc.Replies, "replies", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.ReplyAddresses, cc, cc.ReplyAddresses, "replyAddresses", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.ExternalDocs, cc, cc.ExternalDocs, "externalDocs", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.Tags, cc, cc.Tags, "tags", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.OperationTraits, cc, cc.OperationTraits, "operationTraits", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.MessageTraits, cc, cc.MessageTraits, "messageTraits", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.ServerBindings, cc, cc.ServerBindings, "serverBindings", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.ChannelBindings, cc, cc.ChannelBindings, "channelBindings", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.OperationBindings, cc, cc.OperationBindings, "operationBindings", opts...)...)
	errs = append(errs, validateComponentMap(ctx, c.MessageBindings, cc, cc.MessageBindings, "messageBindings", opts...)...)

	errs = append(errs, unknownPropertyErrors(cc, "components")...)

	c.Valid = noErrors(errs) && coreValid(cc)

	return errs
}

type validator interface {
	Validate(ctx context.Context, opts ...validation.Option) []error
}

// validateComponentMap checks a component map's keys match the component key pattern and validates its values.
func validateComponentMap[V validator](ctx context.Context, m *sequencedmap.Map[string, V], c validation.CoreModeler, node validation.MapKeyNodeGetter, field string, opts ...validation.Option) []error {
	if m == nil {
		return nil
	}

	errs := []error{}
	for key, value := range m.All() {
		if !componentKeyRegex.MatchString(key) {
			errs = append(errs, validation.NewMapKeyError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("components.%s key `%s` must match `%s`", field, key, componentKeyRegex.String()), c, node, key))
		}
		errs = append(errs, value.Validate(ctx, opts...)...)
	}
	return errs
}
