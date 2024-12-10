package arazzo

import (
	"context"
	"fmt"
	"mime"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

// RequestBody represents the request body to pass to an operation represented by a step
type RequestBody struct {
	// ContentType is the content type of the request body
	ContentType *string
	// Payload is a static value, an expression or a value containing inline expressions that will be used to populate the request body
	Payload ValueOrExpression
	// Replacements is a list of expressions that will be used to populate the request body in addition to any in the Payload field
	Replacements []*PayloadReplacement
	// Extensions is a list of extensions to apply to the request body object
	Extensions *extensions.Extensions

	// Valid indicates whether this model passed validation.
	Valid bool

	core core.RequestBody
}

var _ model[core.RequestBody] = (*RequestBody)(nil)

// GetCore will return the low level representation of the request body object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (r *RequestBody) GetCore() *core.RequestBody {
	return &r.core
}

// Validate will validate the request body object against the Arazzo specification.
func (r *RequestBody) Validate(ctx context.Context, opts ...validation.Option) []error {
	errs := []error{}

	if r.ContentType != nil {
		_, _, err := mime.ParseMediaType(*r.ContentType)
		if err != nil {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("contentType must be valid: %s", err),
				Line:    r.core.ContentType.GetValueNodeOrRoot(r.core.RootNode).Line,
				Column:  r.core.ContentType.GetValueNodeOrRoot(r.core.RootNode).Column,
			})
		}
	}

	if r.Payload != nil {
		payloadData, err := yaml.Marshal(r.Payload)
		if err != nil {
			errs = append(errs, &validation.Error{
				Message: err.Error(),
				Line:    r.core.Payload.GetValueNodeOrRoot(r.core.RootNode).Line,
				Column:  r.core.Payload.GetValueNodeOrRoot(r.core.RootNode).Column,
			})
		}

		expressions := expression.ExtractExpressions(string(payloadData))
		for _, expression := range expressions {
			if err := expression.Validate(true); err != nil {
				errs = append(errs, &validation.Error{
					Message: err.Error(),
					Line:    r.core.Payload.GetValueNodeOrRoot(r.core.RootNode).Line,
					Column:  r.core.Payload.GetValueNodeOrRoot(r.core.RootNode).Column,
				})
			}
		}
	}

	for _, replacement := range r.Replacements {
		errs = append(errs, replacement.Validate(ctx, opts...)...)
	}

	if len(errs) == 0 {
		r.Valid = true
	}

	return errs
}
