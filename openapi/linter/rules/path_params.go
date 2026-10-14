package rules

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticPathParams = "semantic-path-params"

type PathParamsRule struct{}

func (r *PathParamsRule) ID() string       { return RuleSemanticPathParams }
func (r *PathParamsRule) Category() string { return CategorySemantic }
func (r *PathParamsRule) Description() string {
	return "Path template variables like {userId} must have corresponding parameter definitions with in='path', and declared path parameters must be used in the URL template. This ensures request routing works correctly and all path variables are properly documented. Parameters can be defined at PathItem level (inherited by all operations) or Operation level (can override PathItem parameters)."
}
func (r *PathParamsRule) Summary() string {
	return "Path template variables must have matching path parameters and vice versa."
}
func (r *PathParamsRule) HowToFix() string {
	return "Ensure every {param} in the path has an in: path parameter and remove any unused path parameters."
}
func (r *PathParamsRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-path-params"
}
func (r *PathParamsRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *PathParamsRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

func (r *PathParamsRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil || docInfo.Document.Paths.Len() == 0 {
		return nil
	}

	var errs []error

	// Validate each path mount, not just inline targets. The index has already
	// resolved references, but a shared target can be mounted under different
	// templates and must be checked separately for each one.
	for path, refPathItem := range docInfo.Document.Paths.All() {
		pathItem := refPathItem.GetResolvedObject()
		if pathItem == nil {
			continue
		}

		templateParams := extractParamsFromPath(path)

		// Get PathItem parameters (in: path)
		pathItemParams := getPathParameters(pathItem.Parameters)

		// Iterate operations
		for method, op := range pathItem.All() {
			// Referenced targets may be shared or live in another file. Attribute
			// usage-specific findings to the mount and do not offer a target edit:
			// the CLI currently writes fixes back to the main document only.
			node := op.GetRootNode()
			if refPathItem.IsReference() {
				node = nil
				if core := refPathItem.GetCore(); core != nil {
					node = core.Reference.KeyNode
				}
			}

			// Merge parameters
			opParams := getPathParameters(op.Parameters)
			effectiveParams := mergeParameters(pathItemParams, opParams)

			// Validate
			// 1. All template params must be in effectiveParams
			for _, tmplParam := range templateParams {
				if _, ok := effectiveParams[tmplParam]; !ok {
					finding := &validation.Error{
						Severity:        config.GetSeverity(r.DefaultSeverity()),
						Rule:            RuleSemanticPathParams,
						UnderlyingError: fmt.Errorf("path parameter `{%s}` is not defined in operation parameters", tmplParam),
						Node:            node,
					}
					if refPathItem.IsReference() {
						finding.UnderlyingError = fmt.Errorf("path parameter `{%s}` is not defined in operation parameters for `%s %s`", tmplParam, strings.ToUpper(string(method)), path)
					} else {
						schemaType, schemaFormat := inferPathParamType(tmplParam)
						finding.Fix = &addPathParameterFix{
							operationNode: node,
							paramName:     tmplParam,
							schemaType:    schemaType,
							schemaFormat:  schemaFormat,
						}
					}
					errs = append(errs, finding)
				}
			}

			// 2. All effectiveParams must be in template params
			for paramName := range effectiveParams {
				found := false
				for _, tmplParam := range templateParams {
					if tmplParam == paramName {
						found = true
						break
					}
				}
				if !found {
					message := fmt.Errorf("parameter `%s` is declared as path parameter but not used in path template `%s`", paramName, path)
					if refPathItem.IsReference() {
						message = fmt.Errorf("parameter `%s` is declared as path parameter but not used in path template `%s` for `%s`", paramName, path, strings.ToUpper(string(method)))
					}
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleSemanticPathParams,
						message,
						node,
					))
				}
			}
		}
	}

	return errs
}

func extractParamsFromPath(path string) []string {
	matches := pathParamRegex.FindAllStringSubmatch(path, -1)
	var params []string
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

func getPathParameters(params []*openapi.ReferencedParameter) map[string]bool {
	result := make(map[string]bool)

	for _, refParam := range params {
		if refParam == nil {
			continue
		}

		param := refParam.GetResolvedObject()
		if param == nil {
			continue
		}

		if param.In == "path" {
			result[param.Name] = true
		}
	}
	return result
}

func mergeParameters(base, override map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

// inferPathParamType guesses the schema type for a path parameter based on naming conventions.
func inferPathParamType(name string) (schemaType, format string) {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "uuid") || strings.Contains(lower, "guid"):
		return "string", "uuid"
	case strings.HasSuffix(lower, "id"):
		return "integer", ""
	default:
		return "string", ""
	}
}
