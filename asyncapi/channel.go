package asyncapi

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Channel describes a shared communication channel.
type Channel struct {
	marshaller.Model[core.Channel]

	// Address is the address of the channel, e.g. a topic or queue name. It may include Channel Address Expressions
	// such as `users.{userId}`. A nil address means the address is unknown or dynamic, which is also the case when the
	// document sets it to null explicitly.
	Address *string
	// Messages are the messages that can be sent on this channel, keyed by message ID.
	Messages *sequencedmap.Map[string, *ReferencedMessage]
	// Title is a human-friendly title for the channel.
	Title *string
	// Summary is a short summary of the channel.
	Summary *string
	// Description is a description of the channel. May contain CommonMark syntax.
	Description *string
	// Servers are references to the root servers this channel is available on. When empty, it is available on all servers.
	Servers []*ReferenceObject
	// Parameters are the parameters included in the channel address.
	Parameters *sequencedmap.Map[string, *ReferencedParameter]
	// Tags are used for logical grouping and categorization of channels.
	Tags []*Tag
	// ExternalDocs is additional external documentation for this channel.
	ExternalDocs *ExternalDocumentation
	// Bindings are protocol-specific definitions for the channel.
	Bindings *ReferencedBindings
	// Extensions provides a list of extensions to the Channel Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Channel] = (*Channel)(nil)

// GetAddress returns the address, or an empty string if the address is unknown (absent or null).
func (c *Channel) GetAddress() string {
	if c == nil || c.Address == nil {
		return ""
	}
	return *c.Address
}

// GetMessages returns the messages, or nil if not set.
func (c *Channel) GetMessages() *sequencedmap.Map[string, *ReferencedMessage] {
	if c == nil {
		return nil
	}
	return c.Messages
}

// GetTitle returns the title, or an empty string if not set.
func (c *Channel) GetTitle() string {
	if c == nil || c.Title == nil {
		return ""
	}
	return *c.Title
}

// GetSummary returns the summary, or an empty string if not set.
func (c *Channel) GetSummary() string {
	if c == nil || c.Summary == nil {
		return ""
	}
	return *c.Summary
}

// GetDescription returns the description, or an empty string if not set.
func (c *Channel) GetDescription() string {
	if c == nil || c.Description == nil {
		return ""
	}
	return *c.Description
}

// GetServers returns the server references, or nil if not set.
func (c *Channel) GetServers() []*ReferenceObject {
	if c == nil {
		return nil
	}
	return c.Servers
}

// GetParameters returns the parameters, or nil if not set.
func (c *Channel) GetParameters() *sequencedmap.Map[string, *ReferencedParameter] {
	if c == nil {
		return nil
	}
	return c.Parameters
}

// GetTags returns the tags, or nil if not set.
func (c *Channel) GetTags() []*Tag {
	if c == nil {
		return nil
	}
	return c.Tags
}

// GetExternalDocs returns the external documentation, or nil if not set.
func (c *Channel) GetExternalDocs() *ExternalDocumentation {
	if c == nil {
		return nil
	}
	return c.ExternalDocs
}

// GetBindings returns the channel bindings, or nil if not set.
func (c *Channel) GetBindings() *ReferencedBindings {
	if c == nil {
		return nil
	}
	return c.Bindings
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (c *Channel) GetExtensions() *extensions.Extensions {
	if c == nil || c.Extensions == nil {
		return extensions.New()
	}
	return c.Extensions
}

// Validate will validate the Channel Object against the AsyncAPI Specification.
func (c *Channel) Validate(ctx context.Context, opts ...validation.Option) []error {
	if c == nil {
		return nil
	}

	cc := c.GetCore()
	errs := []error{}

	// `#` isn't rejected as a fragment because it is a valid topic segment for protocols such as MQTT.
	if c.Address != nil && strings.Contains(*c.Address, "?") {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("channel.address `%s` must not contain query parameters", *c.Address), cc, cc.Address))
	}

	addressVariables := expressionVariables(c.GetAddress())
	errs = append(errs, validateExpressionVariables(c.GetAddress(), func(name string) bool { return c.Parameters != nil && c.Parameters.Has(name) }, "channel.address", "channel.parameters", cc, cc.Address)...)

	if c.Parameters != nil {
		for name, parameter := range c.Parameters.All() {
			if !nameKeyRegex.MatchString(name) {
				errs = append(errs, validation.NewMapKeyError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("channel.parameters key `%s` must match `%s`", name, nameKeyRegex.String()), cc, cc.Parameters, name))
			}
			if !slices.Contains(addressVariables, name) {
				errs = append(errs, validation.NewMapKeyError(validation.SeverityError, validation.RuleValidationInvalidTarget, fmt.Errorf("channel.parameters `%s` is not used in channel.address", name), cc, cc.Parameters, name))
			}
			errs = append(errs, parameter.Validate(ctx, opts...)...)
		}
	}

	if c.Messages != nil {
		for _, message := range c.Messages.All() {
			errs = append(errs, message.Validate(ctx, opts...)...)
		}
	}
	for _, server := range c.Servers {
		errs = append(errs, server.Validate(ctx, opts...)...)
	}
	errs = append(errs, validateTags(ctx, c.Tags, cc, cc.Tags, "channel.tags", opts...)...)
	if c.ExternalDocs != nil {
		errs = append(errs, c.ExternalDocs.Validate(ctx, opts...)...)
	}
	if c.Bindings != nil {
		errs = append(errs, c.Bindings.Validate(ctx, opts...)...)
	}

	errs = append(errs, unknownPropertyErrors(cc, "channel")...)

	c.Valid = noErrors(errs) && coreValid(cc)

	return errs
}

// Parameter describes a parameter included in a channel address.
type Parameter struct {
	marshaller.Model[core.Parameter]

	// Enum is the set of values the parameter may take.
	Enum []string
	// Default is the value to use when no other value is supplied.
	Default *string
	// Description is a description of the parameter. May contain CommonMark syntax.
	Description *string
	// Examples are examples of the parameter's value.
	Examples []string
	// Location is a runtime expression specifying the location of the parameter value, e.g. `$message.payload#/user/id`.
	Location *string
	// Extensions provides a list of extensions to the Parameter Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Parameter] = (*Parameter)(nil)

// GetEnum returns the allowed values, or nil if not set.
func (p *Parameter) GetEnum() []string {
	if p == nil {
		return nil
	}
	return p.Enum
}

// GetDefault returns the default value, or an empty string if not set.
func (p *Parameter) GetDefault() string {
	if p == nil || p.Default == nil {
		return ""
	}
	return *p.Default
}

// GetDescription returns the description, or an empty string if not set.
func (p *Parameter) GetDescription() string {
	if p == nil || p.Description == nil {
		return ""
	}
	return *p.Description
}

// GetExamples returns the examples, or nil if not set.
func (p *Parameter) GetExamples() []string {
	if p == nil {
		return nil
	}
	return p.Examples
}

// GetLocation returns the location runtime expression, or an empty string if not set.
func (p *Parameter) GetLocation() string {
	if p == nil || p.Location == nil {
		return ""
	}
	return *p.Location
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (p *Parameter) GetExtensions() *extensions.Extensions {
	if p == nil || p.Extensions == nil {
		return extensions.New()
	}
	return p.Extensions
}

// Validate will validate the Parameter Object against the AsyncAPI Specification.
func (p *Parameter) Validate(_ context.Context, _ ...validation.Option) []error {
	if p == nil {
		return nil
	}

	c := p.GetCore()
	errs := []error{}

	errs = append(errs, validateEnumDefault(p.Enum, p.Default, c, c.Default, "parameter")...)
	if p.Location != nil {
		if err := validateRuntimeExpression(*p.Location); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("parameter.location: %w", err), c, c.Location))
		}
	}
	errs = append(errs, unknownPropertyErrors(c, "parameter")...)

	p.Valid = noErrors(errs) && coreValid(c)

	return errs
}
