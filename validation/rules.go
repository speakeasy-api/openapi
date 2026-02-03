package validation

const (
	// Spec Validation Rules
	RuleValidationRequiredField           = "validation-required-field"
	RuleValidationTypeMismatch            = "validation-type-mismatch"
	RuleValidationDuplicateKey            = "validation-duplicate-key"
	RuleValidationInvalidFormat           = "validation-invalid-format"
	RuleValidationEmptyValue              = "validation-empty-value"
	RuleValidationInvalidReference        = "validation-invalid-reference"
	RuleValidationInvalidSyntax           = "validation-invalid-syntax"
	RuleValidationInvalidSchema           = "validation-invalid-schema"
	RuleValidationInvalidTarget           = "validation-invalid-target"
	RuleValidationAllowedValues           = "validation-allowed-values"
	RuleValidationMutuallyExclusiveFields = "validation-mutually-exclusive-fields"
	RuleValidationOperationNotFound       = "validation-operation-not-found"
	RuleValidationOperationIdUnique       = "validation-operation-id-unique"
	RuleValidationOperationParameters     = "validation-operation-parameters"
	RuleValidationSchemeNotFound          = "validation-scheme-not-found"
	RuleValidationTagNotFound             = "validation-tag-not-found"
	RuleValidationSupportedVersion        = "validation-supported-version"
	RuleValidationCircularReference       = "validation-circular-reference"
)

type RuleInfo struct {
	Summary     string
	Description string
	HowToFix    string
}

var ruleInfoByID = map[string]RuleInfo{
	RuleValidationRequiredField: {
		Summary:     "Missing required field.",
		Description: "Required fields must be present in the document. Missing required fields cause validation to fail.",
		HowToFix:    "Provide the required field in the document.",
	},
	RuleValidationTypeMismatch: {
		Summary:     "Type mismatch.",
		Description: "Values must match the schema types defined in the specification. Mismatched types can break tooling and validation.",
		HowToFix:    "Update the value to match the schema type or adjust the schema.",
	},
	RuleValidationDuplicateKey: {
		Summary:     "Duplicate key.",
		Description: "Duplicate keys are not allowed in objects. Remove duplicates to avoid parsing ambiguity.",
		HowToFix:    "Remove or rename the duplicate key.",
	},
	RuleValidationInvalidFormat: {
		Summary:     "Invalid format.",
		Description: "Values must match the specified format. Invalid formats can lead to runtime or interoperability issues.",
		HowToFix:    "Use a value that conforms to the required format.",
	},
	RuleValidationEmptyValue: {
		Summary:     "Empty value.",
		Description: "Values must not be empty when the field requires content. Empty values typically indicate missing data.",
		HowToFix:    "Provide a non-empty value.",
	},
	RuleValidationInvalidReference: {
		Summary:     "Invalid reference.",
		Description: "References must resolve to existing components or locations. Broken references prevent correct validation and resolution.",
		HowToFix:    "Fix the $ref target or define the referenced component.",
	},
	RuleValidationInvalidSyntax: {
		Summary:     "Invalid syntax.",
		Description: "Documents must be valid YAML or JSON. Syntax errors prevent parsing.",
		HowToFix:    "Correct the syntax errors in the document.",
	},
	RuleValidationInvalidSchema: {
		Summary:     "Invalid schema.",
		Description: "Schemas must be valid according to the OpenAPI/JSON Schema rules. Invalid schemas can make the document unusable for tooling.",
		HowToFix:    "Correct schema keywords and values to match the specification.",
	},
	RuleValidationInvalidTarget: {
		Summary:     "Invalid target.",
		Description: "Validation targets must exist and be valid for the context. Invalid targets typically indicate a bad reference path.",
		HowToFix:    "Point to a valid target or adjust the reference context.",
	},
	RuleValidationAllowedValues: {
		Summary:     "Value not allowed.",
		Description: "Values must be one of the allowed values. Using disallowed values violates the specification.",
		HowToFix:    "Use a value from the allowed set.",
	},
	RuleValidationMutuallyExclusiveFields: {
		Summary:     "Mutually exclusive fields.",
		Description: "Mutually exclusive fields cannot be used together. Choose one of the conflicting fields.",
		HowToFix:    "Remove one of the conflicting fields.",
	},
	RuleValidationOperationNotFound: {
		Summary:     "Operation not found.",
		Description: "Referenced operations must exist in the specification. Missing operations indicate a broken link.",
		HowToFix:    "Add the operation or fix the reference.",
	},
	RuleValidationOperationIdUnique: {
		Summary:     "Duplicate operationId.",
		Description: "Operation IDs must be unique across the specification. Duplicate IDs cause conflicts in tooling.",
		HowToFix:    "Assign unique operationId values.",
	},
	RuleValidationOperationParameters: {
		Summary:     "Invalid operation parameters.",
		Description: "Operation parameters must be valid and correctly defined. Invalid parameters can break request handling.",
		HowToFix:    "Fix parameter definitions and resolve invalid references.",
	},
	RuleValidationSchemeNotFound: {
		Summary:     "Security scheme not found.",
		Description: "Referenced security schemes must be defined. Missing schemes make security requirements invalid.",
		HowToFix:    "Define the security scheme or fix the scheme reference.",
	},
	RuleValidationTagNotFound: {
		Summary:     "Tag not found.",
		Description: "Operation tags should be defined in the top-level tags array. Undefined tags make documentation inconsistent.",
		HowToFix:    "Add the tag to the top-level tags array or fix the tag name.",
	},
	RuleValidationSupportedVersion: {
		Summary:     "Unsupported OpenAPI version.",
		Description: "The document must use a supported OpenAPI version. Unsupported versions may not be parsed correctly.",
		HowToFix:    "Update the document to a supported OpenAPI version.",
	},
	RuleValidationCircularReference: {
		Summary:     "Circular reference.",
		Description: "Schemas must not contain circular references that cannot be resolved. Unresolvable cycles can break validation and tooling.",
		HowToFix:    "Refactor schemas to break the reference cycle.",
	},
}

func RuleInfoForID(ruleID string) (RuleInfo, bool) {
	info, ok := ruleInfoByID[ruleID]
	return info, ok
}

func RuleSummary(ruleID string) string {
	return ruleInfoByID[ruleID].Summary
}

func RuleDescription(ruleID string) string {
	return ruleInfoByID[ruleID].Description
}

func RuleHowToFix(ruleID string) string {
	return ruleInfoByID[ruleID].HowToFix
}
