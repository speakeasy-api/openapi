package asyncapi

import (
	"context"
	"fmt"
	"iter"
	"slices"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

// knownBindingProtocols are the protocols with bindings defined by AsyncAPI 3.0.
var knownBindingProtocols = []string{
	"http", "ws", "kafka", "anypointmq", "amqp", "amqp1", "mqtt", "mqtt5", "nats", "jms", "sns", "solace", "sqs",
	"stomp", "redis", "mercure", "ibmmq", "googlepubsub", "pulsar",
}

// bindingProtocolsSince31 are the protocols with bindings added in AsyncAPI 3.1.
var bindingProtocolsSince31 = []string{"ros2"}

// Bindings holds protocol-specific information for a server, channel, operation or message, keyed by protocol name.
//
// Each protocol binding is kept as a raw value, so bindings for any protocol (and any binding version) are preserved
// unchanged. Only the shape of each binding is validated; protocol-specific rules are not checked.
type Bindings struct {
	marshaller.Model[core.Bindings]
	*sequencedmap.Map[string, values.Value]

	// Extensions provides a list of extensions to the Bindings Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Bindings] = (*Bindings)(nil)

// NewBindings creates a new Bindings object with the embedded map initialized.
func NewBindings() *Bindings {
	return &Bindings{
		Map: sequencedmap.New[string, values.Value](),
	}
}

// Len returns the number of protocol bindings. nil safe.
func (b *Bindings) Len() int {
	if b == nil || b.Map == nil {
		return 0
	}
	return b.Map.Len()
}

// All returns an iterator over the protocol bindings. nil safe.
func (b *Bindings) All() iter.Seq2[string, values.Value] {
	if b == nil || b.Map == nil {
		return func(yield func(string, values.Value) bool) {}
	}
	return b.Map.All()
}

// GetBindingVersion returns the `bindingVersion` of the binding for the provided protocol, or an empty string if
// the binding or its version is not set.
func (b *Bindings) GetBindingVersion(protocol string) string {
	if b == nil || b.Map == nil {
		return ""
	}
	binding, ok := b.Get(protocol)
	if !ok {
		return ""
	}
	_, versionNode, found := yml.GetMapElementNodes(context.Background(), yml.ResolveAlias(binding), "bindingVersion")
	versionNode = yml.ResolveAlias(versionNode)
	if !found || versionNode == nil {
		return ""
	}
	return versionNode.Value
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (b *Bindings) GetExtensions() *extensions.Extensions {
	if b == nil || b.Extensions == nil {
		return extensions.New()
	}
	return b.Extensions
}

// Validate validates the shape of each protocol binding. Bindings for unknown protocols produce a warning and
// bindings for known protocols a hint, as their protocol-specific contents are preserved but not validated.
func (b *Bindings) Validate(_ context.Context, opts ...validation.Option) []error {
	if b == nil {
		return nil
	}

	c := b.GetCore()
	errs := []error{}

	version := documentVersion(opts)
	supports31 := version == nil || version.Major > 3 || (version.Major == 3 && version.Minor >= 1)

	for protocol, binding := range b.All() {
		keyNode := c.GetMapKeyNodeOrRoot(protocol, c.GetRootNode())

		switch {
		case slices.Contains(knownBindingProtocols, protocol), supports31 && slices.Contains(bindingProtocolsSince31, protocol):
			errs = append(errs, validation.NewValidationError(validation.SeverityHint, validation.RuleValidationUnsupportedFormat, fmt.Errorf("bindings.%s is preserved but its protocol-specific fields are not validated", protocol), keyNode))
		case slices.Contains(bindingProtocolsSince31, protocol):
			errs = append(errs, validation.NewValidationError(validation.SeverityWarning, validation.RuleValidationUnknownProperties, fmt.Errorf("bindings.%s requires AsyncAPI 3.1.0 or later, document is %s", protocol, version), keyNode))
		default:
			errs = append(errs, validation.NewValidationError(validation.SeverityWarning, validation.RuleValidationUnknownProperties, fmt.Errorf("bindings.%s is not a known protocol, it is preserved but not validated", protocol), keyNode))
		}

		node := yml.ResolveAlias(binding)
		if node == nil {
			continue
		}
		if node.Kind != yaml.MappingNode {
			errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError("bindings."+protocol, "bindings.%s expected `object`, got %s", protocol, yml.NodeKindToString(node.Kind)), node))
			continue
		}
		if _, versionNode, found := yml.GetMapElementNodes(context.Background(), node, "bindingVersion"); found {
			if v := yml.ResolveAlias(versionNode); v == nil || v.Kind != yaml.ScalarNode || v.Tag != "!!str" {
				errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError("bindings."+protocol+".bindingVersion", "bindings.%s.bindingVersion expected `string`", protocol), versionNode))
			}
		}
	}

	b.Valid = noErrors(errs) && coreValid(c)

	return errs
}
