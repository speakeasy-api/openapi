// Package analyze provides schema complexity analysis for OpenAPI documents.
// It extracts a directed graph of schema references and computes metrics
// useful for schema maintainers and code generation teams.
package analyze

import (
	"context"
	"fmt"
	"slices"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
)

// EdgeKind describes how one schema references another.
type EdgeKind string

const (
	EdgeProperty             EdgeKind = "property"
	EdgeItems                EdgeKind = "items"
	EdgeAllOf                EdgeKind = "allOf"
	EdgeOneOf                EdgeKind = "oneOf"
	EdgeAnyOf                EdgeKind = "anyOf"
	EdgeAdditionalProperties EdgeKind = "additionalProperties"
	EdgeNot                  EdgeKind = "not"
	EdgePrefixItems          EdgeKind = "prefixItems"
	EdgeIf                   EdgeKind = "if"
	EdgeThen                 EdgeKind = "then"
	EdgeElse                 EdgeKind = "else"
)

// Node represents a schema in the dependency graph.
type Node struct {
	// ID is the unique identifier for this schema, typically the JSON pointer or component name.
	ID string
	// Name is the short display name (component name or last segment of JSON pointer).
	Name string
	// IsComponent is true if this schema lives in #/components/schemas/.
	IsComponent bool
	// JSONPointer is the full JSON pointer path to this schema.
	JSONPointer string
	// PropertyCount is the number of properties defined on this schema.
	PropertyCount int
	// RequiredCount is the number of required properties.
	RequiredCount int
	// Types is the list of types this schema declares.
	Types []string
	// HasDiscriminator is true if this schema has a discriminator defined.
	HasDiscriminator bool
	// CompositionFields tracks which composition keywords are used (allOf, oneOf, anyOf).
	CompositionFields []string
	// IsNullable is true if null is in the type list or nullable is true.
	IsNullable bool
	// DeepPropertyCount is the total properties across all inline sub-schemas.
	DeepPropertyCount int
	// NestingDepth is the maximum depth of inline object nesting (not counting $refs).
	NestingDepth int
	// CompositionDepth is the maximum depth of allOf/oneOf/anyOf nesting.
	CompositionDepth int
	// UnionSites lists all oneOf/anyOf occurrences found in the schema tree.
	UnionSites []UnionSite
}

// UnionSite represents a single oneOf/anyOf occurrence within a schema tree.
type UnionSite struct {
	// Kind is "oneOf" or "anyOf".
	Kind string
	// Width is the number of alternatives.
	Width int
	// HasDiscriminator is true if this site has a discriminator.
	HasDiscriminator bool
	// Path describes where in the tree this site was found (e.g., "data", "root").
	Path string
}

// Edge represents a reference from one schema to another.
type Edge struct {
	// From is the ID of the source schema.
	From string
	// To is the ID of the target schema.
	To string
	// Kind describes how the reference is made (property, items, allOf, etc.).
	Kind EdgeKind
	// FieldName is set when Kind is EdgeProperty — the property name.
	FieldName string
	// Index is set for allOf/oneOf/anyOf — the array index.
	Index int
	// IsRequired is true if this is a required property edge.
	IsRequired bool
	// IsNullable is true if the referencing schema allows null.
	IsNullable bool
	// IsArray is true if this edge goes through items (array wrapper).
	IsArray bool
}

// Graph is a directed graph of schema references extracted from an OpenAPI document.
type Graph struct {
	Nodes map[string]*Node
	Edges []*Edge

	// Adjacency lists for fast traversal.
	OutEdges map[string][]*Edge // edges from a node
	InEdges  map[string][]*Edge // edges to a node
}

// NewGraph creates an empty graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes:    make(map[string]*Node),
		OutEdges: make(map[string][]*Edge),
		InEdges:  make(map[string][]*Edge),
	}
}

func (g *Graph) addNode(n *Node) {
	if _, exists := g.Nodes[n.ID]; exists {
		return
	}
	g.Nodes[n.ID] = n
}

func (g *Graph) addEdge(e *Edge) {
	g.Edges = append(g.Edges, e)
	g.OutEdges[e.From] = append(g.OutEdges[e.From], e)
	g.InEdges[e.To] = append(g.InEdges[e.To], e)
}

// FanOut returns the number of distinct schemas this node references.
func (g *Graph) FanOut(nodeID string) int {
	seen := make(map[string]bool)
	for _, e := range g.OutEdges[nodeID] {
		seen[e.To] = true
	}
	return len(seen)
}

// FanIn returns the number of distinct schemas that reference this node.
func (g *Graph) FanIn(nodeID string) int {
	seen := make(map[string]bool)
	for _, e := range g.InEdges[nodeID] {
		seen[e.From] = true
	}
	return len(seen)
}

// BuildGraph extracts a schema reference graph from an OpenAPI document.
// It walks all component schemas and discovers their references to other schemas.
func BuildGraph(ctx context.Context, doc *openapi.OpenAPI) *Graph {
	g := NewGraph()

	if doc == nil || doc.Components == nil || doc.Components.Schemas == nil {
		return g
	}

	// Phase 1: Register all component schemas as nodes.
	for name, jsonSchema := range doc.Components.Schemas.All() {
		schema := jsonSchema.GetSchema()
		if schema == nil {
			continue
		}

		node := buildNodeFromSchema(name, schema, true)
		g.addNode(node)
	}

	// Phase 2: Walk each component schema and discover edges.
	for name, jsonSchema := range doc.Components.Schemas.All() {
		schema := jsonSchema.GetSchema()
		if schema == nil {
			continue
		}

		extractEdges(g, name, schema)
	}

	return g
}

func buildNodeFromSchema(name string, schema *oas3.Schema, isComponent bool) *Node {
	n := &Node{
		ID:          name,
		Name:        name,
		IsComponent: isComponent,
	}

	if isComponent {
		n.JSONPointer = fmt.Sprintf("#/components/schemas/%s", name)
	}

	if schema.Properties != nil {
		n.PropertyCount = schema.Properties.Len()
	}

	n.RequiredCount = len(schema.Required)
	n.Types = schemaTypeValues(schema)
	n.HasDiscriminator = schema.Discriminator != nil
	n.IsNullable = slices.Contains(n.Types, "null") || (schema.Nullable != nil && *schema.Nullable)

	// Analyze the full schema tree for composition, nesting, union sites
	stats := analyzeSchemaTree(schema)
	n.DeepPropertyCount = stats.deepPropertyCount
	n.NestingDepth = stats.nestingDepth
	n.CompositionDepth = stats.compositionDepth
	n.UnionSites = stats.unionSites
	for _, field := range []string{"allOf", "oneOf", "anyOf"} {
		if stats.compositionFields[field] {
			n.CompositionFields = append(n.CompositionFields, field)
		}
	}

	return n
}

// schemaTreeStats accumulates complexity metrics from walking a schema tree.
type schemaTreeStats struct {
	compositionFields map[string]bool
	deepPropertyCount int
	nestingDepth      int
	compositionDepth  int
	unionSites        []UnionSite
}

// analyzeSchemaTree walks an entire schema tree (recursing into inline sub-schemas)
// and computes deep property counts, nesting depth, composition depth, and union sites.
func analyzeSchemaTree(schema *oas3.Schema) *schemaTreeStats {
	stats := &schemaTreeStats{
		compositionFields: make(map[string]bool),
	}
	stats.walk(schema, 0, 0, "", make(map[*oas3.Schema]bool))
	return stats
}

func (s *schemaTreeStats) walk(schema *oas3.Schema, objectDepth, compDepth int, path string, seen map[*oas3.Schema]bool) {
	if schema == nil || seen[schema] {
		return
	}
	seen[schema] = true

	if objectDepth > s.nestingDepth {
		s.nestingDepth = objectDepth
	}
	if compDepth > s.compositionDepth {
		s.compositionDepth = compDepth
	}

	// Count properties at this level
	if schema.Properties != nil {
		s.deepPropertyCount += schema.Properties.Len()
		for propName, propSchema := range schema.Properties.All() {
			sub := propSchema.GetSchema()
			if sub != nil && sub.Ref == nil {
				childPath := propName
				if path != "" {
					childPath = path + "." + propName
				}
				nextDepth := objectDepth
				if sub.Properties != nil && sub.Properties.Len() > 0 {
					nextDepth = objectDepth + 1
				}
				s.walk(sub, nextDepth, compDepth, childPath, seen)
			}
		}
	}

	// allOf
	if len(schema.AllOf) > 0 {
		s.compositionFields["allOf"] = true
		if compDepth+1 > s.compositionDepth {
			s.compositionDepth = compDepth + 1
		}
		for _, sub := range schema.AllOf {
			if sub.GetSchema() != nil && sub.GetSchema().Ref == nil {
				s.walk(sub.GetSchema(), objectDepth, compDepth+1, path, seen)
			}
		}
	}

	// oneOf
	if len(schema.OneOf) > 0 {
		s.compositionFields["oneOf"] = true
		if compDepth+1 > s.compositionDepth {
			s.compositionDepth = compDepth + 1
		}
		site := UnionSite{
			Kind:             "oneOf",
			Width:            len(schema.OneOf),
			HasDiscriminator: schema.Discriminator != nil,
			Path:             path,
		}
		if site.Path == "" {
			site.Path = "root"
		}
		s.unionSites = append(s.unionSites, site)
		for _, sub := range schema.OneOf {
			if sub.GetSchema() != nil && sub.GetSchema().Ref == nil {
				s.walk(sub.GetSchema(), objectDepth, compDepth+1, path, seen)
			}
		}
	}

	// anyOf
	if len(schema.AnyOf) > 0 {
		s.compositionFields["anyOf"] = true
		if compDepth+1 > s.compositionDepth {
			s.compositionDepth = compDepth + 1
		}
		site := UnionSite{
			Kind:             "anyOf",
			Width:            len(schema.AnyOf),
			HasDiscriminator: schema.Discriminator != nil,
			Path:             path,
		}
		if site.Path == "" {
			site.Path = "root"
		}
		s.unionSites = append(s.unionSites, site)
		for _, sub := range schema.AnyOf {
			if sub.GetSchema() != nil && sub.GetSchema().Ref == nil {
				s.walk(sub.GetSchema(), objectDepth, compDepth+1, path, seen)
			}
		}
	}

	// Items
	if schema.Items != nil && schema.Items.GetSchema() != nil && schema.Items.GetSchema().Ref == nil {
		itemPath := path + "[]"
		if path == "" {
			itemPath = "[]"
		}
		s.walk(schema.Items.GetSchema(), objectDepth, compDepth, itemPath, seen)
	}

	// AdditionalProperties
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.GetSchema() != nil && schema.AdditionalProperties.GetSchema().Ref == nil {
		apPath := path + "{}"
		if path == "" {
			apPath = "{}"
		}
		s.walk(schema.AdditionalProperties.GetSchema(), objectDepth, compDepth, apPath, seen)
	}
}

// extractEdges discovers all outgoing edges from a schema to other component schemas.
// It recursively descends into inline sub-schemas (properties, composition branches)
// to find $ref targets at any depth.
func extractEdges(g *Graph, sourceID string, schema *oas3.Schema) {
	seen := make(map[*oas3.Schema]bool)
	extractEdgesRecursive(g, sourceID, schema, seen)
}

func extractEdgesRecursive(g *Graph, sourceID string, schema *oas3.Schema, seen map[*oas3.Schema]bool) {
	if schema == nil || seen[schema] {
		return
	}
	seen[schema] = true

	// Properties
	if schema.Properties != nil {
		for propName, propSchema := range schema.Properties.All() {
			if target := resolveRefTarget(propSchema); target != "" {
				isRequired := slices.Contains(schema.Required, propName)
				propSchemaObj := propSchema.GetSchema()
				isNullable := propSchemaObj != nil && (slices.Contains(schemaTypeValues(propSchemaObj), "null") || (propSchemaObj.Nullable != nil && *propSchemaObj.Nullable))
				g.addEdge(&Edge{
					From:       sourceID,
					To:         target,
					Kind:       EdgeProperty,
					FieldName:  propName,
					IsRequired: isRequired,
					IsNullable: isNullable,
				})
			} else if propSchema.GetSchema() != nil {
				// Inline schema — recurse into it to find nested $refs
				extractEdgesRecursive(g, sourceID, propSchema.GetSchema(), seen)
			}
			// Also check if property is an array with $ref items
			if propSchema.GetSchema() != nil && propSchema.GetSchema().Items != nil {
				if target := resolveRefTarget(propSchema.GetSchema().Items); target != "" {
					isRequired := slices.Contains(schema.Required, propName)
					g.addEdge(&Edge{
						From:       sourceID,
						To:         target,
						Kind:       EdgeItems,
						FieldName:  propName,
						IsRequired: isRequired,
						IsArray:    true,
					})
				} else if propSchema.GetSchema().Items.GetSchema() != nil {
					extractEdgesRecursive(g, sourceID, propSchema.GetSchema().Items.GetSchema(), seen)
				}
			}
		}
	}

	// Items (top-level array schema)
	if schema.Items != nil {
		if target := resolveRefTarget(schema.Items); target != "" {
			g.addEdge(&Edge{
				From:    sourceID,
				To:      target,
				Kind:    EdgeItems,
				IsArray: true,
			})
		} else if schema.Items.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, schema.Items.GetSchema(), seen)
		}
	}

	// allOf
	for i, sub := range schema.AllOf {
		if target := resolveRefTarget(sub); target != "" {
			g.addEdge(&Edge{
				From:  sourceID,
				To:    target,
				Kind:  EdgeAllOf,
				Index: i,
			})
		} else if sub.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, sub.GetSchema(), seen)
		}
	}

	// oneOf
	for i, sub := range schema.OneOf {
		if target := resolveRefTarget(sub); target != "" {
			g.addEdge(&Edge{
				From:  sourceID,
				To:    target,
				Kind:  EdgeOneOf,
				Index: i,
			})
		} else if sub.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, sub.GetSchema(), seen)
		}
	}

	// anyOf
	for i, sub := range schema.AnyOf {
		if target := resolveRefTarget(sub); target != "" {
			g.addEdge(&Edge{
				From:  sourceID,
				To:    target,
				Kind:  EdgeAnyOf,
				Index: i,
			})
		} else if sub.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, sub.GetSchema(), seen)
		}
	}

	// additionalProperties
	if schema.AdditionalProperties != nil {
		if target := resolveRefTarget(schema.AdditionalProperties); target != "" {
			g.addEdge(&Edge{
				From: sourceID,
				To:   target,
				Kind: EdgeAdditionalProperties,
			})
		} else if schema.AdditionalProperties.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, schema.AdditionalProperties.GetSchema(), seen)
		}
	}

	// not
	if schema.Not != nil {
		if target := resolveRefTarget(schema.Not); target != "" {
			g.addEdge(&Edge{
				From: sourceID,
				To:   target,
				Kind: EdgeNot,
			})
		} else if schema.Not.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, schema.Not.GetSchema(), seen)
		}
	}

	// prefixItems
	for i, sub := range schema.PrefixItems {
		if target := resolveRefTarget(sub); target != "" {
			g.addEdge(&Edge{
				From:  sourceID,
				To:    target,
				Kind:  EdgePrefixItems,
				Index: i,
			})
		} else if sub.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, sub.GetSchema(), seen)
		}
	}

	// if/then/else
	if schema.If != nil {
		if target := resolveRefTarget(schema.If); target != "" {
			g.addEdge(&Edge{From: sourceID, To: target, Kind: EdgeIf})
		} else if schema.If.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, schema.If.GetSchema(), seen)
		}
	}
	if schema.Then != nil {
		if target := resolveRefTarget(schema.Then); target != "" {
			g.addEdge(&Edge{From: sourceID, To: target, Kind: EdgeThen})
		} else if schema.Then.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, schema.Then.GetSchema(), seen)
		}
	}
	if schema.Else != nil {
		if target := resolveRefTarget(schema.Else); target != "" {
			g.addEdge(&Edge{From: sourceID, To: target, Kind: EdgeElse})
		} else if schema.Else.GetSchema() != nil {
			extractEdgesRecursive(g, sourceID, schema.Else.GetSchema(), seen)
		}
	}

	// DependentSchemas
	if schema.DependentSchemas != nil {
		for _, depSchema := range schema.DependentSchemas.All() {
			if target := resolveRefTarget(depSchema); target != "" {
				g.addEdge(&Edge{From: sourceID, To: target, Kind: EdgeProperty})
			} else if depSchema.GetSchema() != nil {
				extractEdgesRecursive(g, sourceID, depSchema.GetSchema(), seen)
			}
		}
	}

	// PatternProperties
	if schema.PatternProperties != nil {
		for _, ppSchema := range schema.PatternProperties.All() {
			if target := resolveRefTarget(ppSchema); target != "" {
				g.addEdge(&Edge{From: sourceID, To: target, Kind: EdgeAdditionalProperties})
			} else if ppSchema.GetSchema() != nil {
				extractEdgesRecursive(g, sourceID, ppSchema.GetSchema(), seen)
			}
		}
	}
}

// schemaTypeValues extracts the type values from a schema's Type field.
// Type is an EitherValue that is either []SchemaType (array) or SchemaType (single).
func schemaTypeValues(schema *oas3.Schema) []string {
	if schema.Type == nil {
		return nil
	}
	// Left = []SchemaType (array of types)
	if schema.Type.IsLeft() {
		left := schema.Type.LeftValue()
		result := make([]string, len(left))
		for i, t := range left {
			result[i] = string(t)
		}
		return result
	}
	// Right = SchemaType (single type)
	if schema.Type.IsRight() {
		right := schema.Type.RightValue()
		if right != "" {
			return []string{string(right)}
		}
	}
	return nil
}

// resolveRefTarget extracts the component schema name from a $ref if it points
// to #/components/schemas/. Returns empty string for inline schemas or external refs.
func resolveRefTarget(jsonSchema *oas3.JSONSchema[oas3.Referenceable]) string {
	if jsonSchema == nil {
		return ""
	}
	schema := jsonSchema.GetSchema()
	if schema == nil || schema.Ref == nil {
		return ""
	}

	ref := schema.Ref.String()
	const prefix = "#/components/schemas/"
	if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ""
}
