package oq_test

import (
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/oq"
	"github.com/speakeasy-api/openapi/oq/expr"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestGraph(t *testing.T) *graph.SchemaGraph {
	t.Helper()

	f, err := os.Open("testdata/petstore.yaml")
	require.NoError(t, err)
	defer f.Close()

	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, f, openapi.WithSkipValidation())
	require.NoError(t, err)
	require.NotNil(t, doc)

	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "testdata/petstore.yaml",
	})

	return graph.Build(ctx, idx)
}

func TestParse_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"simple source", "schemas"},
		{"components source", "schemas.components"},
		{"inline source", "schemas.inline"},
		{"operations source", "operations"},
		{"sort", "schemas | sort depth desc"},
		{"take", "schemas | take 5"},
		{"where", "schemas | where depth > 3"},
		{"select", "schemas | select name, depth"},
		{"count", "schemas | count"},
		{"unique", "schemas | unique"},
		{"group-by", "schemas | group-by hash"},
		{"refs-out", "schemas | refs-out"},
		{"refs-in", "schemas | refs-in"},
		{"reachable", "schemas | reachable"},
		{"ancestors", "schemas | ancestors"},
		{"properties", "schemas | properties"},
		{"union-members", "schemas | union-members"},
		{"items", "schemas | items"},
		{"ops", "schemas | ops"},
		{"schemas from ops", "operations | schemas"},
		{"connected", "schemas.components | where name == \"Pet\" | connected"},
		{"blast-radius", "schemas.components | where name == \"Pet\" | blast-radius"},
		{"neighbors", "schemas.components | where name == \"Pet\" | neighbors 2"},
		{"orphans", "schemas.components | orphans"},
		{"leaves", "schemas.components | leaves"},
		{"cycles", "schemas | cycles"},
		{"clusters", "schemas.components | clusters"},
		{"tag-boundary", "schemas | tag-boundary"},
		{"shared-refs", "operations | take 2 | shared-refs"},
		{"full pipeline", "schemas.components | where depth > 0 | sort depth desc | take 5 | select name, depth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stages, err := oq.Parse(tt.query)
			require.NoError(t, err)
			assert.NotEmpty(t, stages)
		})
	}
}

func TestParse_Error(t *testing.T) {
	t.Parallel()

	_, err := oq.Parse("")
	require.Error(t, err)

	_, err = oq.Parse("schemas | unknown_stage")
	require.Error(t, err)

	_, err = oq.Parse("schemas | take abc")
	require.Error(t, err)
}

func TestExecute_SchemasCount_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | count", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount)
	assert.Positive(t, result.Count)
}

func TestExecute_ComponentSchemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | select name", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Check that we have the expected component schemas
	names := collectNames(result, g)
	assert.Contains(t, names, "Pet")
	assert.Contains(t, names, "Owner")
	assert.Contains(t, names, "Address")
	assert.Contains(t, names, "Error")
	assert.Contains(t, names, "Shape")
	assert.Contains(t, names, "Unused")
}

func TestExecute_Where_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where type == "object" | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet")
	assert.Contains(t, names, "Owner")
}

func TestExecute_WhereInDegree_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Unused schema has no incoming references (from other schemas in components)
	result, err := oq.Execute(`schemas.components | where in_degree == 0 | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Unused should have no references from other schemas
	assert.Contains(t, names, "Unused")
}

func TestExecute_Sort_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | sort property_count desc | take 3 | select name, property_count", g)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Rows), 3)
}

func TestExecute_Reachable_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Pet" | reachable | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Pet references Owner, Owner references Address
	assert.Contains(t, names, "Owner")
	assert.Contains(t, names, "Address")
}

func TestExecute_Ancestors_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Address" | ancestors | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Address is referenced by Owner, which is referenced by Pet
	assert.Contains(t, names, "Owner")
}

func TestExecute_Properties_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Pet" | properties | select name`, g)
	require.NoError(t, err)
	// Pet has 4 properties: id, name, tag, owner
	assert.NotEmpty(t, result.Rows)
}

func TestExecute_UnionMembers_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Shape" | union-members | select name`, g)
	require.NoError(t, err)
	// Shape has oneOf with Circle and Square
	names := collectNames(result, g)
	assert.Contains(t, names, "Circle")
	assert.Contains(t, names, "Square")
}

func TestExecute_Operations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | select name, method, path", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestExecute_OperationSchemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where operation_id == "listPets" | schemas | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet")
}

func TestExecute_GroupBy_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | group-by type`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups)
}

func TestExecute_Unique_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | unique", g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Check no duplicates
	seen := make(map[string]bool)
	for _, n := range names {
		assert.False(t, seen[n], "duplicate: %s", n)
		seen[n] = true
	}
}

func TestExecute_SchemasToOps_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Pet" | ops | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestFormatTable_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | take 3 | select name, type", g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "name")
	assert.Contains(t, table, "type")
	assert.NotEmpty(t, table)
}

func TestFormatJSON_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | take 3 | select name, type", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.True(t, strings.HasPrefix(json, "["))
	assert.True(t, strings.HasSuffix(json, "]"))
}

func TestFormatTable_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | count", g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.NotEmpty(t, table)
}

func TestFormatTable_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "NonExistent"`, g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.Equal(t, "(empty)", table)
}

func TestExecute_MatchesExpression_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name matches ".*Error.*" | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Error")
}

func TestExecute_SortAsc_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | sort name asc | select name", g)
	require.NoError(t, err)

	names := collectNames(result, g)
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i])
	}
}

func TestExecute_Explain_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | where depth > 5 | sort depth desc | take 10 | explain", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Source: schemas.components")
	assert.Contains(t, result.Explain, "Filter: where depth > 5")
	assert.Contains(t, result.Explain, "Sort: depth descending")
	assert.Contains(t, result.Explain, "Limit: take 10")
}

func TestExecute_Fields_Schemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | fields", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "name")
	assert.Contains(t, result.Explain, "depth")
	assert.Contains(t, result.Explain, "property_count")
	assert.Contains(t, result.Explain, "is_component")
}

func TestExecute_Fields_Operations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | fields", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "method")
	assert.Contains(t, result.Explain, "operation_id")
	assert.Contains(t, result.Explain, "schema_count")
	assert.Contains(t, result.Explain, "tag")
	assert.Contains(t, result.Explain, "deprecated")
}

func TestExecute_Head_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | head 3", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3)
}

func TestExecute_Sample_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | sample 3", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3)

	// Running sample again should produce the same result (deterministic)
	result2, err := oq.Execute("schemas.components | sample 3", g)
	require.NoError(t, err)
	assert.Len(t, result2.Rows, len(result.Rows))
}

func TestExecute_Path_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | path Pet Address | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	names := collectNames(result, g)
	// Path should include Pet, something in between, and Address
	assert.Equal(t, "Pet", names[0])
	assert.Equal(t, "Address", names[len(names)-1])
}

func TestExecute_Path_NotFound_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Unused has no outgoing edges to reach Pet
	result, err := oq.Execute(`schemas | path Unused Pet | select name`, g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)
}

func TestExecute_Top_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | top 3 property_count | select name, property_count", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3)

	// Verify descending order
	for i := 1; i < len(result.Rows); i++ {
		prev := oq.FieldValuePublic(result.Rows[i-1], "property_count", g)
		curr := oq.FieldValuePublic(result.Rows[i], "property_count", g)
		assert.GreaterOrEqual(t, prev.Int, curr.Int)
	}
}

func TestExecute_Bottom_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | bottom 3 property_count | select name, property_count", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3)

	// Verify ascending order
	for i := 1; i < len(result.Rows); i++ {
		prev := oq.FieldValuePublic(result.Rows[i-1], "property_count", g)
		curr := oq.FieldValuePublic(result.Rows[i], "property_count", g)
		assert.LessOrEqual(t, prev.Int, curr.Int)
	}
}

func TestExecute_Format_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | take 3 | format json", g)
	require.NoError(t, err)
	assert.Equal(t, "json", result.FormatHint)
}

func TestFormatMarkdown_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | take 3 | select name, type", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| name")
	assert.Contains(t, md, "| --- |")
}

func TestExecute_OperationTag_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | select name, tag, parameter_count", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestParse_NewStages_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"explain", "schemas | explain"},
		{"fields", "schemas | fields"},
		{"head", "schemas | head 5"},
		{"sample", "schemas | sample 10"},
		{"path", `schemas | path "User" "Order"`},
		{"path unquoted", "schemas | path User Order"},
		{"top", "schemas | top 5 depth"},
		{"bottom", "schemas | bottom 5 depth"},
		{"format", "schemas | format json"},
		{"format markdown", "schemas | format markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stages, err := oq.Parse(tt.query)
			require.NoError(t, err)
			assert.NotEmpty(t, stages)
		})
	}
}

func TestExecute_RefsOut_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Pet" | refs-out | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestExecute_RefsIn_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Owner" | refs-in | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestExecute_Items_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// listPets response includes an array with items
	result, err := oq.Execute(`schemas | where type == "array" | items | select name`, g)
	require.NoError(t, err)
	// May or may not have results depending on spec, but should not error
	assert.NotNil(t, result)
}

func TestExecute_Connected_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Start from Pet, connected should return schemas and operations in the same component
	result, err := oq.Execute(`schemas.components | where name == "Pet" | connected`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Should have both schema and operation rows
	hasSchema := false
	hasOp := false
	for _, row := range result.Rows {
		if row.Kind == oq.SchemaResult {
			hasSchema = true
		}
		if row.Kind == oq.OperationResult {
			hasOp = true
		}
	}
	assert.True(t, hasSchema, "connected should include schema nodes")
	assert.True(t, hasOp, "connected should include operation nodes")
}

func TestExecute_Connected_FromOps_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Start from an operation, connected should also find schemas
	result, err := oq.Execute(`operations | take 1 | connected`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	hasSchema := false
	for _, row := range result.Rows {
		if row.Kind == oq.SchemaResult {
			hasSchema = true
		}
	}
	assert.True(t, hasSchema, "connected from operation should include schema nodes")
}

func TestExecute_EdgeAnnotations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Pet" | refs-out | select name, edge_kind, edge_label, edge_from`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Every row should have edge annotations
	for _, row := range result.Rows {
		kind := oq.FieldValuePublic(row, "edge_kind", g)
		assert.NotEmpty(t, kind.Str, "edge_kind should be set")
		from := oq.FieldValuePublic(row, "edge_from", g)
		assert.Equal(t, "Pet", from.Str)
	}
}

func TestExecute_BlastRadius_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Pet" | blast-radius`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Should include both schemas and operations
	hasSchema := false
	hasOp := false
	for _, row := range result.Rows {
		if row.Kind == oq.SchemaResult {
			hasSchema = true
		}
		if row.Kind == oq.OperationResult {
			hasOp = true
		}
	}
	assert.True(t, hasSchema, "blast-radius should include schemas")
	assert.True(t, hasOp, "blast-radius should include operations")
}

func TestExecute_Neighbors_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "Pet" | neighbors 1`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Depth-1 neighbors should include seed + direct refs in both directions
	names := make(map[string]bool)
	for _, row := range result.Rows {
		n := oq.FieldValuePublic(row, "name", g)
		names[n.Str] = true
	}
	assert.True(t, names["Pet"], "neighbors should include the seed node")
}

func TestExecute_Orphans_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | orphans | select name`, g)
	require.NoError(t, err)
	// Result may be empty if all schemas are referenced, that's fine
	assert.NotNil(t, result)
}

func TestExecute_Leaves_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | leaves | select name, out_degree`, g)
	require.NoError(t, err)
	// All returned rows should have out_degree == 0
	for _, row := range result.Rows {
		od := oq.FieldValuePublic(row, "out_degree", g)
		assert.Equal(t, 0, od.Int)
	}
}

func TestExecute_Cycles_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | cycles`, g)
	require.NoError(t, err)
	// Returns groups — may be empty if no cycles in petstore
	assert.NotNil(t, result)
}

func TestExecute_Clusters_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | clusters`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups)

	// Total names across all clusters should equal component count
	total := 0
	for _, grp := range result.Groups {
		total += grp.Count
	}
	// Count component schemas
	compCount, err := oq.Execute(`schemas.components | count`, g)
	require.NoError(t, err)
	assert.Equal(t, compCount.Count, total)
}

func TestExecute_TagBoundary_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | tag-boundary | select name, tag_count`, g)
	require.NoError(t, err)
	// All returned rows should have tag_count > 1
	for _, row := range result.Rows {
		tc := oq.FieldValuePublic(row, "tag_count", g)
		assert.Greater(t, tc.Int, 1)
	}
}

func TestExecute_SharedRefs_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | shared-refs | select name`, g)
	require.NoError(t, err)
	// Schemas shared by ALL operations
	assert.NotNil(t, result)
}

func TestExecute_OpCount_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | sort op_count desc | take 3 | select name, op_count`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestFormatTable_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | group-by type", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups)

	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "count=")
}

func TestFormatJSON_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | group-by type", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "\"key\"")
	assert.Contains(t, json, "\"count\"")
}

func TestFormatMarkdown_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | group-by type", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| Key |")
}

func TestExecute_InlineSource_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.inline | count", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount)
}

func TestExecute_SchemaFields_Coverage(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Select all schema fields to cover fieldValue branches
	result, err := oq.Execute("schemas.components | take 1 | select name, type, depth, in_degree, out_degree, union_width, property_count, is_component, is_inline, is_circular, has_ref, hash, path", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	table := oq.FormatTable(result, g)
	assert.NotEmpty(t, table)

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "\"name\"")
}

func TestExecute_OperationFields_Coverage(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Select all operation fields to cover fieldValue branches
	result, err := oq.Execute("operations | take 1 | select name, method, path, operation_id, schema_count, component_count, tag, parameter_count, deprecated, description, summary", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestFormatJSON_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "NonExistent"`, g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.Equal(t, "[]", json)
}

func TestFormatMarkdown_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "NonExistent"`, g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Equal(t, "(empty)", md)
}

func TestFormatJSON_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | count", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.NotEmpty(t, json)
}

func TestFormatToon_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | take 3 | select name, type", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "results[3]{name,type}:")
	assert.Contains(t, toon, "object")
}

func TestFormatToon_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | count", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "count:")
}

func TestFormatToon_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | group-by type", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "groups[")
	assert.Contains(t, toon, "{key,count,names}:")
}

func TestFormatToon_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas.components | where name == "NonExistent"`, g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Equal(t, "results[0]:\n", toon)
}

func TestFormatToon_Escaping_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Paths contain special chars like / that don't need escaping,
	// but hash values and paths are good coverage
	result, err := oq.Execute("schemas.components | take 1 | select name, hash, path", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "results[1]{name,hash,path}:")
}

func TestFormatMarkdown_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | count", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.NotEmpty(t, md)
}

func TestExecute_Explain_AllStages_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Cover more stage descriptions in explain
	tests := []struct {
		name    string
		query   string
		expects []string
	}{
		{
			"explain with unique and count",
			"schemas.components | unique | count | explain",
			[]string{"Unique:", "Count:"},
		},
		{
			"explain with group-by",
			"schemas.components | group-by type | explain",
			[]string{"Group: group-by"},
		},
		{
			"explain with traversals",
			"schemas.components | where name == \"Pet\" | refs-out | explain",
			[]string{"Traverse: outgoing references"},
		},
		{
			"explain with refs-in",
			"schemas.components | where name == \"Owner\" | refs-in | explain",
			[]string{"Traverse: incoming references"},
		},
		{
			"explain with reachable",
			"schemas.components | where name == \"Pet\" | reachable | explain",
			[]string{"Traverse: all reachable"},
		},
		{
			"explain with ancestors",
			"schemas.components | where name == \"Address\" | ancestors | explain",
			[]string{"Traverse: all ancestor"},
		},
		{
			"explain with properties",
			"schemas.components | where name == \"Pet\" | properties | explain",
			[]string{"Traverse: property children"},
		},
		{
			"explain with union-members",
			"schemas.components | where name == \"Shape\" | union-members | explain",
			[]string{"Traverse: union members"},
		},
		{
			"explain with items",
			"schemas | where type == \"array\" | items | explain",
			[]string{"Traverse: array items"},
		},
		{
			"explain with ops",
			"schemas.components | where name == \"Pet\" | ops | explain",
			[]string{"Navigate: schemas to operations"},
		},
		{
			"explain with schemas from ops",
			"operations | schemas | explain",
			[]string{"Navigate: operations to schemas"},
		},
		{
			"explain with sample",
			"schemas.components | sample 3 | explain",
			[]string{"Sample: random 3"},
		},
		{
			"explain with path",
			"schemas | path Pet Address | explain",
			[]string{"Path: shortest path from Pet to Address"},
		},
		{
			"explain with top",
			"schemas.components | top 3 depth | explain",
			[]string{"Top: 3 by depth"},
		},
		{
			"explain with bottom",
			"schemas.components | bottom 3 depth | explain",
			[]string{"Bottom: 3 by depth"},
		},
		{
			"explain with format",
			"schemas.components | format json | explain",
			[]string{"Format: json"},
		},
		{
			"explain with connected",
			"schemas.components | where name == \"Pet\" | connected | explain",
			[]string{"Traverse: full connected"},
		},
		{
			"explain with blast-radius",
			"schemas.components | where name == \"Pet\" | blast-radius | explain",
			[]string{"Traverse: blast radius"},
		},
		{
			"explain with neighbors",
			"schemas.components | where name == \"Pet\" | neighbors 2 | explain",
			[]string{"Traverse: bidirectional neighbors within 2"},
		},
		{
			"explain with orphans",
			"schemas.components | orphans | explain",
			[]string{"Filter: schemas with no incoming"},
		},
		{
			"explain with leaves",
			"schemas.components | leaves | explain",
			[]string{"Filter: schemas with no outgoing"},
		},
		{
			"explain with cycles",
			"schemas | cycles | explain",
			[]string{"Analyze: strongly connected"},
		},
		{
			"explain with clusters",
			"schemas.components | clusters | explain",
			[]string{"Analyze: weakly connected"},
		},
		{
			"explain with tag-boundary",
			"schemas | tag-boundary | explain",
			[]string{"Filter: schemas used by operations across multiple"},
		},
		{
			"explain with shared-refs",
			"operations | shared-refs | explain",
			[]string{"Analyze: schemas shared"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(tt.query, g)
			require.NoError(t, err)
			for _, exp := range tt.expects {
				assert.Contains(t, result.Explain, exp)
			}
		})
	}
}

func TestExecute_FieldValue_EdgeCases(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test operation fields that require nil checks
	result, err := oq.Execute("operations | take 1 | select name, tag, parameter_count, deprecated, description, summary", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Test edge fields on non-traversal rows (should be empty strings)
	result, err = oq.Execute("schemas.components | take 1 | select name, edge_kind, edge_label, edge_from", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
	edgeKind := oq.FieldValuePublic(result.Rows[0], "edge_kind", g)
	assert.Equal(t, "", edgeKind.Str)

	// Test tag_count field
	result, err = oq.Execute("schemas.components | take 1 | select name, tag_count", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Test op_count field
	result, err = oq.Execute("schemas.components | take 1 | select name, op_count", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Test unknown field returns null (KindNull == 0)
	v := oq.FieldValuePublic(result.Rows[0], "nonexistent_field", g)
	assert.Equal(t, expr.KindNull, v.Kind)
}

func TestExecute_Cycles_NoCycles(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Petstore has no cycles, so cycles should return empty groups
	result, err := oq.Execute("schemas | cycles", g)
	require.NoError(t, err)
	assert.Empty(t, result.Groups, "petstore should have no cycles")
}

func TestExecute_SharedRefs_AllOps(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// shared-refs with all operations — returns schemas shared by all operations
	result, err := oq.Execute("operations | shared-refs | select name", g)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFormatToon_SpecialChars(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test TOON format with bool and int fields to cover toonValue branches
	result, err := oq.Execute("schemas.components | take 1 | select name, depth, is_component", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon)
	assert.Contains(t, toon, "results[1]")
}

func TestFormatJSON_Operations(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | take 2 | select name, method, path", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.True(t, strings.HasPrefix(json, "["))
	assert.Contains(t, json, "\"name\"")
	assert.Contains(t, json, "\"method\"")
}

func TestFormatMarkdown_Operations(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | take 2 | select name, method", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| name")
	assert.Contains(t, md, "| method")
}

func TestParse_Error_MoreCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"empty query", ""},
		{"unknown stage", "schemas | bogus_stage"},
		{"take non-integer", "schemas | take abc"},
		{"sample non-integer", "schemas | sample xyz"},
		{"head non-integer", "schemas | head xyz"},
		{"neighbors non-integer", "schemas | neighbors abc"},
		{"top missing field", "schemas | top 5"},
		{"bottom missing field", "schemas | bottom 5"},
		{"path missing args", "schemas | path"},
		{"path one arg", "schemas | path Pet"},
		{"where empty expr", "schemas | where"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := oq.Parse(tt.query)
			assert.Error(t, err)
		})
	}
}

func TestParse_MoreStages_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"format table", "schemas | format table"},
		{"format toon", "schemas | format toon"},
		{"sort asc explicit", "schemas | sort name asc"},
		{"sort default asc", "schemas | sort name"},
		{"select single field", "schemas | select name"},
		{"select many fields", "schemas | select name, type, depth, in_degree"},
		{"where with string", `schemas | where name == "Pet"`},
		{"where with bool", "schemas | where is_component"},
		{"where with not", "schemas | where not is_inline"},
		{"where with has", "schemas | where has(hash)"},
		{"where with matches", `schemas | where name matches ".*Pet.*"`},
		{"path quoted", `schemas | path "Pet" "Address"`},
		{"shared-refs stage", "operations | take 2 | shared-refs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stages, err := oq.Parse(tt.query)
			require.NoError(t, err)
			assert.NotEmpty(t, stages)
		})
	}
}

func TestExecute_WhereAndOr_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test compound where expressions
	result, err := oq.Execute(`schemas.components | where depth > 0 and is_component`, g)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = oq.Execute(`schemas.components | where depth > 100 or is_component`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "or should match is_component=true schemas")
}

func TestExecute_SortStringField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Sort by string field
	result, err := oq.Execute("schemas.components | sort type asc | select name, type", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestExecute_GroupBy_Type_Details(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | group-by type", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups)

	// Each group should have Count and Names
	for _, grp := range result.Groups {
		assert.Positive(t, grp.Count)
		assert.Len(t, grp.Names, grp.Count)
	}
}

func TestFormatMarkdown_Groups_Details(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | group-by type", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| Key |")
	assert.Contains(t, md, "| Count |")
}

func TestFormatJSON_Explain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | explain", g)
	require.NoError(t, err)

	// All formats should handle explain
	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "Source: schemas")

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "Source: schemas")

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "Source: schemas")

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "Source: schemas")
}

func TestExecute_Leaves_AllZeroOutDegree(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas.components | leaves | select name, out_degree", g)
	require.NoError(t, err)

	// Verify leaves are leaf nodes
	for _, row := range result.Rows {
		od := oq.FieldValuePublic(row, "out_degree", g)
		assert.Equal(t, 0, od.Int, "leaves should have 0 out_degree")
	}
}

func TestExecute_OperationsTraversals(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Operations going to schemas and back
	result, err := oq.Execute("operations | take 1 | schemas | select name", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Schema to operations roundtrip
	result, err = oq.Execute("schemas.components | where name == \"Pet\" | ops | select name", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func loadCyclicGraph(t *testing.T) *graph.SchemaGraph {
	t.Helper()

	f, err := os.Open("testdata/cyclic.yaml")
	require.NoError(t, err)
	defer f.Close()

	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, f, openapi.WithSkipValidation())
	require.NoError(t, err)
	require.NotNil(t, doc)

	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "testdata/cyclic.yaml",
	})

	return graph.Build(ctx, idx)
}

func TestExecute_Cycles_WithCyclicSpec(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// NodeA -> NodeB -> NodeA is a cycle
	result, err := oq.Execute("schemas | cycles", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "cyclic spec should have cycles")

	// Format the groups
	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "cycle-")

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "cycle-")
}

func TestExecute_CyclicSpec_EdgeAnnotations(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Test refs-out to cover edgeKindString branches
	result, err := oq.Execute(`schemas.components | where name == "NodeA" | refs-out | select name, edge_kind, edge_label`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Collect edge kinds
	edgeKinds := make(map[string]bool)
	for _, row := range result.Rows {
		k := oq.FieldValuePublic(row, "edge_kind", g)
		edgeKinds[k.Str] = true
	}
	// NodeA has properties, allOf, anyOf, items etc.
	assert.True(t, edgeKinds["property"], "should have property edges")
}

func TestExecute_CyclicSpec_IsCircular(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	result, err := oq.Execute("schemas.components | where is_circular | select name", g)
	require.NoError(t, err)
	names := collectNames(result, g)
	assert.Contains(t, names, "NodeA")
	assert.Contains(t, names, "NodeB")
}

func TestExecute_CyclicSpec_DeprecatedOp(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// The listNodes operation is deprecated with tags, summary, and description
	result, err := oq.Execute("operations | select name, deprecated, summary, description, tag, parameter_count", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	dep := oq.FieldValuePublic(result.Rows[0], "deprecated", g)
	assert.True(t, dep.Bool, "listNodes should be deprecated")

	summary := oq.FieldValuePublic(result.Rows[0], "summary", g)
	assert.Equal(t, "List all nodes", summary.Str)

	desc := oq.FieldValuePublic(result.Rows[0], "description", g)
	assert.NotEmpty(t, desc.Str)

	tag := oq.FieldValuePublic(result.Rows[0], "tag", g)
	assert.Equal(t, "nodes", tag.Str)
}

func TestExecute_ToonFormat_WithBoolAndInt(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Select fields that cover all toonValue branches (string, int, bool)
	result, err := oq.Execute("schemas.components | take 1 | select name, depth, is_circular", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon)
}

func TestExecute_ToonEscape_SpecialChars(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// path fields contain "/" which doesn't need quoting, but let's cover the formatter
	result, err := oq.Execute("schemas | take 3 | select path", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon)
}

func TestFormatToon_Explain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where depth > 0 | explain", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "Source: schemas")
}

func TestFormatMarkdown_Explain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | explain", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "Source: schemas")
}

// collectNames extracts the "name" field from all rows in the result.
func collectNames(result *oq.Result, g *graph.SchemaGraph) []string {
	var names []string
	for _, row := range result.Rows {
		v := oq.FieldValuePublic(row, "name", g)
		names = append(names, v.Str)
	}
	return names
}
