package expression

import (
	"fmt"
	"regexp"
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
	// ExpressionTypeComponents indicates that the expression represents the components of the Arazzo document.
	ExpressionTypeComponents ExpressionType = "components"
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

var referenceTypes = []string{
	ReferenceTypeHeader,
	ReferenceTypeQuery,
	ReferenceTypePath,
	ReferenceTypeBody,
}

var (
	tokenRegex = regexp.MustCompile("^[!#$%&'*+\\-.^_`|~\\dA-Za-z]+$")
	nameRegex  = regexp.MustCompile("^[\x01-\x7F]+$")
)

// Expression represents a runtime expression as defined by the Arazzo specification.
type Expression string

// Validate will validate the expression is valid as per the Arazzo specification.
func (e Expression) Validate(validateAsExpression bool) error {
	if !e.IsExpression() {
		if !validateAsExpression {
			return nil
		} else {
			return fmt.Errorf("expression is not valid, must begin with $: %s", string(e))
		}
	}

	allowJsonPointers := false

	typ, reference, expressionParts, jp := e.GetParts()

	if string(typ) == "" {
		return fmt.Errorf("expression is not valid, must begin with one of [%s]: %s", strings.Join(expressions, ", "), string(e))
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

		if typ == ExpressionTypeSourceDescriptions && strings.HasSuffix(name, "url") {
			allowJsonPointers = true
		}
	default:
		return fmt.Errorf("expression is not valid, must begin with one of [%s]: %s", strings.Join(expressions, ", "), string(e))
	}

	if jp != "" {
		if allowJsonPointers {
			if err := jp.Validate(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("expression is not valid, json pointers are not allowed in current context: %s", string(e))
		}
	}

	return nil
}

// IsExpression will return true if the expression is a runtime expression or just an identifier/string.
func (e Expression) IsExpression() bool {
	expressions := ExtractExpressions(string(e))

	if len(expressions) != 1 {
		return false
	}

	if len(expressions[0]) != len(string(e)) {
		return false
	}

	return true
}

// GetType will return the type of the expression.
func (e Expression) GetType() ExpressionType {
	typ, _, _, _ := e.GetParts()
	return typ
}

// GetParts will return the type, reference, expression parts and jsonpointer of the expression.
func (e Expression) GetParts() (ExpressionType, string, []string, jsonpointer.JSONPointer) {
	parts := strings.Split(string(e), "#")
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

// TODO the spec is currently ambiguous on how to handle any additional dot seperated parts after the name so just treat as a name for now
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
