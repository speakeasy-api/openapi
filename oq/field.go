package oq

import (
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/graph"
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
		case "edge_kind":
			return expr.StringVal(row.EdgeKind)
		case "edge_label":
			return expr.StringVal(row.EdgeLabel)
		case "edge_from":
			return expr.StringVal(row.EdgeFrom)
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
		case "edge_kind":
			return expr.StringVal(row.EdgeKind)
		case "edge_label":
			return expr.StringVal(row.EdgeLabel)
		case "edge_from":
			return expr.StringVal(row.EdgeFrom)
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
	}
	return expr.NullVal()
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
	default:
		return ""
	}
}

func rowKey(row Row) string {
	switch row.Kind {
	case SchemaResult:
		return "s:" + strconv.Itoa(row.SchemaIdx)
	case GroupRowResult:
		return "g:" + row.GroupKey
	default:
		return "o:" + strconv.Itoa(row.OpIdx)
	}
}
