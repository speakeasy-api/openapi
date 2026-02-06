package converter

// IntermediateConfig is the format-agnostic intermediate representation.
// Parse Spectral/Vacuum/Legacy configs into this, then pass to Generate()
// or consume directly in other projects.
type IntermediateConfig struct {
	// Extends stores the raw extends values from the source config.
	// Examples: ["spectral:oas"], ["speakeasy-recommended"]
	// Interpretation of these values is left to the consumer (Generate maps them).
	Extends []ExtendsEntry

	// Rules is the unified list of all rules from the source config.
	// Each rule is either a severity-only override or a full rule definition.
	// Parse does NOT classify rules as "builtin" vs "custom" — that's a
	// consumer concern. Rules with Given/Then populated are custom rules;
	// rules with only Severity set are overrides.
	Rules []Rule

	// Warnings collected during parsing (malformed entries, etc.)
	Warnings []Warning
}

// ExtendsEntry stores the raw extends value from the source config.
// Known Name values by format:
//
//	Spectral: "spectral:oas", "spectral:asyncapi"
//	Legacy:   "speakeasy-recommended", "speakeasy-generation"
//
// Known Modifier values (Spectral only, empty for Legacy):
//
//	"all"         - enable all rules from the extended ruleset
//	"recommended" - enable only recommended rules
//	"off"         - disable the extended ruleset entirely
type ExtendsEntry struct {
	Name     string // e.g., "spectral:oas", "speakeasy-recommended"
	Modifier string // e.g., "all", "recommended", "off" (empty if not specified)
}

// Rule is a unified representation of any rule from a source config.
// Severity-only overrides have Given/Then empty.
// Full custom rules have Given/Then populated.
//
// Severity is normalized during parsing to canonical values:
//
//	"error", "warn", "info", "hint", "off"
//
// Numeric severities from Spectral (0=error, 1=warn, 2=info, 3=hint)
// are normalized to their string equivalents.
type Rule struct {
	ID          string      // rule name/ID from source config
	Description string      // rule description
	Message     string      // message template with {{placeholders}}
	Severity    string      // normalized severity (see doc comment above)
	Resolved    *bool       // nil = default (resolved), false = unresolved
	Formats     []string    // "oas2", "oas3", "oas3.0", "oas3.1"
	Source      string      // ruleset name this rule came from (for dedup tracing)
	Given       []string    // JSONPath expressions (empty for overrides)
	Then        []RuleCheck // checks to apply (empty for overrides)
}

// IsOverride returns true if this rule is a severity-only override
// (no given/then logic).
func (r *Rule) IsOverride() bool { return len(r.Given) == 0 && len(r.Then) == 0 }

// IsDisabled returns true if severity is "off".
func (r *Rule) IsDisabled() bool { return r.Severity == "off" }

// RuleCheck is a single function application from a "then" clause.
// FunctionOptions shape depends on Function:
//   - "pattern":     { "match": string, "notMatch": string }
//   - "enumeration": { "values": []string }
//   - "length":      { "min": number, "max": number }
//   - "casing":      { "type": string, "disallowDigits": bool, "separator": { "char": string } }
//   - "alphabetical":{ "keyedBy": string }
//   - "schema":      { "schema": object, "dialect": string, "allErrors": bool }
//   - "xor"/"or":    { "properties": []string }
//   - "truthy"/"falsy"/"defined"/"undefined"/"typedEnum": (no options)
type RuleCheck struct {
	Field           string         // field to check (empty = check the node itself)
	Function        string         // "truthy", "pattern", "enumeration", etc.
	FunctionOptions map[string]any // function-specific options (documented above)
	Message         string         // per-check message override
}

// Warning represents a warning generated during parsing or generation.
type Warning struct {
	RuleID  string // the rule this warning relates to (empty for global warnings)
	Phase   string // "parse" or "generate"
	Message string
}

// PatternOptions extracts match/notMatch from a RuleCheck's FunctionOptions.
func PatternOptions(rc RuleCheck) (match, notMatch string) {
	if rc.FunctionOptions == nil {
		return "", ""
	}
	if v, ok := rc.FunctionOptions["match"].(string); ok {
		match = v
	}
	if v, ok := rc.FunctionOptions["notMatch"].(string); ok {
		notMatch = v
	}
	return match, notMatch
}

// EnumerationOptions extracts allowed values from a RuleCheck's FunctionOptions.
func EnumerationOptions(rc RuleCheck) []string {
	if rc.FunctionOptions == nil {
		return nil
	}
	raw, ok := rc.FunctionOptions["values"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// LengthOptions extracts min/max from a RuleCheck's FunctionOptions.
func LengthOptions(rc RuleCheck) (minVal, maxVal *int) {
	if rc.FunctionOptions == nil {
		return nil, nil
	}
	if v, ok := toInt(rc.FunctionOptions["min"]); ok {
		minVal = &v
	}
	if v, ok := toInt(rc.FunctionOptions["max"]); ok {
		maxVal = &v
	}
	return minVal, maxVal
}

// CasingOptions extracts the case type from a RuleCheck's FunctionOptions.
func CasingOptions(rc RuleCheck) string {
	if rc.FunctionOptions == nil {
		return ""
	}
	if v, ok := rc.FunctionOptions["type"].(string); ok {
		return v
	}
	return ""
}

// PropertyOptions extracts properties from xor/or RuleCheck's FunctionOptions.
func PropertyOptions(rc RuleCheck) []string {
	if rc.FunctionOptions == nil {
		return nil
	}
	raw, ok := rc.FunctionOptions["properties"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// toInt converts various numeric types to int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
