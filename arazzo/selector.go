package arazzo

import (
	"context"
	"errors"
	"fmt"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

// Selector represents a Selector Object, which selects a value from structured data using JSONPath, XPath or
// JSON Pointer. Selector Objects can be used for workflow and step outputs, parameter values and payload
// replacement values. Arazzo 1.1+ only.
type Selector struct {
	marshaller.Model[core.Selector]

	// Context is a runtime expression that evaluates to the structured data the selector is applied to.
	Context expression.Expression
	// Selector is the JSONPath, XPath or JSON Pointer used to select a value from the context.
	Selector string
	// Type is the selector type (`jsonpath`, `xpath` or `jsonpointer`) or an Expression Type Object with a version.
	Type criterion.CriterionTypeUnion
	// Extensions provides a list of extensions to the Selector object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Selector] = (*Selector)(nil)

// Validate will validate the selector object against the Arazzo specification.
func (s *Selector) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := s.GetCore()
	errs := []error{}

	o := validation.NewOptions(opts...)

	if !specversion.IsAtLeast11(o) {
		errs = append(errs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("a selector object"), core.RootNode))
	}

	if core.Context.Present && s.Context == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("selector.context is required"), core, core.Context))
	} else if s.Context != "" {
		if err := s.Context.Validate(specversion.ExpressionOptions(o)...); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("selector.context is invalid: %w", err), core, core.Context))
		}
	}

	if core.Selector.Present && s.Selector == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("selector.selector is required"), core, core.Selector))
	}

	typeNode := core.Type.GetValueNodeOrRoot(core.RootNode)
	errs = append(errs, s.Type.ValidateAsSelectorType(typeNode, opts...)...)

	if s.Selector != "" {
		if err := validateSelectorSyntax(s.Type.GetType(), s.Selector); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("selector.selector is invalid: %w", err), core, core.Selector))
		}
	}

	s.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// validateSelectorSyntax validates a selector string against the syntax of the selector type.
// XPath selectors are not currently validated.
func validateSelectorSyntax(typ criterion.CriterionType, selector string) error {
	switch typ {
	case criterion.CriterionTypeJsonPath:
		if _, err := jsonpath.NewPath(selector); err != nil {
			return fmt.Errorf("invalid jsonpath expression: %w", err)
		}
	case criterion.CriterionTypeJsonPointer:
		if err := jsonpointer.JSONPointer(selector).Validate(); err != nil {
			return fmt.Errorf("invalid json pointer: %w", err)
		}
	case criterion.CriterionTypeXPath:
		// TODO validate xpath
	case criterion.CriterionTypeSimple, criterion.CriterionTypeRegex:
		// Not valid selector types, reported when validating the type
	}
	return nil
}

// isSelectorNode returns true if the node looks like a Selector Object (a mapping with both `context` and
// `selector` keys). Used to distinguish Selector Objects from literal object values where both are allowed.
func isSelectorNode(node *yaml.Node) bool {
	node = yml.ResolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}

	hasContext, hasSelector := false, false
	for i := 0; i+1 < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case "context":
			hasContext = true
		case "selector":
			hasSelector = true
		}
	}

	return hasContext && hasSelector
}

// validateSelectorNode validates a raw yaml node as a Selector Object, used for values (parameter values,
// payload replacement values) that are typed as raw nodes.
func validateSelectorNode(ctx context.Context, node *yaml.Node, opts ...validation.Option) []error {
	var selector Selector
	validationErrs, err := marshaller.UnmarshalNode(ctx, "selector", node, &selector)
	if err != nil {
		return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("invalid selector object: %w", err), node)}
	}

	return append(validationErrs, selector.Validate(ctx, opts...)...)
}
