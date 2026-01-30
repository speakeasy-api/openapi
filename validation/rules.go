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
