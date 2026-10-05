package oq_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/oq"
	"github.com/speakeasy-api/openapi/oq/expr"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_PipelineErrors_Error(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)
	for _, tt := range []struct{ query, message string }{
		{`unknown`, `unknown source: "unknown"`},
		{`schemas | where(name ==)`, `where expression error: unexpected token: ""`},
		{`schemas | let $x = name +`, `let expression error: unexpected token: ""`},
		{`path(Missing, Pet)`, `schema "Missing" not found`},
		{`path(Pet, Missing)`, `schema "Missing" not found`},
		{`schemas | path(Pet, Missing)`, `schema "Missing" not found`},
		{`schemas | shared-refs(2)`, "shared-refs requires operation rows, got schema rows\n  hint: operations | shared-refs(2)"},
	} {
		t.Run(tt.query, func(t *testing.T) {
			t.Parallel()
			_, err := oq.Execute(tt.query, g)
			require.EqualError(t, err, tt.message, "pipeline should report the failing stage")
		})
	}
}

func TestExecute_TypeMismatchedStages_ReturnEmpty(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)
	for _, query := range []string{
		`operations | refs`, `operations | properties`, `operations | properties(*)`,
		`operations | items`, `operations | members`, `operations | parent`,
		`operations | orphans`, `operations | leaves`, `operations | cross-tag`,
		`operations | duplicates`, `operations | to-operations`,
		`operations | additional-properties`, `operations | pattern-properties`,
		`schemas | to-schemas`, `schemas | parameters`, `schemas | responses`,
		`schemas | request-body`, `schemas | callbacks`, `schemas | links`,
		`schemas | operation`, `schemas | security`,
		`schemas | where(false) | shared-refs`, `schemas | where(false) | members`,
		`operations | group-by(method, operationId) | members`,
	} {
		t.Run(query, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(query, g)
			require.NoError(t, err, "unsupported row types should be skipped")
			assert.Empty(t, result.Rows, "navigation should not invent rows for unrelated objects")
		})
	}
}

func TestExecute_AbsentSourceMetadata_ReturnEmpty(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		g    *graph.SchemaGraph
	}{
		{"no index", &graph.SchemaGraph{}},
		{"no document", &graph.SchemaGraph{Index: &openapi.Index{}}},
		{"no components", &graph.SchemaGraph{Index: &openapi.Index{Doc: &openapi.OpenAPI{Servers: []*openapi.Server{nil}}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, source := range []string{"components", "security", "servers", "tags", "webhooks"} {
				result, err := oq.Execute(source, tt.g)
				require.NoError(t, err, "source %s should tolerate absent metadata", source)
				assert.Empty(t, result.Rows, "source %s should have no objects", source)
			}
		})
	}
}

func TestExecute_GlobalSecurityAndSparseComponents_Success(t *testing.T) {
	t.Parallel()
	scheme := &openapi.SecurityScheme{Type: openapi.SecuritySchemeTypeAPIKey}
	body := &openapi.RequestBody{}
	header := &openapi.Header{}
	parameter := &openapi.Parameter{Name: "valid"}
	response := &openapi.Response{}
	req := &openapi.SecurityRequirement{Map: *sequencedmap.New(sequencedmap.NewElem("known", []string{"read"}), sequencedmap.NewElem("missing", []string{"write"}))}
	doc := &openapi.OpenAPI{
		Security: []*openapi.SecurityRequirement{nil, req},
		Components: &openapi.Components{
			Parameters:      sequencedmap.New(sequencedmap.NewElem[string, *openapi.ReferencedParameter]("nil", nil), sequencedmap.NewElem("unresolved", &openapi.ReferencedParameter{}), sequencedmap.NewElem("valid", &openapi.ReferencedParameter{Object: parameter})),
			Responses:       sequencedmap.New(sequencedmap.NewElem[string, *openapi.ReferencedResponse]("nil", nil), sequencedmap.NewElem("unresolved", &openapi.ReferencedResponse{}), sequencedmap.NewElem("valid", &openapi.ReferencedResponse{Object: response})),
			RequestBodies:   sequencedmap.New(sequencedmap.NewElem[string, *openapi.ReferencedRequestBody]("nil", nil), sequencedmap.NewElem("unresolved", &openapi.ReferencedRequestBody{}), sequencedmap.NewElem("valid", &openapi.ReferencedRequestBody{Object: body})),
			Headers:         sequencedmap.New(sequencedmap.NewElem[string, *openapi.ReferencedHeader]("nil", nil), sequencedmap.NewElem("unresolved", &openapi.ReferencedHeader{}), sequencedmap.NewElem("valid", &openapi.ReferencedHeader{Object: header})),
			SecuritySchemes: sequencedmap.New(sequencedmap.NewElem[string, *openapi.ReferencedSecurityScheme]("nil", nil), sequencedmap.NewElem("unresolved", &openapi.ReferencedSecurityScheme{}), sequencedmap.NewElem("known", &openapi.ReferencedSecurityScheme{Object: scheme})),
		},
	}
	g := &graph.SchemaGraph{Index: &openapi.Index{Doc: doc}}
	result, err := oq.Execute("components", g)
	require.NoError(t, err, "programmatically built components should skip missing objects")
	assert.Equal(t, []oq.Row{
		{Kind: oq.ParameterResult, Parameter: parameter, ComponentKey: "valid", SourceOpIdx: -1},
		{Kind: oq.ResponseResult, Response: response, ComponentKey: "valid", SourceOpIdx: -1},
		{Kind: oq.RequestBodyResult, RequestBody: body, ComponentKey: "valid", SourceOpIdx: -1},
		{Kind: oq.HeaderResult, Header: header, HeaderName: "valid", SourceOpIdx: -1},
		{Kind: oq.SecuritySchemeResult, SecurityScheme: scheme, SchemeName: "known", SourceOpIdx: -1},
	}, result.Rows, "component sources should preserve order and mark the lack of source operation")
	security, err := oq.Execute("security", g)
	require.NoError(t, err, "global security should resolve known schemes")
	assert.Equal(t, []oq.Row{
		{Kind: oq.SecurityRequirementResult, SchemeName: "known", SecurityScheme: scheme, Scopes: []string{"read"}, SourceOpIdx: -1},
		{Kind: oq.SecurityRequirementResult, SchemeName: "missing", Scopes: []string{"write"}, SourceOpIdx: -1},
	}, security.Rows, "unknown schemes should retain their requirements without a resolved object")
	for _, query := range []string{"components | operation", "security | operation"} {
		result, err := oq.Execute(query, g)
		require.NoError(t, err, "objects without a source operation should be skipped")
		assert.Empty(t, result.Rows, "global objects should not back-navigate to operation zero")
	}
}

func TestExecute_ComposedArraysAndReversePaths_Success(t *testing.T) {
	t.Parallel()
	const spec = `openapi: 3.1.0
info: {title: Composition, version: '1'}
paths: {}
components:
  schemas:
    Item:
      type: object
      properties:
        id: {type: string}
    Nested:
      allOf:
        - allOf:
            - $ref: '#/components/schemas/Item'
    Array:
      allOf:
        - type: array
          items: {$ref: '#/components/schemas/Item'}
    Page:
      allOf:
        - type: object
          properties:
            metadata: {type: string}
            items:
              type: array
              items: {$ref: '#/components/schemas/Item'}
    Isolated: {type: boolean}
`
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(spec), openapi.WithSkipValidation())
	require.NoError(t, err, "composition document should unmarshal")
	g := graph.Build(ctx, openapi.BuildIndex(ctx, doc, references.ResolveOptions{RootDocument: doc, TargetDocument: doc, TargetLocation: "composition.yaml"}))

	for _, name := range []string{"Array", "Page"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(`schemas | where(isComponent and name == "`+name+`") | items`, g)
			require.NoError(t, err, "items should navigate through allOf composition")
			require.Len(t, result.Rows, 1, "composed array should have one item row")
			row := result.Rows[0]
			assert.Equal(t, "items", row.EdgeKind, "item edge should retain its kind")
			assert.Equal(t, name, row.Traversal, "composition should attribute items to the input schema")
			assert.Equal(t, name, row.Seed, "composition should preserve the traversal seed")
			props, err := oq.Execute(`schemas | where(isComponent and name == "`+name+`") | items | properties`, g)
			require.NoError(t, err, "array element reference should resolve to its properties")
			require.Len(t, props.Rows, 1, "item schema should contain exactly one property")
			assert.Equal(t, "id", props.Rows[0].EdgeLabel, "item property should be id")
		})
	}
	props, err := oq.Execute(`schemas | where(isComponent and name == "Nested") | properties`, g)
	require.NoError(t, err, "nested allOf should flatten properties")
	require.Len(t, props.Rows, 1, "nested allOf should expose one property")
	assert.Equal(t, "Nested", props.Rows[0].Traversal, "flattened property should belong to outer composition")
	assert.Equal(t, "id", props.Rows[0].EdgeLabel, "nested property should be id")

	path, err := oq.Execute("path(Item, Page)", g)
	require.NoError(t, err, "path may travel backwards through structural edges")
	require.NotEmpty(t, path.Rows, "reverse path should connect the component schemas")
	assert.Equal(t, expr.StringVal("Item"), oq.FieldValuePublic(path.Rows[0], "name", g), "path should begin at Item")
	assert.Equal(t, expr.StringVal("Page"), oq.FieldValuePublic(path.Rows[len(path.Rows)-1], "name", g), "path should end at Page")
	for i, row := range path.Rows[1:] {
		assert.Equal(t, "←", row.Direction, "all hops should traverse incoming edges")
		assert.Equal(t, i+1, row.Hops, "path should annotate consecutive hops")
	}
	isolated, err := oq.Execute("path(Item, Isolated)", g)
	require.NoError(t, err, "disconnected named nodes should not be an error")
	assert.Empty(t, isolated.Rows, "disconnected nodes should have no path")
	same, err := oq.Execute("path(Item, Item)", g)
	require.NoError(t, err, "same-node path should succeed")
	item, ok := g.SchemaByName("Item")
	require.True(t, ok, "Item should be registered")
	assert.Equal(t, []oq.Row{{Kind: oq.SchemaResult, SchemaIdx: int(item.ID)}}, same.Rows, "same-node path should contain only its seed")
}

func TestExecute_BindingsAndProjectedIdentity_Success(t *testing.T) {
	t.Parallel()
	g := &graph.SchemaGraph{Schemas: []graph.SchemaNode{
		{Name: "first", Type: "object", Depth: 1},
		{Name: "second", Type: "object", Depth: 1},
		{Name: "third", Type: "object", Depth: 2},
	}}
	result, err := oq.Execute(`schemas | select type, depth | format json | to-yaml | unique | let $a = depth | let $b = $a + 1 | where(depth == $b) | last(5) | sample(5)`, g)
	require.NoError(t, err, "bindings should survive consecutive let stages")
	assert.Equal(t, []oq.Row{{Kind: oq.SchemaResult, SchemaIdx: 2}}, result.Rows, "projected identity should consider every field and retain the depth-2 row")
	assert.Equal(t, []string{"type", "depth"}, result.Fields, "selection should survive derived results")
	assert.Equal(t, "json", result.FormatHint, "format preference should survive derived results")
	assert.True(t, result.EmitYAML, "YAML preference should survive derived results")
	grouped, err := oq.Execute(`schemas | group-by(type) | members`, g)
	require.NoError(t, err, "group members should resolve schema names")
	assert.Equal(t, []oq.Row{{Kind: oq.SchemaResult, SchemaIdx: 0}, {Kind: oq.SchemaResult, SchemaIdx: 1}, {Kind: oq.SchemaResult, SchemaIdx: 2}}, grouped.Rows, "member expansion should retain schema order")
}
