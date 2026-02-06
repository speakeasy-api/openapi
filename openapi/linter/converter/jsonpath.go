package converter

import (
	"regexp"
	"strings"
)

// JSONPathMapping represents the result of mapping a JSONPath to an index collection
// and generated TypeScript code.
type JSONPathMapping struct {
	// Collection is the index collection name (e.g., "operations", "inlinePathItems").
	// Empty for direct document access patterns.
	Collection string

	// IsKeyAccess is true when the ~ operator is used (extract map key).
	IsKeyAccess bool

	// FieldAccess is a field to extract from each node (e.g., "url" for $.servers[*].url).
	FieldAccess string

	// Filter is a filter expression stripped during structural matching,
	// to be re-applied as generated TypeScript.
	Filter string

	// HTTPMethod is set when matching $.paths[*].{method} patterns.
	HTTPMethod string

	// IsDirect is true when the path targets a single document location
	// (e.g., $.info, $.components) rather than an indexed collection.
	IsDirect bool

	// DirectAccess is the TypeScript expression for direct access.
	DirectAccess string

	// Unsupported is true when the JSONPath could not be mapped.
	Unsupported bool

	// OriginalPath is the original JSONPath expression.
	OriginalPath string
}

// MapJSONPath maps a JSONPath expression to an index collection and access pattern.
//
// The mapper uses structural matching:
// 1. Strip filter expressions [?(...)] from the path
// 2. Strip the ~ (key name) operator
// 3. Match the structural path against known patterns
// 4. Return mapping info for code generation
//
// Note on ~ operator: The ~ suffix in JSONPath (e.g., $.paths[*]~) is a Spectral
// extension, not standard JSONPath. It extracts the map key rather than the value.
func MapJSONPath(path string) JSONPathMapping {
	result := JSONPathMapping{OriginalPath: path}

	// Strip and capture filter expressions
	cleaned, filter := stripFilters(path)

	// Strip ~ operator
	isKeyAccess := false
	if strings.HasSuffix(cleaned, "~") {
		isKeyAccess = true
		cleaned = strings.TrimSuffix(cleaned, "~")
		cleaned = strings.TrimRight(cleaned, ".")
	}

	result.IsKeyAccess = isKeyAccess
	result.Filter = filter

	// Normalize: remove trailing dots, spaces
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimRight(cleaned, ".")

	// Try matching known patterns (most specific first)
	if m := matchPathPattern(cleaned); m != nil {
		result.Collection = m.collection
		result.IsDirect = m.isDirect
		result.DirectAccess = m.directAccess
		result.FieldAccess = m.fieldAccess
		result.HTTPMethod = m.httpMethod
		result.IsKeyAccess = isKeyAccess || m.isKeyAccess
		return result
	}

	// Fallback: unsupported
	result.Unsupported = true
	return result
}

type patternMatch struct {
	collection   string
	isDirect     bool
	directAccess string
	fieldAccess  string
	httpMethod   string
	isKeyAccess  bool
}

var httpMethods = map[string]bool{
	"get": true, "post": true, "put": true, "delete": true,
	"patch": true, "options": true, "head": true, "trace": true,
}

// patternEntry defines a JSONPath structural pattern to match.
type patternEntry struct {
	match func(string) *patternMatch
}

// patternTable is checked in order — most specific patterns first.
var patternTable = []patternEntry{
	// $.paths[*][*].responses[*]
	{match: matchPathsOperationResponses},
	// $.paths[*][*].requestBody
	{match: matchPathsOperationRequestBody},
	// $.paths[*].{method} — specific HTTP method
	{match: matchPathsMethod},
	// $.paths[*][*] — all operations
	{match: matchPathsAllOps},
	// $.paths[*] — path items
	{match: matchPathsItems},
	// $..parameters[*] or $.paths..parameters[*]
	{match: matchParameters},
	// $.info.contact
	{match: matchInfoContact},
	// $.info.license
	{match: matchInfoLicense},
	// $.info.version or $.info.{field}
	{match: matchInfoField},
	// $.info
	{match: matchInfo},
	// $.servers[*].url or $.servers[*].{field}
	{match: matchServersField},
	// $.servers[*]
	{match: matchServers},
	// $.tags[*].name or $.tags[*].{field}
	{match: matchTagsField},
	// $.tags[*]
	{match: matchTags},
	// $.components.schemas[*]
	{match: matchComponentSchemas},
	// $.components.responses[*]
	{match: matchComponentResponses},
	// $.components.parameters[*]
	{match: matchComponentParameters},
	// $.components.securitySchemes[*]
	{match: matchComponentSecuritySchemes},
	// $.components.examples[*]
	{match: matchComponentExamples},
	// $.definitions[*] (OAS2 schemas)
	{match: matchDefinitions},
	// $..properties[*]
	{match: matchProperties},
	// $.components
	{match: matchComponents},
	// $..description
	{match: matchDescriptionNodes},
	// $..summary
	{match: matchSummaryNodes},
	// $ — root
	{match: matchRoot},
}

func matchPathPattern(cleaned string) *patternMatch {
	for _, entry := range patternTable {
		if m := entry.match(cleaned); m != nil {
			return m
		}
	}
	return nil
}

// --- Pattern matchers ---

var rePathsOperationResponses = regexp.MustCompile(`^\$\.paths\[\*\]\[\*\]\.responses\[\*\]$`)

func matchPathsOperationResponses(s string) *patternMatch {
	if rePathsOperationResponses.MatchString(s) {
		return &patternMatch{collection: "inlineResponses"}
	}
	return nil
}

var rePathsOperationRequestBody = regexp.MustCompile(`^\$\.paths\[\*\]\[\*\]\.requestBody$`)

func matchPathsOperationRequestBody(s string) *patternMatch {
	if rePathsOperationRequestBody.MatchString(s) {
		return &patternMatch{collection: "inlineRequestBodies"}
	}
	return nil
}

func matchPathsMethod(s string) *patternMatch {
	// $.paths[*].get, $.paths[*].post, etc.
	for method := range httpMethods {
		if s == "$.paths[*]."+method {
			return &patternMatch{collection: "operations", httpMethod: method}
		}
	}
	return nil
}

var rePathsAllOps = regexp.MustCompile(`^\$\.paths(\[\*\]|\.\*)\[\*\]$`)

func matchPathsAllOps(s string) *patternMatch {
	if rePathsAllOps.MatchString(s) {
		return &patternMatch{collection: "operations"}
	}
	return nil
}

func matchPathsItems(s string) *patternMatch {
	// $.paths[*] or $.paths.* but NOT $.paths[*][*] or $.paths[*].something
	if s == "$.paths[*]" || s == "$.paths.*" {
		return &patternMatch{collection: "inlinePathItems"}
	}
	return nil
}

var reParamsGlobal = regexp.MustCompile(`^\$\.?\.parameters\[\*\]$|^\$\.paths.*parameters\[\*\]$`)

func matchParameters(s string) *patternMatch {
	if reParamsGlobal.MatchString(s) {
		return &patternMatch{collection: "inlineParameters"}
	}
	// $..parameters[*] — recursive descent
	if s == "$..parameters[*]" {
		return &patternMatch{collection: "inlineParameters"}
	}
	return nil
}

func matchInfoContact(s string) *patternMatch {
	if s == "$.info.contact" {
		return &patternMatch{isDirect: true, directAccess: "docInfo.document.getInfo()?.getContact()"}
	}
	return nil
}

func matchInfoLicense(s string) *patternMatch {
	if s == "$.info.license" {
		return &patternMatch{isDirect: true, directAccess: "docInfo.document.getInfo()?.getLicense()"}
	}
	return nil
}

func matchInfoField(s string) *patternMatch {
	if strings.HasPrefix(s, "$.info.") {
		field := strings.TrimPrefix(s, "$.info.")
		if field != "" && !strings.Contains(field, ".") {
			return &patternMatch{isDirect: true, directAccess: "docInfo.document.getInfo()", fieldAccess: field}
		}
	}
	return nil
}

func matchInfo(s string) *patternMatch {
	if s == "$.info" {
		return &patternMatch{isDirect: true, directAccess: "docInfo.document.getInfo()"}
	}
	return nil
}

func matchServersField(s string) *patternMatch {
	prefixes := []string{"$.servers[*].", "$..servers[*].", "$.servers.."}
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			field := strings.TrimPrefix(s, prefix)
			if field != "" && !strings.Contains(field, ".") {
				return &patternMatch{collection: "servers", fieldAccess: field}
			}
		}
	}
	return nil
}

func matchServers(s string) *patternMatch {
	if s == "$.servers[*]" || s == "$.servers" {
		return &patternMatch{collection: "servers"}
	}
	return nil
}

func matchTagsField(s string) *patternMatch {
	if strings.HasPrefix(s, "$.tags[*].") {
		field := strings.TrimPrefix(s, "$.tags[*].")
		if field != "" {
			return &patternMatch{collection: "tags", fieldAccess: field}
		}
	}
	return nil
}

func matchTags(s string) *patternMatch {
	if s == "$.tags[*]" || s == "$.tags" {
		return &patternMatch{collection: "tags"}
	}
	return nil
}

func matchComponentSchemas(s string) *patternMatch {
	if s == "$.components.schemas[*]" || s == "$.components.schemas.*" {
		return &patternMatch{collection: "componentSchemas"}
	}
	return nil
}

func matchComponentResponses(s string) *patternMatch {
	if s == "$.components.responses[*]" || s == "$.components.responses.*" {
		return &patternMatch{collection: "componentResponses"}
	}
	return nil
}

func matchComponentParameters(s string) *patternMatch {
	if s == "$.components.parameters[*]" || s == "$.components.parameters.*" {
		return &patternMatch{collection: "componentParameters"}
	}
	return nil
}

func matchComponentSecuritySchemes(s string) *patternMatch {
	if s == "$.components.securitySchemes[*]" || s == "$.components.securitySchemes.*" {
		return &patternMatch{collection: "componentSecuritySchemes"}
	}
	return nil
}

func matchComponentExamples(s string) *patternMatch {
	if s == "$.components.examples[*]" || s == "$.components.examples.*" {
		return &patternMatch{collection: "componentExamples"}
	}
	return nil
}

func matchDefinitions(s string) *patternMatch {
	// OAS2 $.definitions[*]
	if s == "$.definitions[*]" || s == "$.definitions.*" {
		return &patternMatch{collection: "componentSchemas"}
	}
	// $.definitions..properties[*]
	if s == "$.definitions..properties[*]" {
		return &patternMatch{collection: "inlineSchemas"}
	}
	return nil
}

func matchProperties(s string) *patternMatch {
	if s == "$..properties[*]" {
		return &patternMatch{collection: "inlineSchemas"}
	}
	return nil
}

func matchComponents(s string) *patternMatch {
	if s == "$.components" {
		return &patternMatch{isDirect: true, directAccess: "docInfo.document.getComponents()"}
	}
	return nil
}

func matchDescriptionNodes(s string) *patternMatch {
	if s == "$..description" {
		return &patternMatch{collection: "descriptionNodes"}
	}
	return nil
}

func matchSummaryNodes(s string) *patternMatch {
	if s == "$..summary" {
		return &patternMatch{collection: "summaryNodes"}
	}
	return nil
}

func matchRoot(s string) *patternMatch {
	if s == "$" {
		return &patternMatch{isDirect: true, directAccess: "docInfo.document"}
	}
	return nil
}

var reFilter = regexp.MustCompile(`\[\?\([^)]*\)\]`)

// stripFilters removes [?(...)] filter expressions from a JSONPath and returns
// both the cleaned path and the extracted filter.
func stripFilters(path string) (cleaned, filter string) {
	filters := reFilter.FindAllString(path, -1)
	cleaned = reFilter.ReplaceAllString(path, "[*]")

	if len(filters) > 0 {
		filter = strings.Join(filters, " ")
	}
	return cleaned, filter
}
