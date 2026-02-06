package converter

// spectralToNativeMapping maps Spectral/Vacuum built-in rule names to native rule IDs.
// Used by Generate, not by Parse.
//
// Maintenance note: This table is hardcoded. When native rules are added/renamed,
// update this table. Native rule IDs are defined as constants in
// openapi/linter/rules/ (e.g., RuleStyleOperationTags = "style-operation-tags").
var spectralToNativeMapping = map[string]string{
	// Operation rules
	"operation-tags":                     "style-operation-tags",
	"operation-tag-defined":              "style-operation-tag-defined",
	"operation-operationId":              "semantic-operation-operation-id",
	"operation-operationId-unique":       "semantic-operation-operation-id",
	"operation-operationId-valid-in-url": "semantic-operation-id-valid-in-url",
	"operation-description":              "style-operation-description",
	"operation-singular-tag":             "style-operation-singular-tag",
	"operation-success-response":         "style-operation-success-response",

	// Markdown rules
	"no-eval-in-markdown":        "semantic-no-eval-in-markdown",
	"no-script-tags-in-markdown": "semantic-no-script-tags-in-markdown",

	// Ref rules
	"no-$ref-siblings": "style-no-ref-siblings",
	"no-ref-siblings":  "style-no-ref-siblings",

	// Schema/enum rules
	"typed-enum":               "semantic-typed-enum",
	"duplicated-entry-in-enum": "semantic-duplicated-enum",

	// Path rules
	"no-ambiguous-paths":           "semantic-no-ambiguous-paths",
	"path-params":                  "semantic-path-params",
	"path-declarations-must-exist": "semantic-path-declarations",
	"path-keys-no-trailing-slash":  "style-path-trailing-slash",
	"path-not-include-query":       "semantic-path-query",

	// Info rules
	"info-description":   "style-info-description",
	"info-contact":       "style-info-contact",
	"info-license":       "style-info-license",
	"license-url":        "style-license-url",
	"contact-properties": "style-contact-properties",

	// Tag rules
	"tag-description":   "style-tag-description",
	"tags-alphabetical": "style-tags-alphabetical",

	// OAS3-specific rules
	"oas3-api-servers":            "style-oas3-api-servers",
	"oas3-server-not-example.com": "style-oas3-host-not-example",
	"oas3-server-trailing-slash":  "style-oas3-host-trailing-slash",
	"oas3-parameter-description":  "style-oas3-parameter-description",
	"oas3-no-nullable":            "oas3-no-nullable",
	"oas3-unused-component":       "semantic-unused-component",

	// General rules
	"openapi-tags":            "style-openapi-tags",
	"description-duplication": "style-description-duplication",
	"oas-schema-check":        "oas-schema-check",
}

// LookupNativeRule maps a Spectral/Vacuum rule name to a native rule ID.
// Returns the native ID and true if found, or empty string and false if not mapped.
func LookupNativeRule(spectralName string) (string, bool) {
	native, ok := spectralToNativeMapping[spectralName]
	return native, ok
}

// mapSeverityToNative converts IR severity to native validation.Severity string.
// IR uses "warn" while native uses "warning", IR uses "info" while native uses "hint".
func mapSeverityToNative(irSeverity string) string {
	switch irSeverity {
	case "error":
		return "error"
	case "warn", "warning":
		return "warning"
	case "info", "hint":
		return "hint"
	default:
		return "warning" // safe default
	}
}
