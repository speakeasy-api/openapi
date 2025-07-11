package arazzo

import (
	"context"
	"regexp"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Components holds reusable components that can be referenced in an Arazzo document.
type Components struct {
	marshaller.Model[core.Components]

	// Inputs provides a list of reusable JSON Schemas that can be referenced from inputs and other JSON Schemas.
	Inputs *sequencedmap.Map[string, oas31.JSONSchema]
	// Parameters provides a list of reusable parameters that can be referenced from workflows and steps.
	Parameters *sequencedmap.Map[string, *Parameter]
	// SuccessActions provides a list of reusable success actions that can be referenced from workflows and steps.
	SuccessActions *sequencedmap.Map[string, *SuccessAction]
	// FailureActions provides a list of reusable failure actions that can be referenced from workflows and steps.
	FailureActions *sequencedmap.Map[string, *FailureAction]
	// Extensions provides a list of extensions to the Components object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Components] = (*Components)(nil)

var componentNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\.\-_]+$`)

type componentKey struct {
	name string
}

// Validate validates the Components object.
func (c *Components) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	for key, input := range c.Inputs.All() {
		if !componentNameRegex.MatchString(key) {
			errs = append(errs, validation.NewMapKeyError(validation.NewValueValidationError("input key must be a valid key [%s]: %s", componentNameRegex.String(), key), core, core.Inputs, key))
		}

		if input.IsLeft() {
			jsOpts := append(opts, validation.WithContextObject(&componentKey{name: key}))

			errs = append(errs, input.Left.Validate(ctx, jsOpts...)...)
		}
	}

	for key, parameter := range c.Parameters.All() {
		if !componentNameRegex.MatchString(key) {
			errs = append(errs, validation.NewMapKeyError(validation.NewValueValidationError("parameter key must be a valid key [%s]: %s", componentNameRegex.String(), key), core, core.Parameters, key))
		}

		paramOps := append(opts, validation.WithContextObject(&componentKey{name: key}))

		errs = append(errs, parameter.Validate(ctx, paramOps...)...)
	}

	for key, successAction := range c.SuccessActions.All() {
		if !componentNameRegex.MatchString(key) {
			errs = append(errs, validation.NewMapKeyError(validation.NewValueValidationError("successAction key must be a valid key [%s]: %s", componentNameRegex.String(), key), core, core.SuccessActions, key))
		}

		successActionOps := append(opts, validation.WithContextObject(&componentKey{name: key}))

		errs = append(errs, successAction.Validate(ctx, successActionOps...)...)
	}

	for key, failureAction := range c.FailureActions.All() {
		if !componentNameRegex.MatchString(key) {
			errs = append(errs, validation.NewMapKeyError(validation.NewValueValidationError("failureAction key must be a valid key [%s]: %s", componentNameRegex.String(), key), core, core.FailureActions, key))
		}

		failureActionOps := append(opts, validation.WithContextObject(&componentKey{name: key}))

		errs = append(errs, failureAction.Validate(ctx, failureActionOps...)...)
	}

	c.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
