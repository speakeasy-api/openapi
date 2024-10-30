package arazzo

import (
	"context"
	"fmt"
	"regexp"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Components holds reusable components that can be referenced in an Arazzo document.
type Components struct {
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

	// Valid indicates whether this model passed validation.
	Valid bool

	core core.Components
}

var _ model[core.Components] = (*Components)(nil)

// GetCore will return the low level representation of the components object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (c *Components) GetCore() *core.Components {
	return &c.core
}

var componentNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\.\-_]+$`)

type componentKey struct {
	name string
}

// Validate validates the Components object.
func (c *Components) Validate(ctx context.Context, opts ...validation.Option) []error {
	errs := []error{}

	for key, input := range c.Inputs.All() {
		if !componentNameRegex.MatchString(key) {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("input key must be a valid key [%s]: %s", componentNameRegex.String(), key),
				Line:    c.core.Inputs.GetMapKeyNodeOrRoot(key, c.core.RootNode).Line,
				Column:  c.core.Inputs.GetMapKeyNodeOrRoot(key, c.core.RootNode).Column,
			})
		}

		if input.IsLeft() {
			jsOpts := append(opts, validation.WithContextObject(&componentKey{name: key}))

			errs = append(errs, input.Left.Validate(ctx, jsOpts...)...)
		}
	}

	for key, parameter := range c.Parameters.All() {
		if !componentNameRegex.MatchString(key) {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("parameter key must be a valid key [%s]: %s", componentNameRegex.String(), key),
				Line:    c.core.Parameters.GetMapKeyNodeOrRoot(key, c.core.RootNode).Line,
				Column:  c.core.Parameters.GetMapKeyNodeOrRoot(key, c.core.RootNode).Column,
			})
		}

		paramOps := append(opts, validation.WithContextObject(&componentKey{name: key}))

		errs = append(errs, parameter.Validate(ctx, paramOps...)...)
	}

	for key, successAction := range c.SuccessActions.All() {
		if !componentNameRegex.MatchString(key) {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("successAction key must be a valid key [%s]: %s", componentNameRegex.String(), key),
				Line:    c.core.SuccessActions.GetMapKeyNodeOrRoot(key, c.core.RootNode).Line,
				Column:  c.core.SuccessActions.GetMapKeyNodeOrRoot(key, c.core.RootNode).Column,
			})
		}

		successActionOps := append(opts, validation.WithContextObject(&componentKey{name: key}))

		errs = append(errs, successAction.Validate(ctx, successActionOps...)...)
	}

	for key, failureAction := range c.FailureActions.All() {
		if !componentNameRegex.MatchString(key) {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("failureAction key must be a valid key [%s]: %s", componentNameRegex.String(), key),
				Line:    c.core.FailureActions.GetMapKeyNodeOrRoot(key, c.core.RootNode).Line,
				Column:  c.core.FailureActions.GetMapKeyNodeOrRoot(key, c.core.RootNode).Column,
			})
		}

		failureActionOps := append(opts, validation.WithContextObject(&componentKey{name: key}))

		errs = append(errs, failureAction.Validate(ctx, failureActionOps...)...)
	}

	if len(errs) == 0 {
		c.Valid = true
	}

	return errs
}
