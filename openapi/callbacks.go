package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Callback represents a set of callbacks related to the parent operation.
// The keys that represent the path items are a runtime expression that can be evaluated in the context of a request/response from the parent operation.
// Callback embeds sequencedmap.Map[string, *ReferencedPathItem] so all map operations are supported.
type Callback struct {
	marshaller.Model[core.Callback]
	*sequencedmap.Map[expression.Expression, *ReferencedPathItem]

	// Extensions provides a list of extensions to the Callback object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Callback] = (*Callback)(nil)

// NewCallback creates a new Callback object with the embedded map initialized.
func NewCallback() *Callback {
	return &Callback{
		Map: sequencedmap.New[expression.Expression, *ReferencedPathItem](),
	}
}

// Len returns the number of elements in the callback map. nil safe.
func (c *Callback) Len() int {
	if c == nil || c.Map == nil {
		return 0
	}
	return c.Map.Len()
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (c *Callback) GetExtensions() *extensions.Extensions {
	if c == nil || c.Extensions == nil {
		return extensions.New()
	}
	return c.Extensions
}

func (c *Callback) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	for exp, pathItem := range c.All() {
		if err := exp.Validate(); err != nil {
			node := core.RootNode

			// Find yaml node from core.RootNode
			for _, n := range core.RootNode.Content {
				if n.Value == string(exp) {
					node = n
					break
				}
			}

			errs = append(errs, validation.NewValidationError(validation.NewValueValidationError("callback expression is invalid: %s", err.Error()), node))
		}

		errs = append(errs, pathItem.Validate(ctx, opts...)...)
	}

	c.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
