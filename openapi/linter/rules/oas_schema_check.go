package rules

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

var _ linter.Rule = (*OASSchemaCheckRule)(nil)

// OASSchemaCheckRule validates that schemas contain appropriate constraints for their types
type OASSchemaCheckRule struct{}

func (r *OASSchemaCheckRule) ID() string {
	return "oas-schema-check"
}

func (r *OASSchemaCheckRule) Category() string {
	return CategorySchemas
}

func (r *OASSchemaCheckRule) Description() string {
	return "Schemas must use type-appropriate constraints and have valid constraint values. For example, `string` types should use `minLength`/`maxLength`/`pattern`, numbers should use `minimum`/`maximum`/`multipleOf`, and constraint values must be logically valid (e.g., `maxLength` >= `minLength`)."
}

func (r *OASSchemaCheckRule) Summary() string {
	return "Schemas must use type-appropriate constraints with valid values."
}

func (r *OASSchemaCheckRule) HowToFix() string {
	return "Add or correct constraints appropriate to each schema type (e.g., `minLength`/`maxLength` for strings, `minimum`/`maximum` for numbers)."
}

func (r *OASSchemaCheckRule) Link() string {
	return "https://quobix.com/vacuum/rules/schemas/oas-schema-check/"
}

func (r *OASSchemaCheckRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}

func (r *OASSchemaCheckRule) Versions() []string {
	return nil // applies to all versions
}

func (r *OASSchemaCheckRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	for _, schemaNode := range docInfo.Index.GetAllSchemas() {
		refSchema := schemaNode.Node
		schema := refSchema.GetSchema()
		if schema == nil {
			continue
		}

		coreSchema := schema.GetCore()
		if coreSchema == nil {
			continue
		}

		schemaTypes := schema.GetType()

		// Validate each type
		for _, schemaType := range schemaTypes {
			typeStr := string(schemaType)
			switch typeStr {
			case "string":
				errs = append(errs, r.validateString(ctx, schema, refSchema, docInfo, config)...)
			case "integer", "number":
				errs = append(errs, r.validateNumber(ctx, schema, refSchema, docInfo, config)...)
			case "boolean":
				errs = append(errs, r.validateBoolean(ctx, schema, refSchema, docInfo, config)...)
			case "array":
				errs = append(errs, r.validateArray(ctx, schema, refSchema, docInfo, config)...)
			case "object":
				errs = append(errs, r.validateObject(ctx, schema, refSchema, docInfo, config)...)
			case "null":
				errs = append(errs, r.validateNull(ctx, schema, refSchema, schemaTypes, docInfo, config)...)
			default:
				// Unknown type
				if coreSchema.Type.Present {
					if rootNode := refSchema.GetRootNode(); rootNode != nil {
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							r.ID(),
							fmt.Errorf("unknown schema type: `%s`", typeStr),
							rootNode,
						))
					}
				}
			}
		}

		// Validate const value matches declared types
		if len(schemaTypes) > 0 {
			errs = append(errs, r.validateConst(ctx, schema, refSchema, schemaTypes, docInfo, config)...)
		}

		// Validate enum and const are not conflicting
		errs = append(errs, r.validateEnumConst(ctx, schema, refSchema, docInfo, config)...)

		// Validate discriminator property existence
		errs = append(errs, r.validateDiscriminator(ctx, schema, refSchema, docInfo, config)...)
	}

	return errs
}

func (r *OASSchemaCheckRule) validateString(ctx context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	errs = append(errs, r.checkTypeMismatchedConstraints(ctx, schema, refSchema, "string", docInfo, config)...)

	coreSchema := schema.GetCore()

	// Validate minLength
	if coreSchema.MinLength.Present && schema.MinLength != nil {
		if *schema.MinLength < 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`minLength` should be a non-negative number"),
					rootNode,
				))
			}
		}
	}

	// Validate maxLength
	if coreSchema.MaxLength.Present && schema.MaxLength != nil {
		if *schema.MaxLength < 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`maxLength` should be a non-negative number"),
					rootNode,
				))
			}
		}
		if coreSchema.MinLength.Present && schema.MinLength != nil {
			if *schema.MinLength > *schema.MaxLength {
				if rootNode := refSchema.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						r.ID(),
						errors.New("`maxLength` should be greater than or equal to `minLength`"),
						rootNode,
					))
				}
			}
		}
	}

	// Validate pattern is valid regex
	if coreSchema.Pattern.Present && schema.Pattern != nil && *schema.Pattern != "" {
		_, err := regexp.Compile(*schema.Pattern)
		if err != nil {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("schema `pattern` should be a valid regular expression"),
					rootNode,
				))
			}
		}
	}

	return errs
}

func (r *OASSchemaCheckRule) validateNumber(ctx context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	errs = append(errs, r.checkTypeMismatchedConstraints(ctx, schema, refSchema, "number", docInfo, config)...)

	coreSchema := schema.GetCore()

	// Validate multipleOf
	if coreSchema.MultipleOf.Present && schema.MultipleOf != nil {
		if *schema.MultipleOf <= 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`multipleOf` should be a number greater than `0`"),
					rootNode,
				))
			}
		}
	}

	// Validate maximum >= minimum
	if coreSchema.Maximum.Present && schema.Maximum != nil {
		if coreSchema.Minimum.Present && schema.Minimum != nil {
			if *schema.Maximum < *schema.Minimum {
				if rootNode := refSchema.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						r.ID(),
						errors.New("`maximum` should be a number greater than or equal to `minimum`"),
						rootNode,
					))
				}
			}
		}
	}

	// Validate exclusiveMaximum >= exclusiveMinimum (only when both are numbers)
	if schema.ExclusiveMaximum != nil && schema.ExclusiveMaximum.IsRight() &&
		schema.ExclusiveMinimum != nil && schema.ExclusiveMinimum.IsRight() {
		exclusiveMax := schema.ExclusiveMaximum.GetRight()
		exclusiveMin := schema.ExclusiveMinimum.GetRight()
		if exclusiveMax != nil && exclusiveMin != nil && *exclusiveMax < *exclusiveMin {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`exclusiveMaximum` should be greater than or equal to `exclusiveMinimum`"),
					rootNode,
				))
			}
		}
	}

	return errs
}

func (r *OASSchemaCheckRule) validateArray(ctx context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	errs = append(errs, r.checkTypeMismatchedConstraints(ctx, schema, refSchema, "array", docInfo, config)...)

	coreSchema := schema.GetCore()

	// Validate minItems
	if coreSchema.MinItems.Present && schema.MinItems != nil {
		if *schema.MinItems < 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`minItems` should be a non-negative number"),
					rootNode,
				))
			}
		}
	}

	// Validate maxItems
	if coreSchema.MaxItems.Present && schema.MaxItems != nil {
		if *schema.MaxItems < 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`maxItems` should be a non-negative number"),
					rootNode,
				))
			}
		}
		if coreSchema.MinItems.Present && schema.MinItems != nil {
			if *schema.MinItems > *schema.MaxItems {
				if rootNode := refSchema.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						r.ID(),
						errors.New("`maxItems` should be greater than or equal to `minItems`"),
						rootNode,
					))
				}
			}
		}
	}

	// Validate minContains
	if coreSchema.MinContains.Present && schema.MinContains != nil {
		if *schema.MinContains < 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`minContains` should be a non-negative number"),
					rootNode,
				))
			}
		}
	}

	// Validate maxContains
	if coreSchema.MaxContains.Present && schema.MaxContains != nil {
		if *schema.MaxContains < 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`maxContains` should be a non-negative number"),
					rootNode,
				))
			}
		}
		if coreSchema.MinContains.Present && schema.MinContains != nil {
			if *schema.MinContains > *schema.MaxContains {
				if rootNode := refSchema.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						r.ID(),
						errors.New("`maxContains` should be greater than or equal to `minContains`"),
						rootNode,
					))
				}
			}
		}
	}

	return errs
}

func (r *OASSchemaCheckRule) validateObject(ctx context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	errs = append(errs, r.checkTypeMismatchedConstraints(ctx, schema, refSchema, "object", docInfo, config)...)

	coreSchema := schema.GetCore()

	// Validate minProperties
	if coreSchema.MinProperties.Present && schema.MinProperties != nil {
		if *schema.MinProperties < 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`minProperties` should be a non-negative number"),
					rootNode,
				))
			}
		}
	}

	// Validate maxProperties
	if coreSchema.MaxProperties.Present && schema.MaxProperties != nil {
		if *schema.MaxProperties < 0 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("`maxProperties` should be a non-negative number"),
					rootNode,
				))
			}
		}
		if coreSchema.MinProperties.Present && schema.MinProperties != nil {
			if *schema.MinProperties > *schema.MaxProperties {
				if rootNode := refSchema.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						r.ID(),
						errors.New("`maxProperties` should be greater than or equal to `minProperties`"),
						rootNode,
					))
				}
			}
		}
	}

	// Validate required fields
	requiredFields := schema.Required
	if len(requiredFields) > 0 {
		properties := schema.Properties

		// Check if there's any polymorphic composition
		polyFound := len(schema.AnyOf) > 0 || len(schema.OneOf) > 0 || len(schema.AllOf) > 0

		// If no properties and no polymorphic composition, error
		if properties == nil && !polyFound {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("object contains `required` fields but no `properties`"),
					rootNode,
				))
			}
		} else {
			// Check each required field
			for _, required := range requiredFields {
				propertyExists := false

				// Check in direct properties
				if properties != nil {
					for propName := range properties.All() {
						if propName == required {
							propertyExists = true
							break
						}
					}
				}

				// Check in polymorphic schemas if not found
				if !propertyExists {
					propertyExists = r.checkPolymorphicProperty(schema, required)
				}

				if !propertyExists {
					if rootNode := refSchema.GetRootNode(); rootNode != nil {
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							r.ID(),
							fmt.Errorf("required property `%s` is not defined in schema `properties`", required),
							rootNode,
						))
					}
				}
			}
		}
	}

	return errs
}

func (r *OASSchemaCheckRule) validateBoolean(ctx context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	return r.checkTypeMismatchedConstraints(ctx, schema, refSchema, "boolean", docInfo, config)
}

func (r *OASSchemaCheckRule) validateNull(ctx context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, schemaTypes []oas3.SchemaType, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	// In OAS 3.1, nullable is expressed as type: [actualType, "null"]
	// Don't check constraints when null is part of a multi-type array
	if len(schemaTypes) > 1 {
		return nil
	}
	return r.checkTypeMismatchedConstraints(ctx, schema, refSchema, "null", docInfo, config)
}

func (r *OASSchemaCheckRule) checkTypeMismatchedConstraints(_ context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, schemaType string, _ *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error
	coreSchema := schema.GetCore()

	// Define which constraint types are invalid for this type
	var invalidConstraints []struct {
		field    string
		validFor string
	}

	switch schemaType {
	case "string":
		invalidConstraints = []struct {
			field    string
			validFor string
		}{
			// Number constraints
			{"minimum", "number/integer"},
			{"maximum", "number/integer"},
			{"multipleOf", "number/integer"},
			{"exclusiveMinimum", "number/integer"},
			{"exclusiveMaximum", "number/integer"},
			// Array constraints
			{"minItems", "array"},
			{"maxItems", "array"},
			{"uniqueItems", "array"},
			{"minContains", "array"},
			{"maxContains", "array"},
			// Object constraints
			{"minProperties", "object"},
			{"maxProperties", "object"},
		}
	case "number", "integer":
		invalidConstraints = []struct {
			field    string
			validFor string
		}{
			// String constraints
			{"pattern", "string"},
			{"minLength", "string"},
			{"maxLength", "string"},
			// Array constraints
			{"minItems", "array"},
			{"maxItems", "array"},
			{"uniqueItems", "array"},
			{"minContains", "array"},
			{"maxContains", "array"},
			// Object constraints
			{"minProperties", "object"},
			{"maxProperties", "object"},
		}
	case "array":
		invalidConstraints = []struct {
			field    string
			validFor string
		}{
			// String constraints
			{"pattern", "string"},
			{"minLength", "string"},
			{"maxLength", "string"},
			// Number constraints
			{"minimum", "number/integer"},
			{"maximum", "number/integer"},
			{"multipleOf", "number/integer"},
			{"exclusiveMinimum", "number/integer"},
			{"exclusiveMaximum", "number/integer"},
			// Object constraints
			{"minProperties", "object"},
			{"maxProperties", "object"},
		}
	case "object":
		invalidConstraints = []struct {
			field    string
			validFor string
		}{
			// String constraints
			{"pattern", "string"},
			{"minLength", "string"},
			{"maxLength", "string"},
			// Number constraints
			{"minimum", "number/integer"},
			{"maximum", "number/integer"},
			{"multipleOf", "number/integer"},
			{"exclusiveMinimum", "number/integer"},
			{"exclusiveMaximum", "number/integer"},
			// Array constraints
			{"minItems", "array"},
			{"maxItems", "array"},
			{"uniqueItems", "array"},
			{"minContains", "array"},
			{"maxContains", "array"},
		}
	case "boolean", "null":
		invalidConstraints = []struct {
			field    string
			validFor string
		}{
			// String constraints
			{"pattern", "string"},
			{"minLength", "string"},
			{"maxLength", "string"},
			// Number constraints
			{"minimum", "number/integer"},
			{"maximum", "number/integer"},
			{"multipleOf", "number/integer"},
			{"exclusiveMinimum", "number/integer"},
			{"exclusiveMaximum", "number/integer"},
			// Array constraints
			{"minItems", "array"},
			{"maxItems", "array"},
			{"uniqueItems", "array"},
			{"minContains", "array"},
			{"maxContains", "array"},
			// Object constraints
			{"minProperties", "object"},
			{"maxProperties", "object"},
		}
	}

	// Check for mismatched constraints
	for _, constraint := range invalidConstraints {
		var isPresent bool
		switch constraint.field {
		case "pattern":
			isPresent = coreSchema.Pattern.Present
		case "minLength":
			isPresent = coreSchema.MinLength.Present
		case "maxLength":
			isPresent = coreSchema.MaxLength.Present
		case "minimum":
			isPresent = coreSchema.Minimum.Present
		case "maximum":
			isPresent = coreSchema.Maximum.Present
		case "multipleOf":
			isPresent = coreSchema.MultipleOf.Present
		case "exclusiveMinimum":
			isPresent = coreSchema.ExclusiveMinimum.Present
		case "exclusiveMaximum":
			isPresent = coreSchema.ExclusiveMaximum.Present
		case "minItems":
			isPresent = coreSchema.MinItems.Present
		case "maxItems":
			isPresent = coreSchema.MaxItems.Present
		case "uniqueItems":
			isPresent = coreSchema.UniqueItems.Present
		case "minContains":
			isPresent = coreSchema.MinContains.Present
		case "maxContains":
			isPresent = coreSchema.MaxContains.Present
		case "minProperties":
			isPresent = coreSchema.MinProperties.Present
		case "maxProperties":
			isPresent = coreSchema.MaxProperties.Present
		}

		if isPresent {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					fmt.Errorf("`%s` constraint is only applicable to %s types, not `%s`",
						constraint.field, constraint.validFor, schemaType),
					rootNode,
				))
			}
		}
	}

	return errs
}

func (r *OASSchemaCheckRule) checkPolymorphicProperty(schema *oas3.Schema, propertyName string) bool {
	// Check in AnyOf schemas
	for _, anyOfRef := range schema.AnyOf {
		anyOfSchema := anyOfRef.GetSchema()
		if anyOfSchema != nil && anyOfSchema.Properties != nil {
			for propName := range anyOfSchema.Properties.All() {
				if propName == propertyName {
					return true
				}
			}
		}
	}

	// Check in OneOf schemas
	for _, oneOfRef := range schema.OneOf {
		oneOfSchema := oneOfRef.GetSchema()
		if oneOfSchema != nil && oneOfSchema.Properties != nil {
			for propName := range oneOfSchema.Properties.All() {
				if propName == propertyName {
					return true
				}
			}
		}
	}

	// Check in AllOf schemas
	for _, allOfRef := range schema.AllOf {
		allOfSchema := allOfRef.GetSchema()
		if allOfSchema != nil && allOfSchema.Properties != nil {
			for propName := range allOfSchema.Properties.All() {
				if propName == propertyName {
					return true
				}
			}
		}
	}

	return false
}

func (r *OASSchemaCheckRule) validateConst(_ context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, schemaTypes []oas3.SchemaType, _ *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error
	coreSchema := schema.GetCore()

	if !coreSchema.Const.Present || schema.Const == nil {
		return errs
	}

	constNode := coreSchema.Const.ValueNode
	if constNode == nil {
		return errs
	}

	// Check if const value matches any of the declared types
	isValid := false
	for _, schemaType := range schemaTypes {
		if r.isConstNodeValidForType(constNode, string(schemaType)) {
			isValid = true
			break
		}
	}

	if !isValid {
		// Convert SchemaType slice to string slice for Join with backticks
		typeStrs := make([]string, len(schemaTypes))
		for i, t := range schemaTypes {
			typeStrs[i] = "`" + string(t) + "`"
		}
		typeList := fmt.Sprintf("[%s]", strings.Join(typeStrs, ", "))
		if rootNode := refSchema.GetRootNode(); rootNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				r.ID(),
				fmt.Errorf("`const` value type does not match schema type %s", typeList),
				rootNode,
			))
		}
	}

	return errs
}

func (r *OASSchemaCheckRule) validateEnumConst(_ context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, _ *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	enumValues := schema.Enum
	constValue := schema.Const

	if len(enumValues) == 0 || constValue == nil {
		return errs
	}

	// Check if const value exists in enum values by comparing the YAML nodes
	constInEnum := false
	for _, enumValue := range enumValues {
		// Compare YAML node values and tags
		if constValue.Value == enumValue.Value && constValue.Tag == enumValue.Tag {
			constInEnum = true
			break
		}
	}

	if !constInEnum {
		if rootNode := refSchema.GetRootNode(); rootNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				r.ID(),
				fmt.Errorf("`const` value `%v` is not present in `enum` values", constValue),
				rootNode,
			))
		}
	} else {
		// Both enum and const are present and compatible - flag as potentially redundant
		if len(enumValues) == 1 {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("schema uses both `enum` with single value and `const` - consider using only `const`"),
					rootNode,
				))
			}
		} else {
			if rootNode := refSchema.GetRootNode(); rootNode != nil {
				errs = append(errs, validation.NewValidationError(
					config.GetSeverity(r.DefaultSeverity()),
					r.ID(),
					errors.New("schema uses both `enum` and `const` - this is likely an oversight as `const` restricts to a single value"),
					rootNode,
				))
			}
		}
	}

	return errs
}

func (r *OASSchemaCheckRule) isConstNodeValidForType(node *yaml.Node, schemaType string) bool {
	switch schemaType {
	case "string":
		return node.Tag == "!!str"
	case "integer":
		if node.Tag == "!!int" {
			return true
		}
		// Allow float values that have no fractional part (e.g., 42.0)
		if node.Tag == "!!float" {
			return r.isFloatWhole(node.Value)
		}
		return false
	case "number":
		return node.Tag == "!!int" || node.Tag == "!!float"
	case "boolean":
		return node.Tag == "!!bool"
	case "null":
		return node.Tag == "!!null"
	case "array":
		return node.Kind == yaml.SequenceNode
	case "object":
		return node.Kind == yaml.MappingNode
	}
	return false
}

func (r *OASSchemaCheckRule) isFloatWhole(value string) bool {
	// Check if a float string represents a whole number (e.g., "42.0" -> true, "42.5" -> false)
	if !strings.Contains(value, ".") {
		return true
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	// Check if fractional part is all zeros
	fractional := parts[1]
	for _, char := range fractional {
		if char != '0' {
			return false
		}
	}
	return true
}

func (r *OASSchemaCheckRule) validateDiscriminator(_ context.Context, schema *oas3.Schema, refSchema *oas3.JSONSchemaReferenceable, _ *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	var errs []error

	discriminator := schema.Discriminator
	if discriminator == nil {
		return errs
	}

	propertyName := discriminator.PropertyName

	// propertyName is required per OpenAPI 3.x spec
	if propertyName == "" {
		if rootNode := refSchema.GetRootNode(); rootNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				r.ID(),
				errors.New("discriminator object is missing required `propertyName` field"),
				rootNode,
			))
		}
		return errs
	}

	// Check if property exists in direct properties
	propertyExists := false
	if properties := schema.Properties; properties != nil {
		for propName := range properties.All() {
			if propName == propertyName {
				propertyExists = true
				break
			}
		}
	}

	// Check polymorphic schemas if not found
	if !propertyExists {
		propertyExists = r.checkPolymorphicProperty(schema, propertyName)
	}

	if !propertyExists {
		if rootNode := refSchema.GetRootNode(); rootNode != nil {
			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				r.ID(),
				fmt.Errorf("discriminator property `%s` is not defined in schema properties", propertyName),
				rootNode,
			))
		}
	}

	return errs
}
