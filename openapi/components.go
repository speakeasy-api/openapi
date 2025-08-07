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

// Components is a container for the reusable objects available to the API.
type Components struct {
	marshaller.Model[core.Components]

	// Schemas is a map of reusable Schema Objects.
	Schemas *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]]
	// Responses is a map of reusable Response Objects.
	Responses *sequencedmap.Map[string, *ReferencedResponse]
	// Parameters is a map of reusable Parameter Objects.
	Parameters *sequencedmap.Map[string, *ReferencedParameter]
	// Examples is a map of reusable Example Objects.
	Examples *sequencedmap.Map[string, *ReferencedExample]
	// RequestBodies is a map of reusable Request Body Objects.
	RequestBodies *sequencedmap.Map[string, *ReferencedRequestBody]
	// Headers is a map of reusable Header Objects.
	Headers *sequencedmap.Map[string, *ReferencedHeader]
	// SecuritySchemes is a map of reusable Security Scheme Objects.
	SecuritySchemes *sequencedmap.Map[string, *ReferencedSecurityScheme]
	// Links is a map of reusable Link Objects.
	Links *sequencedmap.Map[string, *ReferencedLink]
	// Callbacks is a map of reusable Callback Objects.
	Callbacks *sequencedmap.Map[string, *ReferencedCallback]
	// PathItems is a map of reusable Path Item Objects.
	PathItems *sequencedmap.Map[string, *ReferencedPathItem]

	// Extensions provides a list of extensions to the Components object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Components] = (*Components)(nil)

// GetSchemas returns the value of the Schemas field. Returns nil if not set.
func (c *Components) GetSchemas() *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]] {
	if c == nil {
		return nil
	}
	return c.Schemas
}

// GetResponses returns the value of the Responses field. Returns nil if not set.
func (c *Components) GetResponses() *sequencedmap.Map[string, *ReferencedResponse] {
	if c == nil {
		return nil
	}
	return c.Responses
}

// GetParameters returns the value of the Parameters field. Returns nil if not set.
func (c *Components) GetParameters() *sequencedmap.Map[string, *ReferencedParameter] {
	if c == nil {
		return nil
	}
	return c.Parameters
}

// GetExamples returns the value of the Examples field. Returns nil if not set.
func (c *Components) GetExamples() *sequencedmap.Map[string, *ReferencedExample] {
	if c == nil {
		return nil
	}
	return c.Examples
}

// GetRequestBodies returns the value of the RequestBodies field. Returns nil if not set.
func (c *Components) GetRequestBodies() *sequencedmap.Map[string, *ReferencedRequestBody] {
	if c == nil {
		return nil
	}
	return c.RequestBodies
}

// GetHeaders returns the value of the Headers field. Returns nil if not set.
func (c *Components) GetHeaders() *sequencedmap.Map[string, *ReferencedHeader] {
	if c == nil {
		return nil
	}
	return c.Headers
}

// GetSecuritySchemes returns the value of the SecuritySchemes field. Returns nil if not set.
func (c *Components) GetSecuritySchemes() *sequencedmap.Map[string, *ReferencedSecurityScheme] {
	if c == nil {
		return nil
	}
	return c.SecuritySchemes
}

// GetLink returns the value of the Links field. Returns nil if not set.
func (c *Components) GetLinks() *sequencedmap.Map[string, *ReferencedLink] {
	if c == nil {
		return nil
	}
	return c.Links
}

// GetCallbacks returns the value of the Callbacks field. Returns nil if not set.
func (c *Components) GetCallbacks() *sequencedmap.Map[string, *ReferencedCallback] {
	if c == nil {
		return nil
	}
	return c.Callbacks
}

// GetPathItems returns the value of the PathItems field. Returns nil if not set.
func (c *Components) GetPathItems() *sequencedmap.Map[string, *ReferencedPathItem] {
	if c == nil {
		return nil
	}
	return c.PathItems
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (c *Components) GetExtensions() *extensions.Extensions {
	if c == nil || c.Extensions == nil {
		return extensions.New()
	}
	return c.Extensions
}

// Validate will validate the Components object against the OpenAPI Specification.
func (c *Components) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	if c.Schemas != nil {
		for _, schema := range c.Schemas.All() {
			if schema.IsLeft() {
				errs = append(errs, schema.Left.Validate(ctx, opts...)...)
			}
		}
	}

	if c.Responses != nil {
		for _, response := range c.Responses.All() {
			errs = append(errs, response.Validate(ctx, opts...)...)
		}
	}

	if c.Parameters != nil {
		for _, parameter := range c.Parameters.All() {
			errs = append(errs, parameter.Validate(ctx, opts...)...)
		}
	}

	if c.Examples != nil {
		for _, example := range c.Examples.All() {
			errs = append(errs, example.Validate(ctx, opts...)...)
		}
	}

	if c.RequestBodies != nil {
		for _, requestBody := range c.RequestBodies.All() {
			errs = append(errs, requestBody.Validate(ctx, opts...)...)
		}
	}

	if c.Headers != nil {
		for _, header := range c.Headers.All() {
			errs = append(errs, header.Validate(ctx, opts...)...)
		}
	}

	if c.SecuritySchemes != nil {
		for _, securityScheme := range c.SecuritySchemes.All() {
			errs = append(errs, securityScheme.Validate(ctx, opts...)...)
		}
	}

	if c.Links != nil {
		for _, link := range c.Links.All() {
			errs = append(errs, link.Validate(ctx, opts...)...)
		}
	}

	if c.Callbacks != nil {
		for _, callback := range c.Callbacks.All() {
			errs = append(errs, callback.Validate(ctx, opts...)...)
		}
	}

	if c.PathItems != nil {
		for _, pathItem := range c.PathItems.All() {
			errs = append(errs, pathItem.Validate(ctx, opts...)...)
		}
	}

	c.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
