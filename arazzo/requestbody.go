package arazzo

import (
	"context"
	"fmt"
	"mime"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

// RequestBody represents the request body to pass to an operation represented by a step
type RequestBody struct {
	marshaller.Model[core.RequestBody]

	// ContentType is the content type of the request body
	ContentType *string
	// Payload is a static value, an expression or a value containing inline expressions that will be used to populate the request body
	Payload ValueOrExpression
	// Replacements is a list of expressions that will be used to populate the request body in addition to any in the Payload field
	Replacements []*PayloadReplacement
	// Extensions is a list of extensions to apply to the request body object
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.RequestBody] = (*RequestBody)(nil)

// Validate will validate the request body object against the Arazzo specification.
func (r *RequestBody) Validate(ctx context.Context, opts ...validation.Option) []error {
	errs := []error{}
	core := r.GetCore()

	if r.ContentType != nil {
		_, _, err := mime.ParseMediaType(*r.ContentType)
		if err != nil {
			errs = append(errs, validation.NewValueError(fmt.Sprintf("contentType must be valid: %s", err), core, core.ContentType))
		}
	}

	if r.Payload != nil {
		payloadData, err := yaml.Marshal(r.Payload)
		if err != nil {
			errs = append(errs, validation.NewValueError(err.Error(), core, core.Payload))
		}

		expressions := expression.ExtractExpressions(string(payloadData))
		for _, expression := range expressions {
			if err := expression.Validate(true); err != nil {
				errs = append(errs, validation.NewValueError(err.Error(), core, core.Payload))
			}
		}
	}

	for _, replacement := range r.Replacements {
		errs = append(errs, replacement.Validate(ctx, opts...)...)
	}

	r.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
