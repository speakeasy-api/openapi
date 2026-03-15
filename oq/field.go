package oq

import (
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/graph"
	oas3 "github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/oq/expr"
)

// --- Field access ---

type rowAdapter struct {
	row Row
	g   *graph.SchemaGraph
	env map[string]expr.Value
}

func (r rowAdapter) Field(name string) expr.Value {
	if strings.HasPrefix(name, "$") && r.env != nil {
		if v, ok := r.env[name]; ok {
			return v
		}
		return expr.NullVal()
	}
	return fieldValue(r.row, name, r.g)
}

// FieldValuePublic returns the value of a named field for the given row.
// Exported for testing and external consumers.
func FieldValuePublic(row Row, name string, g *graph.SchemaGraph) expr.Value {
	return fieldValue(row, name, g)
}

func fieldValue(row Row, name string, g *graph.SchemaGraph) expr.Value {
	// Universal field: kind returns the row type as a string
	if name == "kind" {
		return expr.StringVal(resultKindName(row.Kind))
	}

	switch row.Kind {
	case SchemaResult:
		if row.SchemaIdx < 0 || row.SchemaIdx >= len(g.Schemas) {
			return expr.NullVal()
		}
		s := &g.Schemas[row.SchemaIdx]
		switch name {
		// --- Graph-level fields (pre-computed) ---
		case "name":
			return expr.StringVal(s.Name)
		case "type":
			return expr.StringVal(s.Type)
		case "depth":
			return expr.IntVal(s.Depth)
		case "inDegree":
			return expr.IntVal(s.InDegree)
		case "outDegree":
			return expr.IntVal(s.OutDegree)
		case "unionWidth":
			return expr.IntVal(s.UnionWidth)
		case "allOfCount":
			return expr.IntVal(s.AllOfCount)
		case "oneOfCount":
			return expr.IntVal(s.OneOfCount)
		case "anyOfCount":
			return expr.IntVal(s.AnyOfCount)
		case "propertyCount":
			return expr.IntVal(s.PropertyCount)
		case "isComponent":
			return expr.BoolVal(s.IsComponent)
		case "isInline":
			return expr.BoolVal(s.IsInline)
		case "isCircular":
			return expr.BoolVal(s.IsCircular)
		case "hasRef":
			return expr.BoolVal(s.HasRef)
		case "hash":
			return expr.StringVal(s.Hash)
		case "path":
			return expr.StringVal(s.Path)
		case "opCount":
			return expr.IntVal(g.SchemaOpCount(graph.NodeID(row.SchemaIdx)))
		case "tagCount":
			return expr.IntVal(schemaTagCount(row.SchemaIdx, g))
		case "via":
			return expr.StringVal(row.Via)
		case "key":
			return expr.StringVal(row.Key)
		case "from":
			return expr.StringVal(row.From)
		case "seed":
			return expr.StringVal(row.Target)
		case "bfsDepth":
			return expr.IntVal(row.BFSDepth)
		case "ref":
			if s.HasRef {
				target := resolveRefTarget(row.SchemaIdx, g)
				if target != row.SchemaIdx {
					return expr.StringVal(schemaName(target, g))
				}
			}
			return expr.StringVal("")
		case "properties":
			return schemaPropertyNames(row.SchemaIdx, g)
		default:
			// Schema-content fields require the underlying schema object
			return schemaContentField(s, name)
		}
	case OperationResult:
		if row.OpIdx < 0 || row.OpIdx >= len(g.Operations) {
			return expr.NullVal()
		}
		o := &g.Operations[row.OpIdx]
		switch name {
		case "name":
			return expr.StringVal(o.Name)
		case "method":
			return expr.StringVal(o.Method)
		case "path":
			return expr.StringVal(o.Path)
		case "operationId":
			return expr.StringVal(o.OperationID)
		case "schemaCount":
			return expr.IntVal(o.SchemaCount)
		case "componentCount":
			return expr.IntVal(o.ComponentCount)
		case "tag":
			if o.Operation != nil && len(o.Operation.Tags) > 0 {
				return expr.StringVal(o.Operation.Tags[0])
			}
			return expr.StringVal("")
		case "parameterCount":
			if o.Operation != nil {
				return expr.IntVal(len(o.Operation.Parameters))
			}
			return expr.IntVal(0)
		case "deprecated":
			if o.Operation != nil {
				return expr.BoolVal(o.Operation.Deprecated != nil && *o.Operation.Deprecated)
			}
			return expr.BoolVal(false)
		case "description":
			if o.Operation != nil {
				return expr.StringVal(o.Operation.GetDescription())
			}
			return expr.StringVal("")
		case "summary":
			if o.Operation != nil {
				return expr.StringVal(o.Operation.GetSummary())
			}
			return expr.StringVal("")
		case "via":
			return expr.StringVal(row.Via)
		case "key":
			return expr.StringVal(row.Key)
		case "from":
			return expr.StringVal(row.From)
		default:
			return operationContentField(o, name)
		}
	case GroupRowResult:
		switch name {
		case "key":
			return expr.StringVal(row.GroupKey)
		case "count":
			return expr.IntVal(row.GroupCount)
		case "name":
			return expr.StringVal(row.GroupKey)
		case "names":
			return expr.StringVal(strings.Join(row.GroupNames, ", "))
		}
	case ParameterResult:
		p := row.Parameter
		if p == nil {
			return expr.NullVal()
		}
		switch name {
		case "name":
			return expr.StringVal(row.ComponentKey)
		case "in":
			return expr.StringVal(string(p.In))
		case "required":
			return expr.BoolVal(p.Required != nil && *p.Required)
		case "deprecated":
			return expr.BoolVal(p.Deprecated != nil && *p.Deprecated)
		case "description":
			return expr.StringVal(p.GetDescription())
		case "style":
			if p.Style != nil {
				return expr.StringVal(string(*p.Style))
			}
			return expr.StringVal("")
		case "explode":
			return expr.BoolVal(p.Explode != nil && *p.Explode)
		case "hasSchema":
			return expr.BoolVal(p.Schema != nil)
		case "allowEmptyValue":
			return expr.BoolVal(p.AllowEmptyValue != nil && *p.AllowEmptyValue)
		case "allowReserved":
			return expr.BoolVal(p.AllowReserved != nil && *p.AllowReserved)
		case "operation":
			return expr.StringVal(operationName(row.SourceOpIdx, g))
		}
	case ResponseResult:
		r := row.Response
		if r == nil {
			return expr.NullVal()
		}
		switch name {
		case "statusCode":
			return expr.StringVal(row.StatusCode)
		case "name":
			if row.ComponentKey != "" {
				return expr.StringVal(row.ComponentKey)
			}
			return expr.StringVal(row.StatusCode)
		case "description":
			return expr.StringVal(r.Description)
		case "contentTypeCount":
			return expr.IntVal(r.Content.Len())
		case "headerCount":
			return expr.IntVal(r.Headers.Len())
		case "linkCount":
			return expr.IntVal(r.Links.Len())
		case "hasContent":
			return expr.BoolVal(r.Content != nil && r.Content.Len() > 0)
		case "operation":
			return expr.StringVal(operationName(row.SourceOpIdx, g))
		}
	case RequestBodyResult:
		rb := row.RequestBody
		if rb == nil {
			return expr.NullVal()
		}
		switch name {
		case "name":
			if row.ComponentKey != "" {
				return expr.StringVal(row.ComponentKey)
			}
			return expr.StringVal("request-body")
		case "description":
			return expr.StringVal(rb.GetDescription())
		case "required":
			return expr.BoolVal(rb.Required != nil && *rb.Required)
		case "contentTypeCount":
			return expr.IntVal(rb.Content.Len())
		case "operation":
			return expr.StringVal(operationName(row.SourceOpIdx, g))
		}
	case ContentTypeResult:
		mt := row.ContentType
		if mt == nil {
			return expr.NullVal()
		}
		switch name {
		case "mediaType", "name":
			return expr.StringVal(row.MediaTypeName)
		case "hasSchema":
			return expr.BoolVal(mt.Schema != nil)
		case "hasEncoding":
			return expr.BoolVal(mt.Encoding != nil && mt.Encoding.Len() > 0)
		case "hasExample":
			return expr.BoolVal(mt.Example != nil || (mt.Examples != nil && mt.Examples.Len() > 0))
		case "statusCode":
			return expr.StringVal(row.StatusCode)
		case "operation":
			return expr.StringVal(operationName(row.SourceOpIdx, g))
		}
	case HeaderResult:
		h := row.Header
		if h == nil {
			return expr.NullVal()
		}
		switch name {
		case "name":
			return expr.StringVal(row.HeaderName)
		case "description":
			return expr.StringVal(h.GetDescription())
		case "required":
			return expr.BoolVal(h.Required != nil && *h.Required)
		case "deprecated":
			return expr.BoolVal(h.Deprecated != nil && *h.Deprecated)
		case "hasSchema":
			return expr.BoolVal(h.Schema != nil)
		case "statusCode":
			return expr.StringVal(row.StatusCode)
		case "operation":
			return expr.StringVal(operationName(row.SourceOpIdx, g))
		}
	case SecuritySchemeResult:
		ss := row.SecurityScheme
		if ss == nil {
			return expr.NullVal()
		}
		switch name {
		case "name":
			return expr.StringVal(row.SchemeName)
		case "type":
			return expr.StringVal(string(ss.GetType()))
		case "in":
			return expr.StringVal(string(ss.GetIn()))
		case "scheme":
			return expr.StringVal(ss.GetScheme())
		case "bearerFormat":
			return expr.StringVal(ss.GetBearerFormat())
		case "description":
			return expr.StringVal(ss.GetDescription())
		case "hasFlows":
			return expr.BoolVal(ss.GetFlows() != nil)
		case "deprecated":
			return expr.BoolVal(ss.Deprecated != nil && *ss.Deprecated)
		}
	case SecurityRequirementResult:
		switch name {
		case "name", "schemeName":
			return expr.StringVal(row.SchemeName)
		case "schemeType":
			if row.SecurityScheme != nil {
				return expr.StringVal(string(row.SecurityScheme.GetType()))
			}
			return expr.StringVal("")
		case "scopes":
			return expr.ArrayVal(row.Scopes)
		case "scopeCount":
			return expr.IntVal(len(row.Scopes))
		case "operation":
			return expr.StringVal(operationName(row.SourceOpIdx, g))
		}
	}
	return expr.NullVal()
}

func resultKindName(k ResultKind) string {
	switch k {
	case SchemaResult:
		return "schema"
	case OperationResult:
		return "operation"
	case GroupRowResult:
		return "group"
	case ParameterResult:
		return "parameter"
	case ResponseResult:
		return "response"
	case RequestBodyResult:
		return "request-body"
	case ContentTypeResult:
		return "content-type"
	case HeaderResult:
		return "header"
	case SecuritySchemeResult:
		return "security-scheme"
	case SecurityRequirementResult:
		return "security-requirement"
	default:
		return "unknown"
	}
}

// schemaPropertyNames returns the property names of a schema as an array value.
func schemaPropertyNames(schemaIdx int, g *graph.SchemaGraph) expr.Value {
	var names []string
	for _, edge := range g.OutEdges(graph.NodeID(schemaIdx)) {
		if edge.Kind == graph.EdgeProperty {
			names = append(names, edge.Label)
		}
	}
	return expr.ArrayVal(names)
}

// operationName returns the operation name for the given index, or empty string if out of range.
func operationName(opIdx int, g *graph.SchemaGraph) string {
	if opIdx >= 0 && opIdx < len(g.Operations) {
		return g.Operations[opIdx].Name
	}
	return ""
}

// schemaContentField resolves fields by reading the underlying schema object.
func schemaContentField(s *graph.SchemaNode, name string) expr.Value {
	schema := getSchema(s)

	switch name {
	// --- Metadata ---
	case "description":
		if schema != nil && schema.Description != nil {
			return expr.StringVal(*schema.Description)
		}
		return expr.StringVal("")
	case "title":
		if schema != nil && schema.Title != nil {
			return expr.StringVal(*schema.Title)
		}
		return expr.StringVal("")
	// --- Format & Pattern ---
	case "format":
		if schema != nil && schema.Format != nil {
			return expr.StringVal(*schema.Format)
		}
		return expr.StringVal("")
	case "pattern":
		if schema != nil && schema.Pattern != nil {
			return expr.StringVal(*schema.Pattern)
		}
		return expr.StringVal("")

	// --- Flags ---
	case "nullable":
		return expr.BoolVal(schema != nil && schema.Nullable != nil && *schema.Nullable)
	case "readOnly":
		return expr.BoolVal(schema != nil && schema.ReadOnly != nil && *schema.ReadOnly)
	case "writeOnly":
		return expr.BoolVal(schema != nil && schema.WriteOnly != nil && *schema.WriteOnly)
	case "deprecated":
		return expr.BoolVal(schema != nil && schema.Deprecated != nil && *schema.Deprecated)
	case "uniqueItems":
		return expr.BoolVal(schema != nil && schema.UniqueItems != nil && *schema.UniqueItems)

	// --- Discriminator ---
	case "discriminatorProperty":
		if schema != nil && schema.Discriminator != nil {
			return expr.StringVal(schema.Discriminator.PropertyName)
		}
		return expr.StringVal("")
	case "discriminatorMappingCount":
		if schema != nil && schema.Discriminator != nil && schema.Discriminator.Mapping != nil {
			return expr.IntVal(schema.Discriminator.Mapping.Len())
		}
		return expr.IntVal(0)

	// --- Counts & Arrays ---
	case "required":
		if schema != nil {
			return expr.ArrayVal(schema.Required)
		}
		return expr.ArrayVal(nil)
	case "requiredCount":
		if schema != nil {
			return expr.IntVal(len(schema.Required))
		}
		return expr.IntVal(0)
	case "enum":
		if schema != nil {
			vals := make([]string, len(schema.Enum))
			for i, e := range schema.Enum {
				if e != nil {
					vals[i] = e.Value
				}
			}
			return expr.ArrayVal(vals)
		}
		return expr.ArrayVal(nil)
	case "enumCount":
		if schema != nil {
			return expr.IntVal(len(schema.Enum))
		}
		return expr.IntVal(0)

	// --- Numeric constraints ---
	case "minimum":
		if schema != nil && schema.Minimum != nil {
			return expr.IntVal(int(*schema.Minimum))
		}
		return expr.NullVal()
	case "maximum":
		if schema != nil && schema.Maximum != nil {
			return expr.IntVal(int(*schema.Maximum))
		}
		return expr.NullVal()

	// --- String constraints ---
	case "minLength":
		if schema != nil && schema.MinLength != nil {
			return expr.IntVal(int(*schema.MinLength))
		}
		return expr.NullVal()
	case "maxLength":
		if schema != nil && schema.MaxLength != nil {
			return expr.IntVal(int(*schema.MaxLength))
		}
		return expr.NullVal()

	// --- Array constraints ---
	case "minItems":
		if schema != nil && schema.MinItems != nil {
			return expr.IntVal(int(*schema.MinItems))
		}
		return expr.NullVal()
	case "maxItems":
		if schema != nil && schema.MaxItems != nil {
			return expr.IntVal(int(*schema.MaxItems))
		}
		return expr.NullVal()

	// --- Object constraints ---
	case "minProperties":
		if schema != nil && schema.MinProperties != nil {
			return expr.IntVal(int(*schema.MinProperties))
		}
		return expr.NullVal()
	case "maxProperties":
		if schema != nil && schema.MaxProperties != nil {
			return expr.IntVal(int(*schema.MaxProperties))
		}
		return expr.NullVal()

	// --- Extensions ---
	case "extensionCount":
		if schema != nil && schema.Extensions != nil {
			return expr.IntVal(schema.Extensions.Len())
		}
		return expr.IntVal(0)

	// --- Content encoding (OAS 3.1+) ---
	case "contentEncoding":
		if schema != nil && schema.ContentEncoding != nil {
			return expr.StringVal(*schema.ContentEncoding)
		}
		return expr.StringVal("")
	case "contentMediaType":
		if schema != nil && schema.ContentMediaType != nil {
			return expr.StringVal(*schema.ContentMediaType)
		}
		return expr.StringVal("")
	}

	// --- Raw YAML fallback ---
	// Unknown fields fall through to the underlying schema object.
	// Supports snake_case (additional_properties) and camelCase (additionalProperties).
	if schema != nil {
		return schemaRawField(schema, name)
	}
	return expr.NullVal()
}

// snakeToCamel converts snake_case to camelCase.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// schemaRawField probes the underlying schema object for arbitrary fields.
// This allows queries like where(has(additionalProperties)) or
// select name, additional_properties without pre-defining every field.
func schemaRawField(schema *oas3.Schema, name string) expr.Value {
	// Normalize: try the name as-is, then convert snake_case → camelCase
	candidates := []string{name}
	if strings.Contains(name, "_") {
		candidates = append(candidates, snakeToCamel(name))
	}

	for _, field := range candidates {
		if v, ok := probeSchemaField(schema, field); ok {
			return v
		}
	}
	return expr.NullVal()
}

// probeSchemaField checks if a named field exists and is non-nil on the schema.
// probeSchemaField checks for arbitrary schema fields not covered by the
// pre-defined field set. Returns (value, true) if the field is recognized,
// (null, false) if not.
func probeSchemaField(schema *oas3.Schema, name string) (expr.Value, bool) {
	// Presence-check fields: non-nil → true, nil → null
	if probe, ok := schemaPresenceProbes[name]; ok {
		if probe(schema) {
			return expr.BoolVal(true), true
		}
		return expr.NullVal(), true
	}

	// Value-returning fields
	switch name {
	case "const":
		if schema.Const != nil {
			return expr.StringVal(schema.Const.Value), true
		}
		return expr.NullVal(), true
	case "multipleOf":
		if schema.MultipleOf != nil {
			return expr.IntVal(int(*schema.MultipleOf)), true
		}
		return expr.NullVal(), true
	case "anchor":
		if schema.Anchor != nil {
			return expr.StringVal(*schema.Anchor), true
		}
		return expr.NullVal(), true
	case "id":
		if schema.ID != nil {
			return expr.StringVal(*schema.ID), true
		}
		return expr.NullVal(), true
	case "schema":
		if schema.Schema != nil {
			return expr.StringVal(*schema.Schema), true
		}
		return expr.NullVal(), true
	case "prefixItems":
		if len(schema.PrefixItems) > 0 {
			return expr.IntVal(len(schema.PrefixItems)), true
		}
		return expr.NullVal(), true
	case "dependentSchemas":
		if schema.DependentSchemas != nil && schema.DependentSchemas.Len() > 0 {
			return expr.IntVal(schema.DependentSchemas.Len()), true
		}
		return expr.NullVal(), true
	case "defs":
		if schema.Defs != nil && schema.Defs.Len() > 0 {
			return expr.IntVal(schema.Defs.Len()), true
		}
		return expr.NullVal(), true
	case "examples":
		if len(schema.Examples) > 0 {
			return expr.IntVal(len(schema.Examples)), true
		}
		return expr.NullVal(), true
	}

	// Check x- extensions
	if strings.HasPrefix(name, "x-") || strings.HasPrefix(name, "x_") {
		extKey := name
		if strings.HasPrefix(name, "x_") {
			extKey = "x-" + name[2:]
		}
		if schema.Extensions != nil {
			if v, ok := schema.Extensions.Get(extKey); ok {
				return expr.StringVal(v.Value), true
			}
		}
		return expr.NullVal(), true
	}

	return expr.NullVal(), false
}

// schemaPresenceProbes maps field names to nil-checks on the schema.
// Each returns true if the field is present/non-nil.
var schemaPresenceProbes = map[string]func(*oas3.Schema) bool{
	"additionalProperties":  func(s *oas3.Schema) bool { return s.AdditionalProperties != nil },
	"patternProperties":     func(s *oas3.Schema) bool { return s.PatternProperties != nil && s.PatternProperties.Len() > 0 },
	"xml":                   func(s *oas3.Schema) bool { return s.XML != nil },
	"externalDocs":          func(s *oas3.Schema) bool { return s.ExternalDocs != nil },
	"not":                   func(s *oas3.Schema) bool { return s.Not != nil },
	"if":                    func(s *oas3.Schema) bool { return s.If != nil },
	"then":                  func(s *oas3.Schema) bool { return s.Then != nil },
	"else":                  func(s *oas3.Schema) bool { return s.Else != nil },
	"contains":              func(s *oas3.Schema) bool { return s.Contains != nil },
	"propertyNames":         func(s *oas3.Schema) bool { return s.PropertyNames != nil },
	"unevaluatedItems":      func(s *oas3.Schema) bool { return s.UnevaluatedItems != nil },
	"unevaluatedProperties": func(s *oas3.Schema) bool { return s.UnevaluatedProperties != nil },
}

// operationContentField resolves fields by reading the underlying operation object.
func operationContentField(o *graph.OperationNode, name string) expr.Value {
	op := o.Operation
	if op == nil {
		return expr.NullVal()
	}

	switch name {
	case "responseCount":
		return expr.IntVal(op.Responses.Len())
	case "hasErrorResponse":
		return expr.BoolVal(hasErrorResponse(op))
	case "hasRequestBody":
		return expr.BoolVal(op.RequestBody != nil)
	case "securityCount":
		return expr.IntVal(len(op.Security))
	case "tags":
		return expr.ArrayVal(op.Tags)
	}

	return expr.NullVal()
}

// getSchema extracts the underlying *Schema from a SchemaNode, if available.
func getSchema(s *graph.SchemaNode) *oas3.Schema {
	if s.Schema == nil {
		return nil
	}
	return s.Schema.GetSchema()
}

// hasErrorResponse returns true if the operation has any 4xx/5xx response codes
// or a default response (which conventionally represents errors).
func hasErrorResponse(op *openapi.Operation) bool {
	if op.Responses.Map != nil {
		for code := range op.Responses.All() {
			if len(code) >= 1 && (code[0] == '4' || code[0] == '5') {
				return true
			}
		}
	}
	return op.Responses.Default != nil
}

func compareValues(a, b expr.Value) int {
	if a.Kind == expr.KindInt && b.Kind == expr.KindInt {
		if a.Int < b.Int {
			return -1
		}
		if a.Int > b.Int {
			return 1
		}
		return 0
	}
	sa := valueToString(a)
	sb := valueToString(b)
	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
}

func valueToString(v expr.Value) string {
	switch v.Kind {
	case expr.KindString:
		return v.Str
	case expr.KindInt:
		return strconv.Itoa(v.Int)
	case expr.KindBool:
		return strconv.FormatBool(v.Bool)
	case expr.KindArray:
		return strings.Join(v.Arr, ", ")
	default:
		return ""
	}
}

func rowKey(row Row) string {
	switch row.Kind {
	case SchemaResult:
		return "s:" + strconv.Itoa(row.SchemaIdx)
	case OperationResult:
		return "o:" + strconv.Itoa(row.OpIdx)
	case GroupRowResult:
		return "g:" + row.GroupKey
	case ParameterResult:
		return "p:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.ComponentKey
	case ResponseResult:
		return "r:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.StatusCode
	case RequestBodyResult:
		return "rb:" + strconv.Itoa(row.SourceOpIdx)
	case ContentTypeResult:
		return "ct:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.StatusCode + ":" + row.MediaTypeName
	case HeaderResult:
		return "h:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.StatusCode + ":" + row.HeaderName
	case SecuritySchemeResult:
		return "ss:" + row.SchemeName
	case SecurityRequirementResult:
		return "sr:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.SchemeName
	default:
		return "?:" + strconv.Itoa(row.OpIdx)
	}
}
