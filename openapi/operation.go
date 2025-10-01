package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Operation represents a single API operation on a path.
type Operation struct {
	marshaller.Model[core.Operation]

	// OperationID is a unique string used to identify the operation.
	OperationID *string
	// Summary is a short summary of what the operation does.
	Summary *string
	// Description is a verbose explanation of the operation behavior. May contain CommonMark syntax.
	Description *string
	// Tags is a list of tags for API documentation control.
	Tags []string
	// Servers is an alternative server array to service this operation.
	Servers []*Server
	// Security is a declaration of which security mechanisms can be used for this operation.
	Security []*SecurityRequirement

	// Parameters is a list of parameters that are applicable for this operation.
	Parameters []*ReferencedParameter
	// RequestBody is the request body applicable for this operation.
	RequestBody *ReferencedRequestBody
	// Responses is the list of possible responses as they are returned from executing this operation.
	Responses Responses
	// Callbacks is a map of possible out-of band callbacks related to the parent operation.
	Callbacks *sequencedmap.Map[string, *ReferencedCallback]

	// Deprecated declares this operation to be deprecated.
	Deprecated *bool
	// ExternalDocs is additional external documentation for this operation.
	ExternalDocs *oas3.ExternalDocumentation

	// Extensions provides a list of extensions to the Operation object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Operation] = (*Operation)(nil)

// GetOperationID returns the value of the OperationID field. Returns empty string if not set.
func (o *Operation) GetOperationID() string {
	if o == nil || o.OperationID == nil {
		return ""
	}
	return *o.OperationID
}

// GetSummary returns the value of the Summary field. Returns empty string if not set.
func (o *Operation) GetSummary() string {
	if o == nil || o.Summary == nil {
		return ""
	}
	return *o.Summary
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (o *Operation) GetDescription() string {
	if o == nil || o.Description == nil {
		return ""
	}
	return *o.Description
}

// GetDeprecated returns the value of the Deprecated field. False by default if not set.
func (o *Operation) GetDeprecated() bool {
	if o == nil || o.Deprecated == nil {
		return false
	}
	return *o.Deprecated
}

// GetTags returns the value of the Tags field. Returns nil if not set.
func (o *Operation) GetTags() []string {
	if o == nil {
		return nil
	}
	return o.Tags
}

// GetServers returns the value of the Servers field. Returns nil if not set.
func (o *Operation) GetServers() []*Server {
	if o == nil {
		return nil
	}
	return o.Servers
}

// GetSecurity returns the value of the Security field. Returns nil if not set.
func (o *Operation) GetSecurity() []*SecurityRequirement {
	if o == nil {
		return nil
	}
	return o.Security
}

// GetParameters returns the value of the Parameters field. Returns nil if not set.
func (o *Operation) GetParameters() []*ReferencedParameter {
	if o == nil {
		return nil
	}
	return o.Parameters
}

// GetRequestBody returns the value of the RequestBody field. Returns nil if not set.
func (o *Operation) GetRequestBody() *ReferencedRequestBody {
	if o == nil {
		return nil
	}
	return o.RequestBody
}

// GetResponses returns the value of the Responses field. Returns nil if not set.
func (o *Operation) GetResponses() *Responses {
	return &o.Responses
}

// GetCallbacks returns the value of the Callbacks field. Returns nil if not set.
func (o *Operation) GetCallbacks() *sequencedmap.Map[string, *ReferencedCallback] {
	if o == nil {
		return nil
	}
	return o.Callbacks
}

// GetExternalDocs returns the value of the ExternalDocs field. Returns nil if not set.
func (o *Operation) GetExternalDocs() *oas3.ExternalDocumentation {
	if o == nil {
		return nil
	}
	return o.ExternalDocs
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (o *Operation) GetExtensions() *extensions.Extensions {
	if o == nil || o.Extensions == nil {
		return extensions.New()
	}
	return o.Extensions
}

// IsDeprecated is an alias for GetDeprecated for backward compatibility.
// Deprecated: Use GetDeprecated instead for consistency with other models.
func (o *Operation) IsDeprecated() bool {
	return o.GetDeprecated()
}

// Validate validates the Operation object against the OpenAPI Specification.
func (o *Operation) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := o.GetCore()
	errs := []error{}

	for _, server := range o.Servers {
		errs = append(errs, server.Validate(ctx, opts...)...)
	}

	for _, securityRequirement := range o.Security {
		errs = append(errs, securityRequirement.Validate(ctx, opts...)...)
	}

	for _, parameter := range o.Parameters {
		errs = append(errs, parameter.Validate(ctx, opts...)...)
	}

	if o.RequestBody != nil {
		errs = append(errs, o.RequestBody.Validate(ctx, opts...)...)
	}

	if core.Responses.Present {
		errs = append(errs, o.Responses.Validate(ctx, opts...)...)
	}

	for _, callback := range o.Callbacks.All() {
		errs = append(errs, callback.Validate(ctx, opts...)...)
	}

	if o.ExternalDocs != nil {
		errs = append(errs, o.ExternalDocs.Validate(ctx, opts...)...)
	}

	o.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
