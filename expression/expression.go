package expression

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/jsonpointer"
)

// ExpressionType represents the type of expression.
type ExpressionType string

// TODO with specific in arazzo doc types we could validate the full expression include names etc match expectations
const (
	// ExpressionTypeURL indicates that the expression represents the request URL.
	ExpressionTypeURL ExpressionType = "url"
	// ExpressionTypeMethod indicates that the expression represents the request method.
	ExpressionTypeMethod ExpressionType = "method"
	// ExpressionTypeStatusCode indicates that the expression represents the response status code.
	ExpressionTypeStatusCode ExpressionType = "statusCode"
	// ExpressionTypeRequest indicates that the expression represents the request itself.
	ExpressionTypeRequest ExpressionType = "request"
	// ExpressionTypeResponse indicates that the expression represents the response itself.
	ExpressionTypeResponse ExpressionType = "response"
	// ExpressionTypeInputs indicates that the expression represents the inputs of the workflow.
	ExpressionTypeInputs ExpressionType = "inputs"
	// ExpressionTypeOutputs indicates that the expression represents the outputs of the step/workflow.
	ExpressionTypeOutputs ExpressionType = "outputs"
	// ExpressionTypeSteps indicates that the expression represents the steps of the workflow.
	ExpressionTypeSteps ExpressionType = "steps"
	// ExpressionTypeWorkflows indicates that the expression represents the workflows of the Arazzo document.
	ExpressionTypeWorkflows ExpressionType = "workflows"
	// ExpressionTypeSourceDescriptions indicates that the expression represents the source descriptions of the Arazzo document.
	ExpressionTypeSourceDescriptions ExpressionType = "sourceDescriptions"
	// ExpressionTypeComponents indicates that the expression represents the components of the Arazzo or OpenAPI document.
	ExpressionTypeComponents ExpressionType = "components"
	// ExpressionTypeMessage indicates that the expression represents an AsyncAPI message (Arazzo 1.1+ only).
	ExpressionTypeMessage ExpressionType = "message"
	// ExpressionTypeSelf indicates that the expression represents the canonical URI of the current Arazzo document as defined by its `$self` field (Arazzo 1.1+ only).
	ExpressionTypeSelf ExpressionType = "self"
)

const (
	// ReferenceTypeHeader indicates that the expression represents a header reference in the request/response.
	ReferenceTypeHeader = "header"
	// ReferenceTypeQuery indicates that the expression represents a query reference in the request.
	ReferenceTypeQuery = "query"
	// ReferenceTypePath indicates that the expression represents a path reference in the request.
	ReferenceTypePath = "path"
	// ReferenceTypeBody indicates that the expression represents a body reference in the request/response.
	ReferenceTypeBody = "body"
	// ReferenceTypePayload indicates that the expression represents a payload reference in the request/response/message (Arazzo 1.1+ only).
	ReferenceTypePayload = "payload"
)

var expressions = []string{
	string(ExpressionTypeURL),
	string(ExpressionTypeMethod),
	string(ExpressionTypeStatusCode),
	string(ExpressionTypeRequest),
	string(ExpressionTypeResponse),
	string(ExpressionTypeInputs),
	string(ExpressionTypeOutputs),
	string(ExpressionTypeSteps),
	string(ExpressionTypeWorkflows),
	string(ExpressionTypeSourceDescriptions),
	string(ExpressionTypeComponents),
}

var arazzo11Expressions = append(slices.Clone(expressions),
	string(ExpressionTypeMessage),
	string(ExpressionTypeSelf),
)

var referenceTypes = []string{
	ReferenceTypeHeader,
	ReferenceTypeQuery,
	ReferenceTypePath,
	ReferenceTypeBody,
}

var arazzo11ReferenceTypes = append(slices.Clone(referenceTypes), ReferenceTypePayload)

var arazzo11ComponentTypes = []string{"parameters", "successActions", "failureActions"}

var (
	tokenRegex = regexp.MustCompile("^[!#$%&'*+\\-.^_`|~\\dA-Za-z]+$")
	nameRegex  = regexp.MustCompile("^[\x01-\x7F]+$")
	// identifierRegex matches the Arazzo 1.1 ABNF `identifier` rule.
	identifierRegex = regexp.MustCompile(`^[A-Za-z0-9._\-]+$`)
	// identifierStrictRegex matches the Arazzo 1.1 ABNF `identifier-strict` rule.
	identifierStrictRegex = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)
)

// ValidateOption configures how expressions are recognized and validated.
type ValidateOption func(o *validateOptions)

type validateOptions struct {
	arazzo11 bool
}

// WithArazzo11Grammar enables the Arazzo 1.1 runtime expression grammar.
//
// This adds the `$message` and `$self` expression roots and the `payload` source, and
// applies the stricter Arazzo 1.1 rules for `$inputs`, `$outputs`, `$steps`, `$workflows`,
// `$sourceDescriptions` and `$components` references. Without this option expressions are
// validated with the original OpenAPI/Arazzo 1.0 rules.
func WithArazzo11Grammar() ValidateOption {
	return func(o *validateOptions) {
		o.arazzo11 = true
	}
}

func newValidateOptions(opts []ValidateOption) validateOptions {
	o := validateOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func (o validateOptions) knownExpressions() []string {
	if o.arazzo11 {
		return arazzo11Expressions
	}
	return expressions
}

// Expression represents a runtime expression as defined by the OpenAPI & Arazzo specifications.
type Expression string

var _ fmt.Stringer = (*Expression)(nil)

func (e Expression) String() string {
	return string(e)
}

// Validate will validate the expression is valid as per the OpenAPI & Arazzo specifications.
// Pass WithArazzo11Grammar() to validate against the Arazzo 1.1 runtime expression grammar.
func (e Expression) Validate(opts ...ValidateOption) error {
	o := newValidateOptions(opts)

	// First check basic format using ExtractExpressions
	extractedExpressions := ExtractExpressions(string(e))
	if len(extractedExpressions) == 0 {
		return fmt.Errorf("expression is not valid, must begin with $: %s", string(e))
	}

	allowJsonPointers := false

	typ, reference, expressionParts, jp := e.GetParts()

	if string(typ) == "" {
		return fmt.Errorf("expression is not valid, must begin with one of [%s]: %s", strings.Join(o.knownExpressions(), ", "), string(e))
	}

	if o.arazzo11 {
		return e.validateArazzo11(typ, reference, expressionParts, jp)
	}

	switch typ {
	case ExpressionTypeURL, ExpressionTypeMethod, ExpressionTypeStatusCode:
		if reference != "" || len(expressionParts) > 0 {
			return fmt.Errorf("expression is not valid, extra characters after $%s: %s", typ, string(e))
		}
	case ExpressionTypeRequest, ExpressionTypeResponse:
		if reference == "" {
			return fmt.Errorf("expression is not valid, expected one of [%s] after $%s: %s", strings.Join(referenceTypes, ", "), typ, string(e))
		}

		switch reference {
		case ReferenceTypeBody:
			allowJsonPointers = true

			if len(expressionParts) > 0 {
				return fmt.Errorf("expression is not valid, only json pointers are allowed after $%s.%s: %s", typ, reference, string(e))
			}
		case ReferenceTypeHeader:
			if len(expressionParts) != 1 {
				return fmt.Errorf("expression is not valid, expected token after $%s.%s: %s", typ, reference, string(e))
			}

			if !tokenRegex.MatchString(expressionParts[0]) {
				return fmt.Errorf("header reference must be a valid token [%s]: %s", tokenRegex.String(), string(e))
			}
		case ReferenceTypeQuery:
			if len(expressionParts) != 1 {
				return fmt.Errorf("expression is not valid, expected name after $%s.%s: %s", typ, reference, string(e))
			}

			if err := validateName(string(e), expressionParts[0], "query reference"); err != nil {
				return err
			}
		case ReferenceTypePath:
			if len(expressionParts) != 1 {
				return fmt.Errorf("expression is not valid, expected name after $%s.%s: %s", typ, reference, string(e))
			}

			if err := validateName(string(e), expressionParts[0], "path reference"); err != nil {
				return err
			}
		default:
			return fmt.Errorf("expression is not valid, expected one of [%s] after $%s: %s", strings.Join(referenceTypes, ", "), typ, string(e))
		}
	case ExpressionTypeInputs, ExpressionTypeOutputs, ExpressionTypeSteps, ExpressionTypeWorkflows, ExpressionTypeSourceDescriptions, ExpressionTypeComponents:
		if reference == "" {
			return fmt.Errorf("expression is not valid, expected name after $%s: %s", typ, string(e))
		}

		name := strings.Join(append([]string{reference}, expressionParts...), ".")

		if err := validateName(string(e), name, "name reference"); err != nil {
			return err
		}

		switch {
		case typ == ExpressionTypeSourceDescriptions && strings.HasSuffix(name, "url"):
			allowJsonPointers = true
		case slices.Contains([]ExpressionType{ExpressionTypeSteps, ExpressionTypeWorkflows}, typ):
			if len(expressionParts) > 0 && expressionParts[0] == "outputs" {
				allowJsonPointers = true
			}
		case typ == ExpressionTypeOutputs:
			allowJsonPointers = true
		}
	default:
		return fmt.Errorf("expression is not valid, must begin with one of [%s]: %s", strings.Join(expressions, ", "), string(e))
	}

	return e.validateJSONPointer(jp, allowJsonPointers)
}

func (e Expression) validateJSONPointer(jp jsonpointer.JSONPointer, allowed bool) error {
	if jp == "" {
		return nil
	}

	if !allowed {
		return fmt.Errorf("expression is not valid, json pointers are not allowed in current context: %s", string(e))
	}

	return jp.Validate()
}

// validateArazzo11 validates the expression against the Arazzo 1.1 runtime expression ABNF.
func (e Expression) validateArazzo11(typ ExpressionType, reference string, expressionParts []string, jp jsonpointer.JSONPointer) error {
	allowJSONPointers := false

	switch typ {
	case ExpressionTypeURL, ExpressionTypeMethod, ExpressionTypeStatusCode, ExpressionTypeSelf:
		if reference != "" || len(expressionParts) > 0 {
			return fmt.Errorf("expression is not valid, extra characters after $%s: %s", typ, string(e))
		}
	case ExpressionTypeRequest, ExpressionTypeResponse, ExpressionTypeMessage:
		if reference == "" {
			return fmt.Errorf("expression is not valid, expected one of [%s] after $%s: %s", strings.Join(arazzo11ReferenceTypes, ", "), typ, string(e))
		}

		switch reference {
		case ReferenceTypeBody, ReferenceTypePayload:
			allowJSONPointers = true

			if len(expressionParts) > 0 {
				return fmt.Errorf("expression is not valid, only json pointers are allowed after $%s.%s: %s", typ, reference, string(e))
			}
		case ReferenceTypeHeader:
			if len(expressionParts) != 1 {
				return fmt.Errorf("expression is not valid, expected token after $%s.%s: %s", typ, reference, string(e))
			}

			if !tokenRegex.MatchString(expressionParts[0]) {
				return fmt.Errorf("header reference must be a valid token [%s]: %s", tokenRegex.String(), string(e))
			}
		case ReferenceTypeQuery, ReferenceTypePath:
			if len(expressionParts) != 1 {
				return fmt.Errorf("expression is not valid, expected name after $%s.%s: %s", typ, reference, string(e))
			}

			if err := validateName(string(e), expressionParts[0], reference+" reference"); err != nil {
				return err
			}
		default:
			return fmt.Errorf("expression is not valid, expected one of [%s] after $%s: %s", strings.Join(arazzo11ReferenceTypes, ", "), typ, string(e))
		}
	case ExpressionTypeInputs, ExpressionTypeOutputs:
		// inputs-reference / outputs-reference = identifier [ "#" json-pointer ]
		name := strings.Join(append([]string{reference}, expressionParts...), ".")
		if !identifierRegex.MatchString(name) {
			return fmt.Errorf("expression is not valid, expected a name matching [%s] after $%s: %s", identifierRegex.String(), typ, string(e))
		}
		allowJSONPointers = true
	case ExpressionTypeSteps:
		// steps-reference = step-id ".outputs." output-name [ "#" json-pointer ]
		if !identifierStrictRegex.MatchString(reference) || len(expressionParts) < 2 || expressionParts[0] != "outputs" {
			return fmt.Errorf("expression is not valid, expected $steps.<stepId>.outputs.<name>: %s", string(e))
		}
		if !identifierRegex.MatchString(strings.Join(expressionParts[1:], ".")) {
			return fmt.Errorf("expression is not valid, output name must match [%s]: %s", identifierRegex.String(), string(e))
		}
		allowJSONPointers = true
	case ExpressionTypeWorkflows:
		// workflows-reference = workflow-id "." ("inputs" / "outputs") "." workflow-field-name [ "#" json-pointer ]
		if !identifierStrictRegex.MatchString(reference) || len(expressionParts) < 2 || (expressionParts[0] != "inputs" && expressionParts[0] != "outputs") {
			return fmt.Errorf("expression is not valid, expected $workflows.<workflowId>.inputs.<name> or $workflows.<workflowId>.outputs.<name>: %s", string(e))
		}
		if !identifierRegex.MatchString(strings.Join(expressionParts[1:], ".")) {
			return fmt.Errorf("expression is not valid, %s name must match [%s]: %s", expressionParts[0], identifierRegex.String(), string(e))
		}
		allowJSONPointers = true
	case ExpressionTypeSourceDescriptions:
		// source-reference = source-name "." source-reference-id
		if !identifierStrictRegex.MatchString(reference) || len(expressionParts) == 0 {
			return fmt.Errorf("expression is not valid, expected $sourceDescriptions.<name>.<reference>: %s", string(e))
		}
		if err := validateName(string(e), strings.Join(expressionParts, "."), "source reference"); err != nil {
			return err
		}
		// Allow `{$sourceDescriptions.<name>.url}#/json/pointer` as used by operationPath and channelPath.
		allowJSONPointers = expressionParts[len(expressionParts)-1] == "url"
	case ExpressionTypeComponents:
		// components-reference = ("parameters" / "successActions" / "failureActions") "." identifier
		if !slices.Contains(arazzo11ComponentTypes, reference) || len(expressionParts) == 0 {
			return fmt.Errorf("expression is not valid, expected $components.<%s>.<name>: %s", strings.Join(arazzo11ComponentTypes, "|"), string(e))
		}
		if !identifierRegex.MatchString(strings.Join(expressionParts, ".")) {
			return fmt.Errorf("expression is not valid, component name must match [%s]: %s", identifierRegex.String(), string(e))
		}
	default:
		return fmt.Errorf("expression is not valid, must begin with one of [%s]: %s", strings.Join(arazzo11Expressions, ", "), string(e))
	}

	return e.validateJSONPointer(jp, allowJSONPointers)
}

// IsExpression will return true if the expression is a runtime expression that starts with a valid expression type.
// Pass WithArazzo11Grammar() to also recognize the Arazzo 1.1 `$message` and `$self` expressions.
func (e Expression) IsExpression(opts ...ValidateOption) bool {
	o := newValidateOptions(opts)

	extractedExpressions := ExtractExpressions(string(e))

	if len(extractedExpressions) != 1 {
		return false
	}

	if len(extractedExpressions[0]) != len(string(e)) {
		return false
	}

	// Check if the expression starts with a valid expression type
	typ := e.GetType()
	if string(typ) == "" {
		return false
	}

	// Verify the type is in the list of valid expression types
	return slices.Contains(o.knownExpressions(), string(typ))
}

// GetType will return the type of the expression.
func (e Expression) GetType() ExpressionType {
	typ, _, _, _ := e.GetParts()
	return typ
}

// GetParts will return the type, reference, expression parts and jsonpointer of the expression.
func (e Expression) GetParts() (ExpressionType, string, []string, jsonpointer.JSONPointer) {
	parts := strings.Split(string(e), "#")
	if len(parts) < 1 {
		return "", "", nil, jsonpointer.JSONPointer("")
	}
	expressionParts, typ := getType(parts[0])

	reference := ""
	if len(expressionParts) > 0 {
		reference = expressionParts[0]
		expressionParts = expressionParts[1:]
	}

	var jp jsonpointer.JSONPointer
	if len(parts) > 1 {
		jp = jsonpointer.JSONPointer(parts[1])
	}

	return typ, reference, expressionParts, jp
}

// GetJSONPointer will return the jsonpointer of the expression.
func (e Expression) GetJSONPointer() jsonpointer.JSONPointer {
	_, _, _, jp := e.GetParts()
	return jp
}

func getType(expression string) ([]string, ExpressionType) {
	rawExpression := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(expression, "{"), "$"), "}")
	expressionParts := strings.Split(rawExpression, ".")

	var typ ExpressionType
	if len(expressionParts) > 0 {
		typ = ExpressionType(expressionParts[0])
		expressionParts = expressionParts[1:]
	}

	return expressionParts, typ
}

// TODO the spec is currently ambiguous on how to handle any additional dot separated parts after the name so just treat as a name for now
// TODO there is probably something required to handle dots within a key name
func validateName(expression, name, referenceType string) error {
	parts := strings.Split(name, ".")
	for _, part := range parts {
		if !nameRegex.MatchString(part) {
			return fmt.Errorf("%s must be a valid name [%s]: %s", referenceType, nameRegex.String(), expression)
		}
	}

	return nil
}
