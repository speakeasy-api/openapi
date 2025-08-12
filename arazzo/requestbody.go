package arazzo

import (
	"context"
	"mime"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// RequestBody represents the request body to pass to an operation represented by a step
type RequestBody struct {
	marshaller.Model[core.RequestBody]

	// ContentType is the content type of the request body
	ContentType *string
	// Payload is a static value, an expression or a value containing inline expressions that will be used to populate the request body
	Payload expression.ValueOrExpression
	// Replacements is a list of expressions that will be used to populate the request body in addition to any in the Payload field
	Replacements []*PayloadReplacement
	// Extensions is a list of extensions to apply to the request body object
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.RequestBody] = (*RequestBody)(nil)

// Validate will validate the request body object against the Arazzo specification.
func (r *RequestBody) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := r.GetCore()
	errs := []error{}

	if r.ContentType != nil {
		_, _, err := mime.ParseMediaType(*r.ContentType)
		if err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("requestBody field contentType is not valid: %s", err.Error()), core, core.ContentType))
		}
	}

	// Validate payload only if it is a top-level Arazzo runtime expression
	// Skip validation for values containing embedded expressions or arbitrary data
	if r.Payload != nil {
		_, exp, err := expression.GetValueOrExpressionValue(r.Payload)
		if err == nil && exp != nil {
			// Only validate if the entire payload IS an expression (not just contains expressions)
			if err := exp.Validate(); err != nil {
				errs = append(errs, validation.NewValueError(validation.NewValueValidationError("requestBody field payload expression is not valid: %s", err.Error()), core, core.Payload))
			}
		}
		// If exp is nil, the payload is a value (not an expression) - no validation needed
	}

	for _, replacement := range r.Replacements {
		errs = append(errs, replacement.Validate(ctx, opts...)...)
	}

	r.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
