package arazzo

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// PayloadReplacement represents a replacement of a value within a payload such as a request body.
type PayloadReplacement struct {
	marshaller.Model[core.PayloadReplacement]

	// Target is a JSON Pointer, XPath expression or JSONPath (Arazzo 1.1+) to the value to be replaced.
	// How the target is interpreted is determined by TargetSelectorType, or the payload content type if not set.
	Target jsonpointer.JSONPointer
	// TargetSelectorType is the selector type of the target (`jsonpath`, `xpath` or `jsonpointer`) or an Expression Type Object. Arazzo 1.1+ only.
	TargetSelectorType *criterion.CriterionTypeUnion
	// Value represents either the static value of the replacement, an expression that will be evaluated to produce the value
	// or a Selector Object (Arazzo 1.1+).
	Value expression.ValueOrExpression
	// Extensions provides a list of extensions to the PayloadReplacement object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.PayloadReplacement] = (*PayloadReplacement)(nil)

// Validate will validate the payload replacement object against the Arazzo specification.
// If a RequestBody object is provided via validation options with validation.WithContextObject() then its
// content type is used to determine how the target is interpreted when targetSelectorType is not set.
func (p *PayloadReplacement) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := p.GetCore()
	errs := []error{}

	o := validation.NewOptions(opts...)
	is11 := specversion.IsAtLeast11(o)
	exprOpts := specversion.ExpressionOptions(o)

	if core.Target.Present && p.Target == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("payloadReplacement.target is required"), core, core.Target))
	}

	targetType := criterion.CriterionTypeJsonPointer
	if core.TargetSelectorType.Present && p.TargetSelectorType != nil {
		if !is11 {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("`payloadReplacement.targetSelectorType`"), core, core.TargetSelectorType))
		}
		errs = append(errs, p.TargetSelectorType.ValidateAsSelectorType(core.TargetSelectorType.GetValueNodeOrRoot(core.RootNode), opts...)...)
		targetType = p.TargetSelectorType.GetType()
	} else if rb := validation.GetContextObject[RequestBody](o); rb != nil && isXMLContentType(rb.ContentType) {
		targetType = criterion.CriterionTypeXPath
	}

	if p.Target != "" || !core.Target.Present {
		if err := validateSelectorSyntax(targetType, string(p.Target)); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("payloadReplacement.target is invalid: %w", err), core, core.Target))
		}
	}

	switch {
	case core.Value.Present && p.Value == nil:
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("payloadReplacement.value is required"), core, core.Value))
	case is11 && isSelectorNode(p.Value):
		errs = append(errs, validateSelectorNode(ctx, p.Value, opts...)...)
	case p.Value != nil:
		_, expression, err := expression.GetValueOrExpressionValue(p.Value, exprOpts...)
		if err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("payloadReplacement.value is invalid: %w", err), core, core.Value))
		}
		if expression != nil {
			if err := expression.Validate(exprOpts...); err != nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("payloadReplacement.value is invalid: %w", err), core, core.Value))
			}
		}
	}

	p.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// isXMLContentType returns true if the content type is `application/xml`, `text/xml` or an XML based media type (`+xml`).
func isXMLContentType(contentType *string) bool {
	if contentType == nil {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(*contentType)
	if err != nil {
		return false
	}

	return mediaType == "application/xml" || mediaType == "text/xml" || strings.HasSuffix(mediaType, "+xml")
}
