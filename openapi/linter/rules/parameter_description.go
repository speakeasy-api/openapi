package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOAS3ParameterDescription = "style-oas3-parameter-description"

type OAS3ParameterDescriptionRule struct{}

func (r *OAS3ParameterDescriptionRule) ID() string {
	return RuleStyleOAS3ParameterDescription
}

func (r *OAS3ParameterDescriptionRule) Description() string {
	return "Parameters should include descriptions that explain their purpose and expected values. Clear parameter documentation helps developers understand how to construct valid requests and what each parameter controls."
}

func (r *OAS3ParameterDescriptionRule) Summary() string {
	return "Parameters should include descriptions."
}

func (r *OAS3ParameterDescriptionRule) HowToFix() string {
	return "Add descriptions to parameters that explain their purpose and expected values."
}

func (r *OAS3ParameterDescriptionRule) Category() string {
	return CategoryStyle
}

func (r *OAS3ParameterDescriptionRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *OAS3ParameterDescriptionRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-oas3-parameter-description"
}

func (r *OAS3ParameterDescriptionRule) Versions() []string {
	return []string{"3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1.0", "3.1.1", "3.2.0"}
}

func (r *OAS3ParameterDescriptionRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	doc := docInfo.Document
	var errs []error

	// Check only inline parameters (component parameters are checked by component_description rule)
	for _, paramNode := range docInfo.Index.InlineParameters {
		refParam := paramNode.Node
		if refParam == nil {
			continue
		}

		param := refParam.GetObject()
		if param == nil {
			continue
		}

		description := param.GetDescription()
		if description == "" {
			paramName := param.GetName()

			// Extract method and path manually from location
			var method, path string
			for _, loc := range paramNode.Location {
				switch openapi.GetParentType(loc) {
				case "Paths":
					if loc.ParentKey != nil {
						path = *loc.ParentKey
					}
				case "PathItem":
					if loc.ParentKey != nil {
						method = *loc.ParentKey
					}
				}
			}

			errNode := GetFieldValueNode(param, "description", doc)
			if errNode == nil {
				errNode = param.GetRootNode()
			}

			var msg string
			if method != "" && path != "" {
				msg = "parameter `" + paramName + "` in `" + method + " " + path + "` is missing a description"
			} else {
				msg = "parameter `" + paramName + "` is missing a description"
			}

			paramRootNode := param.GetRootNode()
			errs = append(errs, &validation.Error{
				UnderlyingError: errors.New(msg),
				Node:            errNode,
				Severity:        config.GetSeverity(r.DefaultSeverity()),
				Rule:            RuleStyleOAS3ParameterDescription,
				Fix:             &addDescriptionFix{targetNode: paramRootNode, targetLabel: "parameter '" + paramName + "'"},
			})
		}
	}

	return errs
}
