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
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	doc := docInfo.Document

	// Build resolve options from config
	resolveOpts := openapi.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: docInfo.Location,
	}
	if config.ResolveOptions != nil {
		resolveOpts.VirtualFS = config.ResolveOptions.VirtualFS
		resolveOpts.HTTPClient = config.ResolveOptions.HTTPClient
		resolveOpts.DisableExternalRefs = config.ResolveOptions.DisableExternalRefs
	}

	// Use the pre-computed InlinePathItems index (path items from /paths)
	// These are the only ones with path templates in their location parent key
	for _, pathItemNode := range docInfo.Index.InlinePathItems {
		refPathItem := pathItemNode.Node

		// Extract path from location (parent key of the path item)
		path := pathItemNode.Location.ParentKey()
		if path == "" {
			continue
		}

		pathItem := refPathItem.GetObject()
		if pathItem == nil {
			continue
		}

		templateParams := extractParamsFromPath(path)

		// Get PathItem parameters (in: path)
		pathItemParams, pathItemErrs := getPathParameters(ctx, pathItem.Parameters, resolveOpts, config)
		errs = append(errs, pathItemErrs...)

		// Iterate operations
		for _, op := range pathItem.All() {
			// Merge parameters
			opParams, opErrs := getPathParameters(ctx, op.Parameters, resolveOpts, config)
			errs = append(errs, opErrs...)
			effectiveParams := mergeParameters(pathItemParams, opParams)

			// Validate
			// 1. All template params must be in effectiveParams
			for _, tmplParam := range templateParams {
				if _, ok := effectiveParams[tmplParam]; !ok {
					schemaType, schemaFormat := inferPathParamType(tmplParam)
					errs = append(errs, &validation.Error{
						Severity:        config.GetSeverity(r.DefaultSeverity()),
						Rule:            RuleSemanticPathParams,
						UnderlyingError: fmt.Errorf("path parameter `{%s}` is not defined in operation parameters", tmplParam),
						Node:            op.GetRootNode(),
						Fix: &addPathParameterFix{
							operationNode: op.GetRootNode(),
							paramName:     tmplParam,
							schemaType:    schemaType,
							schemaFormat:  schemaFormat,
						},
					})
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
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleSemanticPathParams,
						fmt.Errorf("parameter `%s` is declared as path parameter but not used in path template `%s`", paramName, path),
						op.GetRootNode(),
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

func getPathParameters(ctx context.Context, params []*openapi.ReferencedParameter, resolveOpts openapi.ResolveOptions, _ *linter.RuleConfig) (map[string]bool, []error) {
	result := make(map[string]bool)
	var resolutionErrs []error

	for _, refParam := range params {
		if refParam == nil {
			continue
		}

		// Resolve reference if needed
		if refParam.IsReference() && !refParam.IsResolved() {
			validErrs, err := refParam.Resolve(ctx, resolveOpts)
			if err != nil {
				// Resolution failed - report as validation error
				resolutionErrs = append(resolutionErrs, validation.NewValidationError(
					validation.SeverityError,
					RuleSemanticPathParams,
					fmt.Errorf("failed to resolve parameter reference `%s`: %w", refParam.GetReference(), err),
					refParam.GetRootNode(),
				))
				continue
			}
			// Append any validation errors from resolution
			resolutionErrs = append(resolutionErrs, validErrs...)
		}

		// GetObject() returns the resolved object for references, or inline object
		param := refParam.GetObject()
		if param == nil {
			continue
		}

		if param.In == "path" {
			result[param.Name] = true
		}
	}
	return result, resolutionErrs
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
