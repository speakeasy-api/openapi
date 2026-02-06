package converter

import (
	"fmt"
	"strings"
	"unicode"
)

// GenerateRuleTypeScript generates a self-contained TypeScript rule file
// from an IR Rule. Returns the TypeScript source code.
func GenerateRuleTypeScript(rule Rule, rulePrefix string) (string, []Warning) {
	var warnings []Warning
	var b strings.Builder

	ruleID := rulePrefix + rule.ID
	className := toClassName(rule.ID)
	severity := mapSeverityToTSSeverity(rule.Severity)

	// Header comment
	fmt.Fprintf(&b, "// Auto-generated from Spectral rule: %s\n", rule.ID)
	if rule.Description != "" {
		fmt.Fprintf(&b, "// Original description: %s\n", rule.Description)
	}

	// Imports
	b.WriteString(`import {
  Rule,
  registerRule,
  createValidationError,
} from '@speakeasy-api/openapi-linter-types';
import type {
  Context,
  DocumentInfo,
  RuleConfig,
  ValidationError,
} from '@speakeasy-api/openapi-linter-types';
`)
	b.WriteString("\n")

	// Class definition
	fmt.Fprintf(&b, "class %s extends Rule {\n", className)
	fmt.Fprintf(&b, "  id(): string { return '%s'; }\n", escapeTS(ruleID))
	fmt.Fprintf(&b, "  category(): string { return '%s'; }\n", inferCategory(rule))
	fmt.Fprintf(&b, "  description(): string { return '%s'; }\n", escapeTS(rule.Description))
	fmt.Fprintf(&b, "  summary(): string { return '%s'; }\n", escapeTS(summaryFromDesc(rule.Description)))
	fmt.Fprintf(&b, "  defaultSeverity(): 'error' | 'warning' | 'hint' { return '%s'; }\n", severity)

	// Versions from formats
	if len(rule.Formats) > 0 {
		versions := formatsToVersions(rule.Formats)
		fmt.Fprintf(&b, "  versions(): string[] { return [%s]; }\n", joinQuoted(versions))
	}

	b.WriteString("\n")

	// Run method
	b.WriteString("  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {\n")
	b.WriteString("    const errors: ValidationError[] = [];\n")

	// Generate body from given/then
	bodyWarnings := generateBody(&b, rule)
	warnings = append(warnings, bodyWarnings...)

	b.WriteString("    return errors;\n")
	b.WriteString("  }\n")
	b.WriteString("}\n\n")

	// Registration
	fmt.Fprintf(&b, "registerRule(new %s());\n", className)

	return b.String(), warnings
}

// generateBody generates the rule body from given paths and then checks.
func generateBody(b *strings.Builder, rule Rule) []Warning {
	var warnings []Warning

	for _, givenPath := range rule.Given {
		mapping := MapJSONPath(givenPath)

		if mapping.Unsupported {
			// Generate validation error for unsupported pattern
			warnings = append(warnings, Warning{
				RuleID:  rule.ID,
				Phase:   "generate",
				Message: "unsupported JSONPath: " + givenPath,
			})
			fmt.Fprintf(b, "    // TODO: Unsupported JSONPath: %q\n", givenPath)
			b.WriteString("    // This rule could not be fully converted. Implement the JSONPath traversal manually.\n")
			b.WriteString("    errors.push(createValidationError(\n")
			b.WriteString("      config.getSeverity(this.defaultSeverity()),\n")
			b.WriteString("      this.id(),\n")
			fmt.Fprintf(b, "      'Rule not fully converted: unsupported JSONPath %q — implement manually',\n", escapeTS(givenPath))
			b.WriteString("      docInfo.document.getRootNode()\n")
			b.WriteString("    ));\n")
			continue
		}

		if mapping.IsDirect {
			generateDirectAccess(b, rule, mapping)
		} else {
			generateCollectionAccess(b, rule, mapping)
		}
	}

	return warnings
}

// generateDirectAccess generates code for direct document access ($.info, $.components, etc.)
func generateDirectAccess(b *strings.Builder, rule Rule, mapping JSONPathMapping) {
	accessor := mapping.DirectAccess
	fmt.Fprintf(b, "    const target = %s;\n", accessor)
	b.WriteString("    if (target) {\n")

	if mapping.FieldAccess != "" {
		fmt.Fprintf(b, "      %s\n", generateFieldAccess("target", mapping.FieldAccess, "value"))
		for _, check := range rule.Then {
			if check.Field != "" && check.Field != mapping.FieldAccess {
				// Then has a different field — access it from the target
				generateCheckCode(b, rule, check, "target", "      ")
			} else {
				generateFunctionCheck(b, rule, check, "value", "target", "      ")
			}
		}
	} else {
		for _, check := range rule.Then {
			generateCheckCode(b, rule, check, "target", "      ")
		}
	}

	b.WriteString("    }\n")
}

// generateCollectionAccess generates code for indexed collection iteration.
func generateCollectionAccess(b *strings.Builder, rule Rule, mapping JSONPathMapping) {
	collection := mapping.Collection

	fmt.Fprintf(b, "    if (docInfo.index && docInfo.index.%s) {\n", collection)
	fmt.Fprintf(b, "      for (const indexNode of docInfo.index.%s) {\n", collection)
	b.WriteString("        const node = indexNode.node;\n")

	// HTTP method filter
	if mapping.HTTPMethod != "" {
		b.WriteString("        const loc = indexNode.location;\n")
		fmt.Fprintf(b, "        if (loc && loc.length > 0 && loc[loc.length - 1].parentKey() !== '%s') continue;\n", mapping.HTTPMethod)
	}

	// Key access via ~ operator
	switch {
	case mapping.IsKeyAccess:
		b.WriteString("        const loc = indexNode.location;\n")
		b.WriteString("        const key = loc && loc.length > 0 ? loc[loc.length - 1].parentKey() : '';\n")
		for _, check := range rule.Then {
			generateFunctionCheck(b, rule, check, "key", "node", "        ")
		}
	case mapping.FieldAccess != "":
		// Field access (e.g., $.servers[*].url -> node.url)
		fmt.Fprintf(b, "        %s\n", generateFieldAccess("node", mapping.FieldAccess, "value"))
		for _, check := range rule.Then {
			if check.Field != "" && check.Field != mapping.FieldAccess {
				generateCheckCode(b, rule, check, "node", "        ")
			} else {
				generateFunctionCheck(b, rule, check, "value", "node", "        ")
			}
		}
	default:
		for _, check := range rule.Then {
			generateCheckCode(b, rule, check, "node", "        ")
		}
	}

	b.WriteString("      }\n")
	b.WriteString("    }\n")
}

// generateCheckCode generates code for a single RuleCheck against a node variable.
// When the check accesses a field, it wraps the code in a block scope to avoid
// duplicate variable declarations when multiple checks target different fields.
func generateCheckCode(b *strings.Builder, rule Rule, check RuleCheck, nodeVar, indent string) {
	if check.Field != "" {
		// Wrap in block scope to allow multiple field checks with same variable name
		b.WriteString(indent + "{\n")
		fieldVar := "fieldValue"
		fmt.Fprintf(b, "%s  %s\n", indent, generateFieldAccess(nodeVar, check.Field, fieldVar))
		generateFunctionCheck(b, rule, check, fieldVar, nodeVar, indent+"  ")
		b.WriteString(indent + "}\n")
	} else {
		generateFunctionCheck(b, rule, check, nodeVar, nodeVar, indent)
	}
}

// generateFunctionCheck generates the function application code.
func generateFunctionCheck(b *strings.Builder, rule Rule, check RuleCheck, valueVar, nodeVar, indent string) {
	msg := buildMessage(rule, check)

	switch check.Function {
	case "truthy":
		fmt.Fprintf(b, "%sif (!%s) {\n", indent, valueVar)
		fmt.Fprintf(b, "%s  errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
		b.WriteString(indent + "}\n")

	case "falsy":
		fmt.Fprintf(b, "%sif (%s) {\n", indent, valueVar)
		fmt.Fprintf(b, "%s  errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
		b.WriteString(indent + "}\n")

	case "defined":
		fmt.Fprintf(b, "%sif (%s === undefined || %s === null) {\n", indent, valueVar, valueVar)
		fmt.Fprintf(b, "%s  errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
		b.WriteString(indent + "}\n")

	case "undefined":
		fmt.Fprintf(b, "%sif (%s !== undefined && %s !== null) {\n", indent, valueVar, valueVar)
		fmt.Fprintf(b, "%s  errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
		b.WriteString(indent + "}\n")

	case "pattern":
		generatePatternCheck(b, rule, check, valueVar, nodeVar, indent, msg)

	case "enumeration":
		generateEnumerationCheck(b, check, valueVar, nodeVar, indent, msg)

	case "length":
		generateLengthCheck(b, check, valueVar, nodeVar, indent, msg)

	case "casing":
		generateCasingCheck(b, check, valueVar, nodeVar, indent, msg)

	case "alphabetical":
		generateAlphabeticalCheck(b, check, valueVar, nodeVar, indent, msg)

	case "xor":
		generateXorCheck(b, check, nodeVar, indent, msg)

	case "or":
		generateOrCheck(b, check, nodeVar, indent, msg)

	case "schema", "typedEnum", "unreferencedReusableObject":
		// These require capabilities not available in the goja runtime
		fmt.Fprintf(b, "%s// TODO: Function %q requires capabilities not available in the custom rules runtime.\n", indent, check.Function)
		switch check.Function {
		case "typedEnum":
			b.WriteString(indent + "// Consider using the native rule 'semantic-typed-enum' instead.\n")
		case "unreferencedReusableObject":
			b.WriteString(indent + "// Consider using the native rule 'semantic-unused-component' instead.\n")
		}
		fmt.Fprintf(b, "%serrors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Rule not fully converted: function %q — implement manually', docInfo.document.getRootNode()));\n", indent, escapeTS(check.Function))

	default:
		// Unknown/custom function
		fmt.Fprintf(b, "%s// TODO: Custom Spectral function %q — implement manually\n", indent, check.Function)
		fmt.Fprintf(b, "%serrors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Rule not fully converted: unsupported function %q — implement manually', docInfo.document.getRootNode()));\n", indent, escapeTS(check.Function))
	}
}

func generatePatternCheck(b *strings.Builder, _ Rule, check RuleCheck, valueVar, nodeVar, indent, msg string) {
	match, notMatch := PatternOptions(check)

	if match != "" {
		b.WriteString(indent + "{\n")
		b.WriteString(indent + "  let re: RegExp | null = null;\n")
		b.WriteString(indent + "  try {\n")
		fmt.Fprintf(b, "%s    re = new RegExp('%s');\n", indent, escapeTS(match))
		b.WriteString(indent + "  } catch (e) {\n")
		b.WriteString(indent + "    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), `Invalid regex pattern: ${e}`, docInfo.document.getRootNode()));\n")
		b.WriteString(indent + "  }\n")
		fmt.Fprintf(b, "%s  if (re && !re.test(String(%s || ''))) {\n", indent, valueVar)
		fmt.Fprintf(b, "%s    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
		b.WriteString(indent + "  }\n")
		b.WriteString(indent + "}\n")
	}

	if notMatch != "" {
		b.WriteString(indent + "{\n")
		b.WriteString(indent + "  let re: RegExp | null = null;\n")
		b.WriteString(indent + "  try {\n")
		fmt.Fprintf(b, "%s    re = new RegExp('%s');\n", indent, escapeTS(notMatch))
		b.WriteString(indent + "  } catch (e) {\n")
		b.WriteString(indent + "    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), `Invalid regex pattern: ${e}`, docInfo.document.getRootNode()));\n")
		b.WriteString(indent + "  }\n")
		fmt.Fprintf(b, "%s  if (re && re.test(String(%s || ''))) {\n", indent, valueVar)
		fmt.Fprintf(b, "%s    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
		b.WriteString(indent + "  }\n")
		b.WriteString(indent + "}\n")
	}
}

func generateEnumerationCheck(b *strings.Builder, check RuleCheck, valueVar, nodeVar, indent, msg string) {
	values := EnumerationOptions(check)
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, "'"+escapeTS(v)+"'")
	}
	b.WriteString(indent + "{\n")
	fmt.Fprintf(b, "%s  const allowed = [%s];\n", indent, strings.Join(quoted, ", "))
	fmt.Fprintf(b, "%s  if (!allowed.includes(String(%s))) {\n", indent, valueVar)
	fmt.Fprintf(b, "%s    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
	b.WriteString(indent + "  }\n")
	b.WriteString(indent + "}\n")
}

func generateLengthCheck(b *strings.Builder, check RuleCheck, valueVar, nodeVar, indent, msg string) {
	minVal, maxVal := LengthOptions(check)
	b.WriteString(indent + "{\n")
	fmt.Fprintf(b, "%s  const sLen = typeof %s === 'string' || Array.isArray(%s) ? %s.length : (typeof %s === 'number' ? %s : 0);\n", indent, valueVar, valueVar, valueVar, valueVar, valueVar)
	if minVal != nil {
		fmt.Fprintf(b, "%s  if (sLen < %d) {\n", indent, *minVal)
		fmt.Fprintf(b, "%s    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
		b.WriteString(indent + "  }\n")
	}
	if maxVal != nil {
		fmt.Fprintf(b, "%s  if (sLen > %d) {\n", indent, *maxVal)
		fmt.Fprintf(b, "%s    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
		b.WriteString(indent + "  }\n")
	}
	b.WriteString(indent + "}\n")
}

func generateCasingCheck(b *strings.Builder, check RuleCheck, valueVar, nodeVar, indent, msg string) {
	caseType := CasingOptions(check)
	pattern := casingRegex(caseType)
	b.WriteString(indent + "{\n")
	fmt.Fprintf(b, "%s  const casingRe = %s;\n", indent, pattern)
	fmt.Fprintf(b, "%s  if (typeof %s === 'string' && %s !== '' && !casingRe.test(%s)) {\n", indent, valueVar, valueVar, valueVar)
	fmt.Fprintf(b, "%s    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
	b.WriteString(indent + "  }\n")
	b.WriteString(indent + "}\n")
}

func generateAlphabeticalCheck(b *strings.Builder, check RuleCheck, valueVar, nodeVar, indent, msg string) {
	b.WriteString(indent + "{\n")
	keyedBy := ""
	if check.FunctionOptions != nil {
		if v, ok := check.FunctionOptions["keyedBy"].(string); ok {
			keyedBy = v
		}
	}
	fmt.Fprintf(b, "%s  const items = Array.isArray(%s) ? %s : [];\n", indent, valueVar, valueVar)
	b.WriteString(indent + "  for (let i = 1; i < items.length; i++) {\n")
	if keyedBy != "" {
		fmt.Fprintf(b, "%s    const prev = items[i - 1]?.%s ?? '';\n", indent, keyedBy)
		fmt.Fprintf(b, "%s    const curr = items[i]?.%s ?? '';\n", indent, keyedBy)
	} else {
		b.WriteString(indent + "    const prev = String(items[i - 1] ?? '');\n")
		b.WriteString(indent + "    const curr = String(items[i] ?? '');\n")
	}
	b.WriteString(indent + "    if (prev.localeCompare(curr) > 0) {\n")
	fmt.Fprintf(b, "%s      errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
	b.WriteString(indent + "      break;\n")
	b.WriteString(indent + "    }\n")
	b.WriteString(indent + "  }\n")
	b.WriteString(indent + "}\n")
}

func generateXorCheck(b *strings.Builder, check RuleCheck, nodeVar, indent, msg string) {
	props := PropertyOptions(check)
	b.WriteString(indent + "{\n")
	fmt.Fprintf(b, "%s  const props = [%s];\n", indent, joinQuoted(props))
	b.WriteString(indent + "  const present = props.filter(p => {\n")
	fmt.Fprintf(b, "%s    const v = (%s as any)[p];\n", indent, nodeVar)
	b.WriteString(indent + "    return v !== undefined && v !== null;\n")
	b.WriteString(indent + "  });\n")
	b.WriteString(indent + "  if (present.length !== 1) {\n")
	fmt.Fprintf(b, "%s    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
	b.WriteString(indent + "  }\n")
	b.WriteString(indent + "}\n")
}

func generateOrCheck(b *strings.Builder, check RuleCheck, nodeVar, indent, msg string) {
	props := PropertyOptions(check)
	b.WriteString(indent + "{\n")
	fmt.Fprintf(b, "%s  const props = [%s];\n", indent, joinQuoted(props))
	b.WriteString(indent + "  const present = props.filter(p => {\n")
	fmt.Fprintf(b, "%s    const v = (%s as any)[p];\n", indent, nodeVar)
	b.WriteString(indent + "    return v !== undefined && v !== null;\n")
	b.WriteString(indent + "  });\n")
	b.WriteString(indent + "  if (present.length === 0) {\n")
	fmt.Fprintf(b, "%s    errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), %s, %s.getRootNode ? %s.getRootNode() : null));\n", indent, msg, nodeVar, nodeVar)
	b.WriteString(indent + "  }\n")
	b.WriteString(indent + "}\n")
}

// --- Field access helpers ---

// knownGetters maps field names to their TypeScript getter method names.
// Built from the document.ts type definitions.
var knownGetters = map[string]string{
	"description":          "getDescription",
	"summary":              "getSummary",
	"operationId":          "getOperationID",
	"operationID":          "getOperationID",
	"tags":                 "getTags",
	"parameters":           "getParameters",
	"requestBody":          "getRequestBody",
	"responses":            "getResponses",
	"security":             "getSecurity",
	"servers":              "getServers",
	"deprecated":           "getDeprecated",
	"required":             "getRequired",
	"schema":               "getSchema",
	"contact":              "getContact",
	"license":              "getLicense",
	"termsOfService":       "getTermsOfService",
	"content":              "getContent",
	"headers":              "getHeaders",
	"links":                "getLinks",
	"examples":             "getExamples",
	"encoding":             "getEncoding",
	"scopes":               "getScopes",
	"schemes":              "getSecuritySchemes",
	"properties":           "getProperties",
	"items":                "getItems",
	"additionalProperties": "getAdditionalProperties",
	"allOf":                "getAllOf",
	"anyOf":                "getAnyOf",
	"oneOf":                "getOneOf",
	"not":                  "getNot",
	"type":                 "getType",
	"format":               "getFormat",
	"pattern":              "getPattern",
	"enum":                 "getEnum",
	"default":              "getDefault",
	"minimum":              "getMinimum",
	"maximum":              "getMaximum",
	"minLength":            "getMinLength",
	"maxLength":            "getMaxLength",
	"minItems":             "getMinItems",
	"maxItems":             "getMaxItems",
	"nullable":             "getNullable",
	"externalDocs":         "getExternalDocs",
	"variables":            "getVariables",
	"flows":                "getFlows",
	"bearerFormat":         "getBearerFormat",
	"openIdConnectUrl":     "getOpenIdConnectUrl",
}

// generateFieldAccess generates code to access a field on a node.
// Returns a TypeScript statement like "const value = node.getSummary();"
func generateFieldAccess(nodeVar, field, resultVar string) string {
	// Handle @key special field — only valid inside collection loops where indexNode exists.
	// In direct-access contexts, fall through to the warning path.
	if field == "@key" && nodeVar == "node" {
		return fmt.Sprintf("const %s = indexNode.location && indexNode.location.length > 0 ? indexNode.location[indexNode.location.length - 1].parentKey() : '';", resultVar)
	}

	// Check for known getter
	if getter, ok := knownGetters[field]; ok {
		return fmt.Sprintf("const %s = %s.%s ? %s.%s() : undefined;", resultVar, nodeVar, getter, nodeVar, getter)
	}

	// Handle dotted field paths like "headers.X-Request-ID"
	if idx := strings.IndexByte(field, '.'); idx != -1 {
		firstField := field[:idx]
		rest := field[idx+1:]
		if getter, ok := knownGetters[firstField]; ok {
			return fmt.Sprintf("const %s = %s.%s ? (%s.%s() as any)?.[%q] : undefined;", resultVar, nodeVar, getter, nodeVar, getter, rest)
		}
		// No known getter for first part either
		return fmt.Sprintf("// WARNING: No known getter for field '%s' — using dynamic access\nconst %s = (%s as any)?.[%q]?.[%q];", field, resultVar, nodeVar, firstField, rest)
	}

	// Fallback: dynamic access with warning
	return fmt.Sprintf("// WARNING: No known getter for field '%s' — using dynamic access\nif (!('%s' in (%s as any))) { /* field may not exist */ }\nconst %s = (%s as any)['%s'];", field, escapeTS(field), nodeVar, resultVar, nodeVar, escapeTS(field))
}

// --- Utility functions ---

// toClassName converts a kebab-case rule ID to PascalCase class name.
func toClassName(id string) string {
	var b strings.Builder
	capitalize := true
	for _, ch := range id {
		if ch == '-' || ch == '_' || ch == '.' {
			capitalize = true
			continue
		}
		if capitalize {
			b.WriteRune(unicode.ToUpper(ch))
			capitalize = false
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// escapeTS escapes a string for use in TypeScript string literals.
// Handles single quotes (primary usage) and backticks (template literals in regex error paths).
func escapeTS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// mapSeverityToTSSeverity converts IR severity to TypeScript rule severity.
func mapSeverityToTSSeverity(irSeverity string) string {
	switch irSeverity {
	case "error":
		return "error"
	case "warn", "warning":
		return "warning"
	case "info", "hint":
		return "hint"
	default:
		return "warning"
	}
}

// inferCategory infers a rule category from the rule content.
// This is minimal — Spectral doesn't have a category concept, so most converted
// rules will be "style". Users can change categories in the generated .ts files.
func inferCategory(rule Rule) string {
	desc := strings.ToLower(rule.Description)
	if strings.Contains(desc, "security") || strings.Contains(desc, "auth") {
		return "security"
	}
	return "style"
}

// summaryFromDesc creates a short summary from a description.
func summaryFromDesc(desc string) string {
	if desc == "" {
		return ""
	}
	// Take first sentence or max 80 chars
	if idx := strings.IndexAny(desc, ".!"); idx != -1 && idx < 80 {
		return desc[:idx+1]
	}
	if len(desc) > 80 {
		return desc[:77] + "..."
	}
	return desc
}

// formatsToVersions maps Spectral format names to OpenAPI version strings.
func formatsToVersions(formats []string) []string {
	var versions []string
	seen := make(map[string]bool)
	for _, f := range formats {
		switch f {
		case "oas2":
			if !seen["2.0"] {
				versions = append(versions, "2.0")
				seen["2.0"] = true
			}
		case "oas3":
			for _, v := range []string{"3.0", "3.1"} {
				if !seen[v] {
					versions = append(versions, v)
					seen[v] = true
				}
			}
		case "oas3.0":
			if !seen["3.0"] {
				versions = append(versions, "3.0")
				seen["3.0"] = true
			}
		case "oas3.1":
			if !seen["3.1"] {
				versions = append(versions, "3.1")
				seen["3.1"] = true
			}
		}
	}
	return versions
}

// buildMessage constructs a TypeScript error message expression from rule/check.
// Spectral templates use {{property}}, {{value}}, {{path}}, etc. which are
// runtime context variables. Since our generated code doesn't have those in scope,
// we replace them with static strings derived from the rule/check context.
func buildMessage(rule Rule, check RuleCheck) string {
	msg := check.Message
	if msg == "" {
		msg = rule.Message
	}
	if msg == "" {
		msg = rule.Description
		if msg == "" {
			msg = "Rule violation"
		}
	}
	// Replace Spectral {{placeholders}} with static values
	if strings.Contains(msg, "{{") {
		msg = expandMessageTemplate(msg, rule, check)
	}
	return "'" + escapeTS(msg) + "'"
}

// expandMessageTemplate replaces Spectral {{placeholders}} with static strings.
// Spectral provides these as runtime context, but we inline them as static values
// to avoid undefined variable references in generated TypeScript.
func expandMessageTemplate(msg string, rule Rule, check RuleCheck) string {
	property := check.Field
	if property == "" {
		property = "property"
	}
	description := rule.Description
	if description == "" {
		description = "description"
	}
	result := msg
	result = strings.ReplaceAll(result, "{{property}}", property)
	result = strings.ReplaceAll(result, "{{description}}", description)
	result = strings.ReplaceAll(result, "{{error}}", "error")
	result = strings.ReplaceAll(result, "{{value}}", "value")
	result = strings.ReplaceAll(result, "{{path}}", "path")
	return result
}

// casingRegex returns a JavaScript regex literal for a casing type.
func casingRegex(caseType string) string {
	switch caseType {
	case "camelCase":
		return `/^[a-z][a-zA-Z0-9]*$/`
	case "PascalCase":
		return `/^[A-Z][a-zA-Z0-9]*$/`
	case "kebab-case":
		return `/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/`
	case "snake_case":
		return `/^[a-z][a-z0-9]*(_[a-z0-9]+)*$/`
	case "SCREAMING_SNAKE_CASE", "macro_case":
		return `/^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$/`
	case "cobol-case", "COBOL-CASE":
		return `/^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$/`
	case "flatcase":
		return `/^[a-z][a-z0-9]*$/`
	default:
		return `/^.+$/` // permissive fallback
	}
}

// joinQuoted creates a comma-separated list of single-quoted strings.
func joinQuoted(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, "'"+escapeTS(v)+"'")
	}
	return strings.Join(quoted, ", ")
}
