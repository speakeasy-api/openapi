package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOAS3ExampleMissing = "oas3-example-missing"

type OAS3ExampleMissingRule struct{}

func (r *OAS3ExampleMissingRule) ID() string {
	return RuleOAS3ExampleMissing
}
func (r *OAS3ExampleMissingRule) Category() string {
	return CategoryStyle
}
func (r *OAS3ExampleMissingRule) Description() string {
	return "Schemas, parameters, headers, and media types should include example values to illustrate expected data formats. Examples improve documentation quality, help developers understand how to use the API correctly, and enable better testing and validation."
}
func (r *OAS3ExampleMissingRule) Summary() string {
	return "Schemas, parameters, headers, and media types should include example values."
}
func (r *OAS3ExampleMissingRule) HowToFix() string {
	return "Add example or examples values to schemas, parameters, headers, and media types."
}
func (r *OAS3ExampleMissingRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#oas3-example-missing"
}
func (r *OAS3ExampleMissingRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}
func (r *OAS3ExampleMissingRule) Versions() []string {
	return nil // Applies to all versions
}

func (r *OAS3ExampleMissingRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Build a set of schemas that are used by types that have examples
	// These schemas don't need their own examples
	schemasWithExamplesElsewhere := make(map[*oas3.JSONSchemaReferenceable]bool)

	// Collect schemas from parameters with examples
	allParameters := docInfo.Index.GetAllParameters()
	for _, paramNode := range allParameters {
		param := paramNode.Node
		if param == nil {
			continue
		}
		paramObj := param.GetObject()
		if paramObj == nil {
			continue
		}
		// If parameter has example, mark its schema as having an example elsewhere
		if paramObj.GetExample() != nil || (paramObj.GetExamples() != nil && paramObj.GetExamples().Len() > 0) {
			schema := paramObj.GetSchema()
			if schema != nil {
				schemasWithExamplesElsewhere[schema] = true
			}
		}
	}

	// Collect schemas from headers with examples
	allHeaders := docInfo.Index.GetAllHeaders()
	for _, headerNode := range allHeaders {
		header := headerNode.Node
		if header == nil {
			continue
		}
		headerObj := header.GetObject()
		if headerObj == nil {
			continue
		}
		// If header has example, mark its schema as having an example elsewhere
		if headerObj.GetExample() != nil || (headerObj.GetExamples() != nil && headerObj.GetExamples().Len() > 0) {
			schema := headerObj.GetSchema()
			if schema != nil {
				schemasWithExamplesElsewhere[schema] = true
			}
		}
	}

	// Collect schemas from media types with examples
	for _, mtNode := range docInfo.Index.MediaTypes {
		mt := mtNode.Node
		if mt == nil {
			continue
		}
		// If media type has example, mark its schema as having an example elsewhere
		if mt.GetExample() != nil || (mt.GetExamples() != nil && mt.GetExamples().Len() > 0) {
			schema := mt.GetSchema()
			if schema != nil {
				schemasWithExamplesElsewhere[schema] = true
			}
		}
	}

	// Check schemas for missing examples
	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		refSchema := schemaNode.Node
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}

		// Skip if this schema is used by a parameter/header/media type that has an example
		if schemasWithExamplesElsewhere[refSchema] {
			continue
		}

		// Skip if schema has example or examples
		if schema.GetExample() != nil || len(schema.GetExamples()) > 0 {
			continue
		}

		// Skip if schema has const, default, or enum (these serve as implicit examples)
		if schema.GetConst() != nil || schema.GetDefault() != nil || len(schema.GetEnum()) > 0 {
			continue
		}

		// Skip primitive types and schemas without type
		types := schema.GetType()
		if len(types) == 0 {
			continue
		}

		// Skip boolean, number, integer, string types (unless they have no constraints)
		// These are often building blocks and don't need examples themselves
		isPrimitive := false
		for _, t := range types {
			if t == "boolean" || t == "number" || t == "integer" || t == "string" {
				isPrimitive = true
				break
			}
		}
		if isPrimitive {
			continue
		}

		// Report missing example
		if rootNode := refSchema.GetRootNode(); rootNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleOAS3ExampleMissing,
				errors.New("schema is missing `example` or `examples`"),
				rootNode,
			))
		}
	}

	// Check parameters for missing examples
	for _, paramNode := range allParameters {
		param := paramNode.Node
		if param == nil {
			continue
		}

		paramObj := param.GetObject()
		if paramObj == nil {
			continue
		}

		// Skip if parameter has example or examples
		if paramObj.GetExample() != nil {
			continue
		}
		paramExamples := paramObj.GetExamples()
		if paramExamples != nil && paramExamples.Len() > 0 {
			continue
		}

		// Report missing example
		if rootNode := param.GetRootNode(); rootNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleOAS3ExampleMissing,
				errors.New("parameter is missing `example` or `examples`"),
				rootNode,
			))
		}
	}

	// Check headers for missing examples
	for _, headerNode := range allHeaders {
		header := headerNode.Node
		if header == nil {
			continue
		}

		headerObj := header.GetObject()
		if headerObj == nil {
			continue
		}

		// Skip if header has example or examples
		if headerObj.GetExample() != nil {
			continue
		}
		headerExamples := headerObj.GetExamples()
		if headerExamples != nil && headerExamples.Len() > 0 {
			continue
		}

		// Report missing example
		if rootNode := header.GetRootNode(); rootNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleOAS3ExampleMissing,
				errors.New("header is missing `example` or `examples`"),
				rootNode,
			))
		}
	}

	// Check media types for missing examples
	for _, mtNode := range docInfo.Index.MediaTypes {
		mt := mtNode.Node
		if mt == nil {
			continue
		}

		// Skip if media type has example or examples
		if mt.GetExample() != nil {
			continue
		}
		mtExamples := mt.GetExamples()
		if mtExamples != nil && mtExamples.Len() > 0 {
			continue
		}

		// Report missing example
		if rootNode := mt.GetRootNode(); rootNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleOAS3ExampleMissing,
				errors.New("media type is missing `example` or `examples`"),
				rootNode,
			))
		}
	}

	return errs
}
