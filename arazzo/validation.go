package arazzo

import (
	"context"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

func validateJSONSchema(ctx context.Context, js *oas3.JSONSchema[oas3.Referenceable], valueNode *yaml.Node, opts ...validation.Option) []error {
	errs := []error{}

	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)

	if a == nil {
		return []error{
			validation.NewValidationError(validation.NewValueValidationError("An Arazzo object must be passed via validation options to validate a JSONSchema"), valueNode),
		}
	}

	if js.IsRight() {
		errs = append(errs, validation.NewValidationError(validation.NewValueValidationError("inputs schema must represent an object with specific properties for inputs"), valueNode))
	} else {
		errs = append(errs, js.Left.Validate(ctx, opts...)...)

		if js.Left.Ref != nil {
			// TODO we will need to dereference and validate
		} else if js.Left.AllOf != nil {
			// TODO we will want to try and deduce if this boils down to a compatible object but just assume it does for now
		} else if js.Left.Type != nil {
			if js.Left.Type != nil && js.Left.Type.IsLeft() {
				types := js.Left.Type.LeftValue()
				if len(types) != 1 || types[0] != "object" {
					errs = append(errs, validation.NewValidationError(validation.NewValueValidationError("inputs schema must represent an object with specific properties for inputs"), valueNode))
				}
			}
			if js.Left.Type.IsRight() {
				if js.Left.Type.RightValue() != "object" {
					errs = append(errs, validation.NewValidationError(validation.NewValueValidationError("inputs schema must represent an object with specific properties for inputs"), valueNode))
				}
			}
		} else {
			if js.Left.Properties.Len() == 0 {
				errs = append(errs, validation.NewValidationError(validation.NewValueValidationError("inputs schema must represent an object with specific properties for inputs"), valueNode))
			}
		}
	}

	return errs
}
