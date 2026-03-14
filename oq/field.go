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
		case "in_degree":
			return expr.IntVal(s.InDegree)
		case "out_degree":
			return expr.IntVal(s.OutDegree)
		case "union_width":
			return expr.IntVal(s.UnionWidth)
		case "property_count":
			return expr.IntVal(s.PropertyCount)
		case "is_component":
			return expr.BoolVal(s.IsComponent)
		case "is_inline":
			return expr.BoolVal(s.IsInline)
		case "is_circular":
			return expr.BoolVal(s.IsCircular)
		case "has_ref":
			return expr.BoolVal(s.HasRef)
		case "hash":
			return expr.StringVal(s.Hash)
		case "path":
			return expr.StringVal(s.Path)
		case "op_count":
			return expr.IntVal(g.SchemaOpCount(graph.NodeID(row.SchemaIdx)))
		case "tag_count":
			return expr.IntVal(schemaTagCount(row.SchemaIdx, g))
		case "via":
			return expr.StringVal(row.Via)
		case "key":
			return expr.StringVal(row.Key)
		case "from":
			return expr.StringVal(row.From)
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
		case "operation_id":
			return expr.StringVal(o.OperationID)
		case "schema_count":
			return expr.IntVal(o.SchemaCount)
		case "component_count":
			return expr.IntVal(o.ComponentCount)
		case "tag":
			if o.Operation != nil && len(o.Operation.Tags) > 0 {
				return expr.StringVal(o.Operation.Tags[0])
			}
			return expr.StringVal("")
		case "parameter_count":
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
			return expr.StringVal(row.ParamName)
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
		case "has_schema":
			return expr.BoolVal(p.Schema != nil)
		case "allow_empty_value":
			return expr.BoolVal(p.AllowEmptyValue != nil && *p.AllowEmptyValue)
		case "allow_reserved":
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
		case "status_code", "name":
			return expr.StringVal(row.StatusCode)
		case "description":
			return expr.StringVal(r.Description)
		case "content_type_count":
			return expr.IntVal(r.Content.Len())
		case "header_count":
			return expr.IntVal(r.Headers.Len())
		case "link_count":
			return expr.IntVal(r.Links.Len())
		case "has_content":
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
			return expr.StringVal("request-body")
		case "description":
			return expr.StringVal(rb.GetDescription())
		case "required":
			return expr.BoolVal(rb.Required != nil && *rb.Required)
		case "content_type_count":
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
		case "media_type", "name":
			return expr.StringVal(row.MediaTypeName)
		case "has_schema":
			return expr.BoolVal(mt.Schema != nil)
		case "has_encoding":
			return expr.BoolVal(mt.Encoding != nil && mt.Encoding.Len() > 0)
		case "has_example":
			return expr.BoolVal(mt.Example != nil || (mt.Examples != nil && mt.Examples.Len() > 0))
		case "status_code":
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
		case "has_schema":
			return expr.BoolVal(h.Schema != nil)
		case "status_code":
			return expr.StringVal(row.StatusCode)
		case "operation":
			return expr.StringVal(operationName(row.SourceOpIdx, g))
		}
	}
	return expr.NullVal()
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
	case "has_description":
		return expr.BoolVal(schema != nil && schema.Description != nil && *schema.Description != "")
	case "title":
		if schema != nil && schema.Title != nil {
			return expr.StringVal(*schema.Title)
		}
		return expr.StringVal("")
	case "has_title":
		return expr.BoolVal(schema != nil && schema.Title != nil && *schema.Title != "")

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
	case "read_only":
		return expr.BoolVal(schema != nil && schema.ReadOnly != nil && *schema.ReadOnly)
	case "write_only":
		return expr.BoolVal(schema != nil && schema.WriteOnly != nil && *schema.WriteOnly)
	case "deprecated":
		return expr.BoolVal(schema != nil && schema.Deprecated != nil && *schema.Deprecated)
	case "unique_items":
		return expr.BoolVal(schema != nil && schema.UniqueItems != nil && *schema.UniqueItems)

	// --- Discriminator ---
	case "has_discriminator":
		return expr.BoolVal(schema != nil && schema.Discriminator != nil)
	case "discriminator_property":
		if schema != nil && schema.Discriminator != nil {
			return expr.StringVal(schema.Discriminator.PropertyName)
		}
		return expr.StringVal("")
	case "discriminator_mapping_count":
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
	case "required_count":
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
	case "enum_count":
		if schema != nil {
			return expr.IntVal(len(schema.Enum))
		}
		return expr.IntVal(0)

	// --- Defaults & Examples ---
	case "has_default":
		return expr.BoolVal(schema != nil && schema.Default != nil)
	case "has_example":
		return expr.BoolVal(schema != nil && (schema.Example != nil || len(schema.Examples) > 0))

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
	case "min_length":
		if schema != nil && schema.MinLength != nil {
			return expr.IntVal(int(*schema.MinLength))
		}
		return expr.NullVal()
	case "max_length":
		if schema != nil && schema.MaxLength != nil {
			return expr.IntVal(int(*schema.MaxLength))
		}
		return expr.NullVal()

	// --- Array constraints ---
	case "min_items":
		if schema != nil && schema.MinItems != nil {
			return expr.IntVal(int(*schema.MinItems))
		}
		return expr.NullVal()
	case "max_items":
		if schema != nil && schema.MaxItems != nil {
			return expr.IntVal(int(*schema.MaxItems))
		}
		return expr.NullVal()

	// --- Object constraints ---
	case "min_properties":
		if schema != nil && schema.MinProperties != nil {
			return expr.IntVal(int(*schema.MinProperties))
		}
		return expr.NullVal()
	case "max_properties":
		if schema != nil && schema.MaxProperties != nil {
			return expr.IntVal(int(*schema.MaxProperties))
		}
		return expr.NullVal()

	// --- Extensions ---
	case "extension_count":
		if schema != nil && schema.Extensions != nil {
			return expr.IntVal(schema.Extensions.Len())
		}
		return expr.IntVal(0)

	// --- Content encoding (OAS 3.1+) ---
	case "content_encoding":
		if schema != nil && schema.ContentEncoding != nil {
			return expr.StringVal(*schema.ContentEncoding)
		}
		return expr.StringVal("")
	case "content_media_type":
		if schema != nil && schema.ContentMediaType != nil {
			return expr.StringVal(*schema.ContentMediaType)
		}
		return expr.StringVal("")
	}

	return expr.NullVal()
}

// operationContentField resolves fields by reading the underlying operation object.
func operationContentField(o *graph.OperationNode, name string) expr.Value {
	op := o.Operation
	if op == nil {
		return expr.NullVal()
	}

	switch name {
	case "response_count":
		return expr.IntVal(op.Responses.Len())
	case "has_error_response":
		return expr.BoolVal(hasErrorResponse(op))
	case "has_request_body":
		return expr.BoolVal(op.RequestBody != nil)
	case "security_count":
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
		return "p:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.ParamName
	case ResponseResult:
		return "r:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.StatusCode
	case RequestBodyResult:
		return "rb:" + strconv.Itoa(row.SourceOpIdx)
	case ContentTypeResult:
		return "ct:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.StatusCode + ":" + row.MediaTypeName
	case HeaderResult:
		return "h:" + strconv.Itoa(row.SourceOpIdx) + ":" + row.StatusCode + ":" + row.HeaderName
	default:
		return "?:" + strconv.Itoa(row.OpIdx)
	}
}
