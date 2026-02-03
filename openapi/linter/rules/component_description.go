package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleComponentDescription = "style-component-description"

type ComponentDescriptionRule struct{}

func (r *ComponentDescriptionRule) ID() string {
	return RuleStyleComponentDescription
}

func (r *ComponentDescriptionRule) Description() string {
	return "Reusable components (schemas, parameters, responses, etc.) should include descriptions to explain their purpose and usage. Clear component descriptions improve API documentation quality and help developers understand how to properly use shared definitions across the specification."
}

func (r *ComponentDescriptionRule) Summary() string {
	return "Reusable components (schemas, parameters, responses, etc.) should include descriptions to explain their purpose and usage."
}

func (r *ComponentDescriptionRule) HowToFix() string {
	return "Add a description to each reusable component (schemas, parameters, responses, requestBodies, headers, examples, links, callbacks, securitySchemes)."
}

func (r *ComponentDescriptionRule) Category() string {
	return CategoryStyle
}

func (r *ComponentDescriptionRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}

func (r *ComponentDescriptionRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-component-description"
}

func (r *ComponentDescriptionRule) Versions() []string {
	return nil // applies to all versions
}

func (r *ComponentDescriptionRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	doc := docInfo.Document
	components := doc.GetComponents()
	if components == nil {
		return nil
	}

	componentsCore := components.GetCore()
	if componentsCore == nil {
		return nil
	}

	componentsRoot := components.GetRootNode()

	var errs []error

	// Check schemas
	schemas := components.GetSchemas()
	if schemas != nil {
		for schemaKey := range schemas.All() {
			jsonSchema, _ := schemas.Get(schemaKey)
			if jsonSchema != nil {
				schema := jsonSchema.GetSchema()
				if schema != nil {
					description := schema.GetDescription()
					if description == "" {
						node := componentsCore.Schemas.GetMapKeyNodeOrRoot(schemaKey, componentsRoot)
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleComponentDescription,
							fmt.Errorf("`schemas` component `%s` is missing a description", schemaKey),
							node,
						))
					}
				}
			}
		}
	}

	// Check parameters
	parameters := components.GetParameters()
	if parameters != nil {
		for paramKey := range parameters.All() {
			refParam, _ := parameters.Get(paramKey)
			if refParam != nil {
				param := refParam.GetObject()
				if param != nil {
					description := param.GetDescription()
					if description == "" {
						node := componentsCore.Parameters.GetMapKeyNodeOrRoot(paramKey, componentsRoot)
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleComponentDescription,
							fmt.Errorf("`parameters` component `%s` is missing a description", paramKey),
							node,
						))
					}
				}
			}
		}
	}

	// Check requestBodies
	requestBodies := components.GetRequestBodies()
	if requestBodies != nil {
		for rbKey := range requestBodies.All() {
			refRB, _ := requestBodies.Get(rbKey)
			if refRB != nil {
				rb := refRB.GetObject()
				if rb != nil {
					description := rb.GetDescription()
					if description == "" {
						node := componentsCore.RequestBodies.GetMapKeyNodeOrRoot(rbKey, componentsRoot)
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleComponentDescription,
							fmt.Errorf("`requestBodies` component `%s` is missing a description", rbKey),
							node,
						))
					}
				}
			}
		}
	}

	// Check responses
	responses := components.GetResponses()
	if responses != nil {
		for respKey := range responses.All() {
			refResp, _ := responses.Get(respKey)
			if refResp != nil {
				resp := refResp.GetObject()
				if resp != nil {
					description := resp.GetDescription()
					if description == "" {
						node := componentsCore.Responses.GetMapKeyNodeOrRoot(respKey, componentsRoot)
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleComponentDescription,
							fmt.Errorf("`responses` component `%s` is missing a description", respKey),
							node,
						))
					}
				}
			}
		}
	}

	// Check examples
	examples := components.GetExamples()
	if examples != nil {
		for exKey := range examples.All() {
			refEx, _ := examples.Get(exKey)
			if refEx != nil {
				ex := refEx.GetObject()
				if ex != nil {
					description := ex.GetDescription()
					if description == "" {
						node := componentsCore.Examples.GetMapKeyNodeOrRoot(exKey, componentsRoot)
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleComponentDescription,
							fmt.Errorf("`examples` component `%s` is missing a description", exKey),
							node,
						))
					}
				}
			}
		}
	}

	// Check headers
	headers := components.GetHeaders()
	if headers != nil {
		for headerKey := range headers.All() {
			refHeader, _ := headers.Get(headerKey)
			if refHeader != nil {
				header := refHeader.GetObject()
				if header != nil {
					description := header.GetDescription()
					if description == "" {
						node := componentsCore.Headers.GetMapKeyNodeOrRoot(headerKey, componentsRoot)
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleComponentDescription,
							fmt.Errorf("`headers` component `%s` is missing a description", headerKey),
							node,
						))
					}
				}
			}
		}
	}

	// Check links
	links := components.GetLinks()
	if links != nil {
		for linkKey := range links.All() {
			refLink, _ := links.Get(linkKey)
			if refLink != nil {
				link := refLink.GetObject()
				if link != nil {
					description := link.GetDescription()
					if description == "" {
						node := componentsCore.Links.GetMapKeyNodeOrRoot(linkKey, componentsRoot)
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleComponentDescription,
							fmt.Errorf("`links` component `%s` is missing a description", linkKey),
							node,
						))
					}
				}
			}
		}
	}

	// Check securitySchemes
	securitySchemes := components.GetSecuritySchemes()
	if securitySchemes != nil {
		for ssKey := range securitySchemes.All() {
			refSS, _ := securitySchemes.Get(ssKey)
			if refSS != nil {
				ss := refSS.GetObject()
				if ss != nil {
					description := ss.GetDescription()
					if description == "" {
						node := componentsCore.SecuritySchemes.GetMapKeyNodeOrRoot(ssKey, componentsRoot)
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleComponentDescription,
							fmt.Errorf("`securitySchemes` component `%s` is missing a description", ssKey),
							node,
						))
					}
				}
			}
		}
	}

	return errs
}
