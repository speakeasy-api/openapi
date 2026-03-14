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
		{"components source", "schemas | select(is_component)"},
		{"inline source", "schemas | select(is_inline)"},
		{"operations source", "operations"},
		{"sort_by", "schemas | sort_by(depth; desc)"},
		{"first", "schemas | first(5)"},
		{"select", "schemas | select(depth > 3)"},
		{"pick", "schemas | pick name, depth"},
		{"length", "schemas | length"},
		{"unique", "schemas | unique"},
		{"group_by", "schemas | group_by(hash)"},
		{"refs-out", "schemas | refs-out"},
		{"refs-in", "schemas | refs-in"},
		{"reachable", "schemas | reachable"},
		{"ancestors", "schemas | ancestors"},
		{"properties", "schemas | properties"},
		{"union-members", "schemas | union-members"},
		{"items", "schemas | items"},
		{"ops", "schemas | ops"},
		{"schemas from ops", "operations | schemas"},
		{"connected", "schemas | select(is_component) | select(name == \"Pet\") | connected"},
		{"blast-radius", "schemas | select(is_component) | select(name == \"Pet\") | blast-radius"},
		{"neighbors", "schemas | select(is_component) | select(name == \"Pet\") | neighbors 2"},
		{"orphans", "schemas | select(is_component) | orphans"},
		{"leaves", "schemas | select(is_component) | leaves"},
		{"cycles", "schemas | cycles"},
		{"clusters", "schemas | select(is_component) | clusters"},
		{"tag-boundary", "schemas | tag-boundary"},
		{"shared-refs", "operations | first(2) | shared-refs"},
		{"full pipeline", "schemas | select(is_component) | select(depth > 0) | sort_by(depth; desc) | first(5) | pick name, depth"},
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

	result, err := oq.Execute("schemas | length", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount, "should be a count result")
	assert.Positive(t, result.Count, "count should be positive")
}

func TestExecute_ComponentSchemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | pick name", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have component schema rows")

	// Check that we have the expected component schemas
	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "should include Pet schema")
	assert.Contains(t, names, "Owner", "should include Owner schema")
	assert.Contains(t, names, "Address", "should include Address schema")
	assert.Contains(t, names, "Error", "should include Error schema")
	assert.Contains(t, names, "Shape", "should include Shape schema")
	assert.Contains(t, names, "Unused", "should include Unused schema")
}

func TestExecute_Where_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(type == "object") | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "should include Pet schema")
	assert.Contains(t, names, "Owner", "should include Owner schema")
}

func TestExecute_WhereInDegree_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Unused schema has no incoming references (from other schemas in components)
	result, err := oq.Execute(`schemas | select(is_component) | select(in_degree == 0) | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Unused should have no references from other schemas
	assert.Contains(t, names, "Unused", "should include Unused schema with in_degree 0")
}

func TestExecute_Sort_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | sort_by(property_count; desc) | first(3) | pick name, property_count", g)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Rows), 3, "should return at most 3 rows")
}

func TestExecute_Reachable_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | reachable | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Pet references Owner, Owner references Address
	assert.Contains(t, names, "Owner", "Pet should reach Owner")
	assert.Contains(t, names, "Address", "Pet should reach Address")
}

func TestExecute_Ancestors_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Address") | ancestors | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Address is referenced by Owner, which is referenced by Pet
	assert.Contains(t, names, "Owner", "Address ancestors should include Owner")
}

func TestExecute_Properties_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | properties | pick name`, g)
	require.NoError(t, err)
	// Pet has 4 properties: id, name, tag, owner
	assert.NotEmpty(t, result.Rows, "Pet should have properties")
}

func TestExecute_UnionMembers_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Shape") | union-members | pick name`, g)
	require.NoError(t, err)
	// Shape has oneOf with Circle and Square
	names := collectNames(result, g)
	assert.Contains(t, names, "Circle", "Shape union members should include Circle")
	assert.Contains(t, names, "Square", "Shape union members should include Square")
}

func TestExecute_Operations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | pick name, method, path", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operations")
}

func TestExecute_OperationSchemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(operation_id == "listPets") | schemas | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "listPets operation should reference Pet schema")
}

func TestExecute_GroupBy_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | group_by(type)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "should have groups")
}

func TestExecute_Unique_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | unique", g)
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

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | ops | pick name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operations using Pet schema")
}

func TestFormatTable_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | first(3) | pick name, type", g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "name", "table should include name column")
	assert.Contains(t, table, "type", "table should include type column")
	assert.NotEmpty(t, table, "table should not be empty")
}

func TestFormatJSON_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | first(3) | pick name, type", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.True(t, strings.HasPrefix(json, "["), "JSON output should start with [")
	assert.True(t, strings.HasSuffix(json, "]"), "JSON output should end with ]")
}

func TestFormatTable_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | length", g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.NotEmpty(t, table, "count table should not be empty")
}

func TestFormatTable_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "NonExistent")`, g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.Equal(t, "(empty)", table, "empty result should format as (empty)")
}

func TestExecute_MatchesExpression_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name matches ".*Error.*") | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Error", "regex match should return Error schema")
}

func TestExecute_SortAsc_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | sort_by(name) | pick name", g)
	require.NoError(t, err)

	names := collectNames(result, g)
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "names should be in ascending order")
	}
}

func TestExecute_Explain_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | select(depth > 5) | sort_by(depth; desc) | first(10) | explain", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Source: schemas", "explain should show source")
	assert.Contains(t, result.Explain, "Filter: select(depth > 5)", "explain should show filter stage")
	assert.Contains(t, result.Explain, "Sort: sort_by(depth; desc)", "explain should show sort stage")
	assert.Contains(t, result.Explain, "Limit: first(10)", "explain should show limit stage")
}

func TestExecute_Fields_Schemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | fields", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "name", "fields output should list name")
	assert.Contains(t, result.Explain, "depth", "fields output should list depth")
	assert.Contains(t, result.Explain, "property_count", "fields output should list property_count")
	assert.Contains(t, result.Explain, "is_component", "fields output should list is_component")
}

func TestExecute_Fields_Operations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | fields", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "method", "fields output should list method")
	assert.Contains(t, result.Explain, "operation_id", "fields output should list operation_id")
	assert.Contains(t, result.Explain, "schema_count", "fields output should list schema_count")
	assert.Contains(t, result.Explain, "tag", "fields output should list tag")
	assert.Contains(t, result.Explain, "deprecated", "fields output should list deprecated")
}

func TestExecute_Sample_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | sample 3", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3, "sample should return exactly 3 rows")

	// Running sample again should produce the same result (deterministic)
	result2, err := oq.Execute("schemas | select(is_component) | sample 3", g)
	require.NoError(t, err)
	assert.Len(t, result2.Rows, len(result.Rows), "sample should be deterministic")
}

func TestExecute_Path_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | path Pet Address | pick name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "path from Pet to Address should have results")

	names := collectNames(result, g)
	// Path should include Pet, something in between, and Address
	assert.Equal(t, "Pet", names[0], "path should start at Pet")
	assert.Equal(t, "Address", names[len(names)-1], "path should end at Address")
}

func TestExecute_Path_NotFound_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Unused has no outgoing edges to reach Pet
	result, err := oq.Execute(`schemas | path Unused Pet | pick name`, g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows, "no path should exist from Unused to Pet")
}

func TestExecute_Top_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | top 3 property_count | pick name, property_count", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3, "top should return exactly 3 rows")

	// Verify descending order
	for i := 1; i < len(result.Rows); i++ {
		prev := oq.FieldValuePublic(result.Rows[i-1], "property_count", g)
		curr := oq.FieldValuePublic(result.Rows[i], "property_count", g)
		assert.GreaterOrEqual(t, prev.Int, curr.Int, "top should be in descending order")
	}
}

func TestExecute_Bottom_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | bottom 3 property_count | pick name, property_count", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3, "bottom should return exactly 3 rows")

	// Verify ascending order
	for i := 1; i < len(result.Rows); i++ {
		prev := oq.FieldValuePublic(result.Rows[i-1], "property_count", g)
		curr := oq.FieldValuePublic(result.Rows[i], "property_count", g)
		assert.LessOrEqual(t, prev.Int, curr.Int, "bottom should be in ascending order")
	}
}

func TestExecute_Format_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | first(3) | format json", g)
	require.NoError(t, err)
	assert.Equal(t, "json", result.FormatHint, "format hint should be json")
}

func TestFormatMarkdown_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | first(3) | pick name, type", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| name", "markdown should include name column header")
	assert.Contains(t, md, "| --- |", "markdown should include separator row")
}

func TestExecute_OperationTag_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | pick name, tag, parameter_count", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operation rows")
}

func TestParse_NewStages_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"explain", "schemas | explain"},
		{"fields", "schemas | fields"},
		{"first bare", "schemas | first(5)"},
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
			assert.NotEmpty(t, stages, "should parse into non-empty stages")
		})
	}
}

func TestExecute_RefsOut_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | refs-out | pick name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "Pet should have outgoing refs")
}

func TestExecute_RefsIn_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Owner") | refs-in | pick name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "Owner should have incoming refs")
}

func TestExecute_Items_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// listPets response includes an array with items
	result, err := oq.Execute(`schemas | select(type == "array") | items | pick name`, g)
	require.NoError(t, err)
	// May or may not have results depending on spec, but should not error
	assert.NotNil(t, result, "result should not be nil")
}

func TestExecute_Connected_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Start from Pet, connected should return schemas and operations in the same component
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | connected`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "connected should return rows")

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
	result, err := oq.Execute(`operations | first(1) | connected`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "connected from operation should return rows")

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

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | refs-out | pick name, via, key, from`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "refs-out from Pet should have results")

	// Every row should have edge annotations
	for _, row := range result.Rows {
		kind := oq.FieldValuePublic(row, "via", g)
		assert.NotEmpty(t, kind.Str, "via should be set")
		from := oq.FieldValuePublic(row, "from", g)
		assert.Equal(t, "Pet", from.Str, "from should be Pet")
	}
}

func TestExecute_BlastRadius_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | blast-radius`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "blast-radius should return rows")

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

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | neighbors 1`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "neighbors should return rows")

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

	result, err := oq.Execute(`schemas | select(is_component) | orphans | pick name`, g)
	require.NoError(t, err)
	// Result may be empty if all schemas are referenced, that's fine
	assert.NotNil(t, result, "result should not be nil")
}

func TestExecute_Leaves_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | leaves | pick name, out_degree`, g)
	require.NoError(t, err)
	// All returned rows should have out_degree == 0
	for _, row := range result.Rows {
		od := oq.FieldValuePublic(row, "out_degree", g)
		assert.Equal(t, 0, od.Int, "leaf nodes should have out_degree 0")
	}
}

func TestExecute_Cycles_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | cycles`, g)
	require.NoError(t, err)
	// Returns groups — may be empty if no cycles in petstore
	assert.NotNil(t, result, "result should not be nil")
}

func TestExecute_Clusters_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | clusters`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "should have clusters")

	// Total names across all clusters should equal component count
	total := 0
	for _, grp := range result.Groups {
		total += grp.Count
	}
	// Count component schemas
	compCount, err := oq.Execute(`schemas | select(is_component) | length`, g)
	require.NoError(t, err)
	assert.Equal(t, compCount.Count, total, "cluster totals should equal component count")
}

func TestExecute_TagBoundary_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | tag-boundary | pick name, tag_count`, g)
	require.NoError(t, err)
	// All returned rows should have tag_count > 1
	for _, row := range result.Rows {
		tc := oq.FieldValuePublic(row, "tag_count", g)
		assert.Greater(t, tc.Int, 1, "tag-boundary schemas should have tag_count > 1")
	}
}

func TestExecute_SharedRefs_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | shared-refs | pick name`, g)
	require.NoError(t, err)
	// Schemas shared by ALL operations
	assert.NotNil(t, result, "result should not be nil")
}

func TestExecute_OpCount_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | sort_by(op_count; desc) | first(3) | pick name, op_count`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have schemas sorted by op_count")
}

func TestFormatTable_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | group_by(type)", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "should have groups")

	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "count=", "group table should show count")
}

func TestFormatJSON_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | group_by(type)", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "\"key\"", "group JSON should include key field")
	assert.Contains(t, json, "\"count\"", "group JSON should include count field")
}

func TestFormatMarkdown_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | group_by(type)", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| Key |", "group markdown should include Key column")
}

func TestExecute_InlineSource_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_inline) | length", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount, "should be a count result")
}

func TestExecute_SchemaFields_Coverage(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Select all schema fields to cover fieldValue branches
	result, err := oq.Execute("schemas | select(is_component) | first(1) | pick name, type, depth, in_degree, out_degree, union_width, property_count, is_component, is_inline, is_circular, has_ref, hash, path", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have schema rows")

	table := oq.FormatTable(result, g)
	assert.NotEmpty(t, table, "table output should not be empty")

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "\"name\"", "JSON should include name field")
}

func TestExecute_OperationFields_Coverage(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Select all operation fields to cover fieldValue branches
	result, err := oq.Execute("operations | first(1) | pick name, method, path, operation_id, schema_count, component_count, tag, parameter_count, deprecated, description, summary", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operation rows")
}

func TestFormatJSON_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "NonExistent")`, g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.Equal(t, "[]", json, "empty result JSON should be []")
}

func TestFormatMarkdown_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "NonExistent")`, g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Equal(t, "(empty)", md, "empty result markdown should be (empty)")
}

func TestFormatJSON_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | length", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.NotEmpty(t, json, "count JSON should not be empty")
}

func TestFormatToon_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | first(3) | pick name, type", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "results[3]{name,type}:", "toon should show result count and fields")
	assert.Contains(t, toon, "object", "toon should include object type value")
}

func TestFormatToon_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | length", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "count:", "toon should show count label")
}

func TestFormatToon_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | group_by(type)", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "groups[", "toon should show groups header")
	assert.Contains(t, toon, "{key,count,names}:", "toon should show group fields")
}

func TestFormatToon_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "NonExistent")`, g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Equal(t, "results[0]:\n", toon, "empty toon should show results[0]")
}

func TestFormatToon_Escaping_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Paths contain special chars like / that don't need escaping,
	// but hash values and paths are good coverage
	result, err := oq.Execute("schemas | select(is_component) | first(1) | pick name, hash, path", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "results[1]{name,hash,path}:", "toon should show result count and selected fields")
}

func TestFormatMarkdown_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | length", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.NotEmpty(t, md, "count markdown should not be empty")
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
			"schemas | select(is_component) | unique | length | explain",
			[]string{"Unique:", "Count:"},
		},
		{
			"explain with group_by",
			"schemas | select(is_component) | group_by(type) | explain",
			[]string{"Group: group_by("},
		},
		{
			"explain with traversals",
			"schemas | select(is_component) | select(name == \"Pet\") | refs-out | explain",
			[]string{"Traverse: outgoing references"},
		},
		{
			"explain with refs-in",
			"schemas | select(is_component) | select(name == \"Owner\") | refs-in | explain",
			[]string{"Traverse: incoming references"},
		},
		{
			"explain with reachable",
			"schemas | select(is_component) | select(name == \"Pet\") | reachable | explain",
			[]string{"Traverse: all reachable"},
		},
		{
			"explain with ancestors",
			"schemas | select(is_component) | select(name == \"Address\") | ancestors | explain",
			[]string{"Traverse: all ancestor"},
		},
		{
			"explain with properties",
			"schemas | select(is_component) | select(name == \"Pet\") | properties | explain",
			[]string{"Traverse: property children"},
		},
		{
			"explain with union-members",
			"schemas | select(is_component) | select(name == \"Shape\") | union-members | explain",
			[]string{"Traverse: union members"},
		},
		{
			"explain with items",
			"schemas | select(type == \"array\") | items | explain",
			[]string{"Traverse: array items"},
		},
		{
			"explain with ops",
			"schemas | select(is_component) | select(name == \"Pet\") | ops | explain",
			[]string{"Navigate: schemas to operations"},
		},
		{
			"explain with schemas from ops",
			"operations | schemas | explain",
			[]string{"Navigate: operations to schemas"},
		},
		{
			"explain with sample",
			"schemas | select(is_component) | sample 3 | explain",
			[]string{"Sample: random 3"},
		},
		{
			"explain with path",
			"schemas | path Pet Address | explain",
			[]string{"Path: shortest path from Pet to Address"},
		},
		{
			"explain with top",
			"schemas | select(is_component) | top 3 depth | explain",
			[]string{"Top: 3 by depth"},
		},
		{
			"explain with bottom",
			"schemas | select(is_component) | bottom 3 depth | explain",
			[]string{"Bottom: 3 by depth"},
		},
		{
			"explain with format",
			"schemas | select(is_component) | format json | explain",
			[]string{"Format: json"},
		},
		{
			"explain with connected",
			"schemas | select(is_component) | select(name == \"Pet\") | connected | explain",
			[]string{"Traverse: full connected"},
		},
		{
			"explain with blast-radius",
			"schemas | select(is_component) | select(name == \"Pet\") | blast-radius | explain",
			[]string{"Traverse: blast radius"},
		},
		{
			"explain with neighbors",
			"schemas | select(is_component) | select(name == \"Pet\") | neighbors 2 | explain",
			[]string{"Traverse: bidirectional neighbors within 2"},
		},
		{
			"explain with orphans",
			"schemas | select(is_component) | orphans | explain",
			[]string{"Filter: schemas with no incoming"},
		},
		{
			"explain with leaves",
			"schemas | select(is_component) | leaves | explain",
			[]string{"Filter: schemas with no outgoing"},
		},
		{
			"explain with cycles",
			"schemas | cycles | explain",
			[]string{"Analyze: strongly connected"},
		},
		{
			"explain with clusters",
			"schemas | select(is_component) | clusters | explain",
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
				assert.Contains(t, result.Explain, exp, "explain should contain: "+exp)
			}
		})
	}
}

func TestExecute_FieldValue_EdgeCases(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test operation fields that require nil checks
	result, err := oq.Execute("operations | first(1) | pick name, tag, parameter_count, deprecated, description, summary", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operation rows")

	// Test edge fields on non-traversal rows (should be empty strings)
	result, err = oq.Execute("schemas | select(is_component) | first(1) | pick name, via, key, from", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have schema rows")
	viaVal := oq.FieldValuePublic(result.Rows[0], "via", g)
	assert.Empty(t, viaVal.Str, "via should be empty for non-traversal rows")

	// Test tag_count field
	result, err = oq.Execute("schemas | select(is_component) | first(1) | pick name, tag_count", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have rows for tag_count test")

	// Test op_count field
	result, err = oq.Execute("schemas | select(is_component) | first(1) | pick name, op_count", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have rows for op_count test")

	// Test unknown field returns null (KindNull == 0)
	v := oq.FieldValuePublic(result.Rows[0], "nonexistent_field", g)
	assert.Equal(t, expr.KindNull, v.Kind, "unknown field should return KindNull")
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
	result, err := oq.Execute("operations | shared-refs | pick name", g)
	require.NoError(t, err)
	assert.NotNil(t, result, "result should not be nil")
}

func TestFormatToon_SpecialChars(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test TOON format with bool and int fields to cover toonValue branches
	result, err := oq.Execute("schemas | select(is_component) | first(1) | pick name, depth, is_component", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon, "toon output should not be empty")
	assert.Contains(t, toon, "results[1]", "toon should show one result")
}

func TestFormatJSON_Operations(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | first(2) | pick name, method, path", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.True(t, strings.HasPrefix(json, "["), "JSON output should start with [")
	assert.Contains(t, json, "\"name\"", "JSON should include name field")
	assert.Contains(t, json, "\"method\"", "JSON should include method field")
}

func TestFormatMarkdown_Operations(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | first(2) | pick name, method", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| name", "markdown should include name column")
	assert.Contains(t, md, "| method", "markdown should include method column")
}

func TestParse_Error_MoreCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"empty query", ""},
		{"unknown stage", "schemas | bogus_stage"},
		{"first non-integer", "schemas | first abc"},
		{"sample non-integer", "schemas | sample xyz"},
		{"neighbors non-integer", "schemas | neighbors abc"},
		{"top missing field", "schemas | top 5"},
		{"bottom missing field", "schemas | bottom 5"},
		{"path missing args", "schemas | path"},
		{"path one arg", "schemas | path Pet"},
		{"select empty expr", "schemas | select()"},
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
		{"sort_by", "schemas | sort_by(name)"},
		{"pick single field", "schemas | pick name"},
		{"pick many fields", "schemas | pick name, type, depth, in_degree"},
		{"select with string", `schemas | select(name == "Pet")`},
		{"select with bool", "schemas | select(is_component)"},
		{"select with not", "schemas | select(not is_inline)"},
		{"select with has", "schemas | select(has(hash))"},
		{"select with matches", `schemas | select(name matches ".*Pet.*")`},
		{"path quoted", `schemas | path "Pet" "Address"`},
		{"shared-refs stage", "operations | first(2) | shared-refs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stages, err := oq.Parse(tt.query)
			require.NoError(t, err)
			assert.NotEmpty(t, stages, "should parse into non-empty stages")
		})
	}
}

func TestExecute_WhereAndOr_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test compound where expressions
	result, err := oq.Execute(`schemas | select(is_component) | select(depth > 0 and is_component)`, g)
	require.NoError(t, err)
	assert.NotNil(t, result, "result should not be nil")

	result, err = oq.Execute(`schemas | select(is_component) | select(depth > 100 or is_component)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "or should match is_component=true schemas")
}

func TestExecute_SortStringField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Sort by string field
	result, err := oq.Execute("schemas | select(is_component) | sort_by(type) | pick name, type", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have schemas sorted by type")
}

func TestExecute_GroupBy_Type_Details(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | group_by(type)", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "should have groups")

	// Each group should have Count and Names
	for _, grp := range result.Groups {
		assert.Positive(t, grp.Count, "group count should be positive")
		assert.Len(t, grp.Names, grp.Count, "group names length should match count")
	}
}

func TestFormatMarkdown_Groups_Details(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | group_by(type)", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| Key |", "group markdown should include Key column")
	assert.Contains(t, md, "| Count |", "group markdown should include Count column")
}

func TestFormatJSON_Explain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | explain", g)
	require.NoError(t, err)

	// All formats should handle explain
	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "Source: schemas", "table should render explain output")

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "Source: schemas", "JSON should render explain output")

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "Source: schemas", "markdown should render explain output")

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "Source: schemas", "toon should render explain output")
}

func TestExecute_Leaves_AllZeroOutDegree(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | leaves | pick name, out_degree", g)
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
	result, err := oq.Execute("operations | first(1) | schemas | pick name", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "operation schemas should have results")

	// Schema to operations roundtrip
	result, err = oq.Execute("schemas | select(is_component) | select(name == \"Pet\") | ops | pick name", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "Pet should be used by operations")
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
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "NodeA") | refs-out | pick name, via, key`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "NodeA should have outgoing refs")

	// Collect edge kinds (via)
	edgeKinds := make(map[string]bool)
	for _, row := range result.Rows {
		k := oq.FieldValuePublic(row, "via", g)
		edgeKinds[k.Str] = true
	}
	// NodeA has properties, allOf, anyOf, items etc.
	assert.True(t, edgeKinds["property"], "should have property edges")
}

func TestExecute_CyclicSpec_IsCircular(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | select(is_circular) | pick name", g)
	require.NoError(t, err)
	names := collectNames(result, g)
	assert.Contains(t, names, "NodeA", "NodeA is in the A↔B cycle")
	assert.Contains(t, names, "NodeB", "NodeB is in the A↔B cycle")

	// NodeC is NOT in the cycle — it's only referenced by NodeA via allOf
	assert.NotContains(t, names, "NodeC", "NodeC should not be marked circular")
}

func TestExecute_CyclicSpec_DeprecatedOp(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// The listNodes operation is deprecated with tags, summary, and description
	result, err := oq.Execute("operations | pick name, deprecated, summary, description, tag, parameter_count", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operation rows")

	dep := oq.FieldValuePublic(result.Rows[0], "deprecated", g)
	assert.True(t, dep.Bool, "listNodes should be deprecated")

	summary := oq.FieldValuePublic(result.Rows[0], "summary", g)
	assert.Equal(t, "List all nodes", summary.Str, "summary should match spec")

	desc := oq.FieldValuePublic(result.Rows[0], "description", g)
	assert.NotEmpty(t, desc.Str, "description should not be empty")

	tag := oq.FieldValuePublic(result.Rows[0], "tag", g)
	assert.Equal(t, "nodes", tag.Str, "tag should be nodes")
}

func TestExecute_ToonFormat_WithBoolAndInt(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Select fields that cover all toonValue branches (string, int, bool)
	result, err := oq.Execute("schemas | select(is_component) | first(1) | pick name, depth, is_circular", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon, "toon output should not be empty")
}

func TestExecute_ToonEscape_SpecialChars(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// path fields contain "/" which doesn't need quoting, but let's cover the formatter
	result, err := oq.Execute("schemas | first(3) | pick path", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon, "toon output should not be empty")
}

func TestFormatToon_Explain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(depth > 0) | explain", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "Source: schemas", "toon should render explain output")
}

func TestFormatMarkdown_Explain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | explain", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "Source: schemas", "markdown should render explain output")
}

// --- New jq-style syntax tests ---

func TestParse_NewSyntax_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"select filter", `schemas | select(depth > 3)`},
		{"pick fields", "schemas | pick name, depth"},
		{"sort_by asc", "schemas | sort_by(depth)"},
		{"sort_by desc", "schemas | sort_by(depth; desc)"},
		{"first", "schemas | first(5)"},
		{"last", "schemas | last(5)"},
		{"length", "schemas | length"},
		{"group_by", "schemas | group_by(type)"},
		{"sample call", "schemas | sample(3)"},
		{"neighbors call", "schemas | neighbors(2)"},
		{"path call", "schemas | path(Pet; Address)"},
		{"top call", "schemas | top(3; depth)"},
		{"bottom call", "schemas | bottom(3; depth)"},
		{"format call", "schemas | format(json)"},
		{"let binding", `schemas | select(name == "Pet") | let $pet = name`},
		{"full new pipeline", `schemas | select(is_component) | select(depth > 5) | sort_by(depth; desc) | first(10) | pick name, depth`},
		{"def inline", `def hot: select(in_degree > 0); schemas | select(is_component) | hot`},
		{"def with params", `def impact($name): select(name == $name); schemas | select(is_component) | impact("Pet")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stages, err := oq.Parse(tt.query)
			require.NoError(t, err, "query: %s", tt.query)
			assert.NotEmpty(t, stages, "should parse into non-empty stages")
		})
	}
}

func TestExecute_SelectFilter_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(type == "object") | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "select filter should match Pet")
	assert.Contains(t, names, "Owner", "select filter should match Owner")
}

func TestExecute_SortBy_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | sort_by(property_count; desc) | first(3) | pick name, property_count", g)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Rows), 3, "should return at most 3 rows")
}

func TestExecute_First_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | first(3)", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3, "first should return exactly 3 rows")
}

func TestExecute_Last_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | last(2)", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 2, "last should return exactly 2 rows")
}

func TestExecute_Length_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | length", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount, "length should be a count result")
	assert.Positive(t, result.Count, "count should be positive")
}

func TestExecute_GroupBy_NewSyntax_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | group_by(type)", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "should have groups")
}

func TestExecute_LetBinding_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// let $pet = name, then use $pet in subsequent filter
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | let $pet = name | reachable | select(name != $pet) | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.NotContains(t, names, "Pet", "should not include the $pet variable value")
	assert.Contains(t, names, "Owner", "should include reachable schemas")
}

func TestExecute_DefExpansion_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`def hot: select(in_degree > 0); schemas | select(is_component) | hot | pick name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "def expansion should produce results")

	// All results should have in_degree > 0
	for _, row := range result.Rows {
		v := oq.FieldValuePublic(row, "in_degree", g)
		assert.Positive(t, v.Int, "hot filter should require in_degree > 0")
	}
}

func TestExecute_DefWithParams_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`def impact($name): select(name == $name) | blast-radius; schemas | select(is_component) | impact("Pet")`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "parameterized def should produce results")
}

func TestExecute_AlternativeOperator_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// name // "none" — name is always set, so should not be "none"
	result, err := oq.Execute(`schemas | select(is_component) | select(name // "none" != "none") | pick name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "alternative operator should work")
}

func TestExecute_IfThenElse_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(if is_component then depth >= 0 else true end) | pick name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "if-then-else should work in select")
}

func TestExecute_ExplainNewSyntax_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(depth > 5) | sort_by(depth; desc) | first(10) | pick name, depth | explain`, g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Filter: select(depth > 5)", "explain should show select filter")
	assert.Contains(t, result.Explain, "Sort: sort_by(depth; desc)", "explain should show sort_by")
	assert.Contains(t, result.Explain, "Limit: first(10)", "explain should show first")
	assert.Contains(t, result.Explain, "Project: pick name, depth", "explain should show pick")
}

func TestExecute_ExplainLast_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | select(is_component) | last(3) | explain", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Limit: last(3)", "explain should show last")
}

func TestExecute_ExplainLet_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | let $pet = name | explain`, g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Bind: let $pet = name", "explain should show let binding")
}

func TestParse_NewSyntax_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"select call empty", "schemas | select()"},
		{"sort_by no parens", "schemas | sort_by depth"},
		{"group_by no parens", "schemas | group_by type"},
		{"let no dollar", "schemas | let x = name"},
		{"let no equals", "schemas | let $x name"},
		{"let empty expr", "schemas | let $x ="},
		{"def missing colon", "def hot select(depth > 0); schemas | hot"},
		{"def missing semicolon", "def hot: select(depth > 0) schemas | hot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := oq.Parse(tt.query)
			assert.Error(t, err, "query should fail: %s", tt.query)
		})
	}
}

// --- Navigation stage tests ---

func TestExecute_Parameters(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | parameters", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should find parameters")

	// Check that parameters have the right fields
	for _, row := range result.Rows {
		assert.Equal(t, oq.ParameterResult, row.Kind)
		name := oq.FieldValuePublic(row, "name", g)
		assert.NotEmpty(t, name.Str, "parameter should have a name")
		in := oq.FieldValuePublic(row, "in", g)
		assert.Contains(t, []string{"query", "path", "header", "cookie"}, in.Str)
	}

	// Operation back-navigation
	ops, err := oq.Execute("operations | parameters | operation | unique", g)
	require.NoError(t, err)
	assert.NotEmpty(t, ops.Rows, "should find source operations")
	for _, row := range ops.Rows {
		assert.Equal(t, oq.OperationResult, row.Kind)
	}
}

func TestExecute_Responses(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | first(1) | responses", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should find responses")

	for _, row := range result.Rows {
		assert.Equal(t, oq.ResponseResult, row.Kind)
		sc := oq.FieldValuePublic(row, "status_code", g)
		assert.NotEmpty(t, sc.Str, "response should have status_code")
	}
}

func TestExecute_ContentTypes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | first(1) | responses | content-types", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should find content types")

	for _, row := range result.Rows {
		assert.Equal(t, oq.ContentTypeResult, row.Kind)
		mt := oq.FieldValuePublic(row, "media_type", g)
		assert.NotEmpty(t, mt.Str, "content type should have media_type")
	}
}

func TestExecute_RequestBody(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// createPet has a request body
	result, err := oq.Execute(`operations | select(name == "createPet") | request-body`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 1, "createPet should have one request body")
	assert.Equal(t, oq.RequestBodyResult, result.Rows[0].Kind)

	// request-body | content-types
	ct, err := oq.Execute(`operations | select(name == "createPet") | request-body | content-types`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, ct.Rows, "request body should have content types")
}

func TestExecute_SchemaResolvesRef(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// The schema stage should resolve $ref wrappers to the component they reference
	result, err := oq.Execute("operations | first(1) | responses | content-types | schema", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should resolve schemas")

	for _, row := range result.Rows {
		assert.Equal(t, oq.SchemaResult, row.Kind)
		// After resolving $ref, the schema should not be a bare $ref wrapper
		hasRef := oq.FieldValuePublic(row, "has_ref", g)
		isComp := oq.FieldValuePublic(row, "is_component", g)
		// If the original was a $ref, the resolved schema should be the component
		if hasRef.Bool {
			assert.True(t, isComp.Bool, "resolved $ref schema should be a component")
		}
	}
}

func TestExecute_SchemaFromParameterBridgesToGraph(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Get schema from a parameter, then use graph traversal on it
	result, err := oq.Execute("operations | parameters | first(1) | schema", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "parameter should have a schema")
	assert.Equal(t, oq.SchemaResult, result.Rows[0].Kind)
}

func TestExecute_UniqueAfterPick(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Without pick, unique deduplicates by row identity
	all, err := oq.Execute("operations | responses | content-types", g)
	require.NoError(t, err)

	// With pick media_type + unique, should deduplicate by the projected value
	deduped, err := oq.Execute("operations | responses | content-types | pick media_type | unique", g)
	require.NoError(t, err)
	assert.Less(t, len(deduped.Rows), len(all.Rows), "unique after pick should reduce rows")

	// All remaining rows should have distinct media_type values
	seen := make(map[string]bool)
	for _, row := range deduped.Rows {
		mt := oq.FieldValuePublic(row, "media_type", g)
		assert.False(t, seen[mt.Str], "media_type %q should not be duplicated", mt.Str)
		seen[mt.Str] = true
	}
}

func TestExecute_UniqueWithoutPick_UsesRowKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Without pick, unique should use row identity (original behavior)
	result, err := oq.Execute("schemas | select(is_component) | unique | length", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount)

	all, err := oq.Execute("schemas | select(is_component) | length", g)
	require.NoError(t, err)
	assert.Equal(t, all.Count, result.Count, "unique on already-unique rows should keep all")
}

func TestExecute_NavStageOnWrongType_EmptyNotError(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// parameters on schemas should be empty, not error
	result, err := oq.Execute("schemas | first(1) | parameters", g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)

	// headers on operations (need responses first) should be empty
	result, err = oq.Execute("operations | first(1) | headers", g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)

	// content-types on schemas should be empty
	result, err = oq.Execute("schemas | first(1) | content-types", g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)
}

func TestExecute_ComponentsSources(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// components.schemas = schemas | select(is_component)
	compSchemas, err := oq.Execute("components.schemas | length", g)
	require.NoError(t, err)
	filteredSchemas, err := oq.Execute("schemas | select(is_component) | length", g)
	require.NoError(t, err)
	assert.Equal(t, filteredSchemas.Count, compSchemas.Count)

	// components.parameters should work (may be 0 for petstore)
	_, err = oq.Execute("components.parameters | length", g)
	require.NoError(t, err)

	// components.responses should work
	_, err = oq.Execute("components.responses | length", g)
	require.NoError(t, err)
}

func TestExecute_NavigationFullChain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Full navigation chain: operation → responses → content-types → schema → graph traversal
	result, err := oq.Execute("operations | first(1) | responses | content-types | schema | refs-out | first(3)", g)
	require.NoError(t, err)
	// May be empty depending on whether the schema has refs, but should not error
	for _, row := range result.Rows {
		assert.Equal(t, oq.SchemaResult, row.Kind)
	}
}

func TestExecute_OperationBackNav(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Get distinct operations from responses
	result, err := oq.Execute("operations | responses | operation | unique | length", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount)

	opCount, err := oq.Execute("operations | length", g)
	require.NoError(t, err)
	// Every operation should have responses, so back-nav should recover all ops
	assert.Equal(t, opCount.Count, result.Count, "back-nav should recover all operations")
}

func TestExecute_ResponseContextPropagation(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Content-type rows should inherit status_code from their parent response
	result, err := oq.Execute("operations | first(1) | responses | content-types | pick status_code, media_type, operation", g)
	require.NoError(t, err)
	for _, row := range result.Rows {
		sc := oq.FieldValuePublic(row, "status_code", g)
		assert.NotEmpty(t, sc.Str, "content-type should inherit status_code")
		op := oq.FieldValuePublic(row, "operation", g)
		assert.NotEmpty(t, op.Str, "content-type should inherit operation")
	}
}

func TestParse_NavigationStages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"parameters", "operations | parameters"},
		{"responses", "operations | responses"},
		{"request-body", "operations | request-body"},
		{"content-types", "operations | responses | content-types"},
		{"headers", "operations | responses | headers"},
		{"schema", "operations | parameters | schema"},
		{"operation", "operations | parameters | operation"},
		{"security", "operations | security"},
		{"components.schemas", "components.schemas"},
		{"components.parameters", "components.parameters"},
		{"components.responses", "components.responses"},
		{"components.request-bodies", "components.request-bodies"},
		{"components.headers", "components.headers"},
		{"components.security-schemes", "components.security-schemes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := oq.Parse(tt.query)
			require.NoError(t, err)
		})
	}
}

func TestExecute_SecurityGlobalInheritance(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// listPets has no per-operation security, should inherit global bearerAuth
	result, err := oq.Execute(`operations | select(name == "listPets") | security`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1, "listPets should inherit global security")
	assert.Equal(t, oq.SecurityRequirementResult, result.Rows[0].Kind)

	schemeName := oq.FieldValuePublic(result.Rows[0], "scheme_name", g)
	assert.Equal(t, "bearerAuth", schemeName.Str)

	schemeType := oq.FieldValuePublic(result.Rows[0], "scheme_type", g)
	assert.Equal(t, "http", schemeType.Str)
}

func TestExecute_SecurityPerOperation(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// createPet has per-operation security with scopes
	result, err := oq.Execute(`operations | select(name == "createPet") | security`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	scopes := oq.FieldValuePublic(result.Rows[0], "scopes", g)
	assert.Equal(t, expr.KindArray, scopes.Kind)
	assert.Equal(t, []string{"pets:write"}, scopes.Arr)
}

func TestExecute_SecurityOptOut(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// streamEvents has security: [] (explicit opt-out)
	result, err := oq.Execute(`operations | select(name == "streamEvents") | security`, g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows, "streamEvents should have no security (explicit opt-out)")
}

func TestExecute_ComponentsSecuritySchemes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("components.security-schemes", g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "bearerAuth", name.Str)

	schemeType := oq.FieldValuePublic(result.Rows[0], "type", g)
	assert.Equal(t, "http", schemeType.Str)

	scheme := oq.FieldValuePublic(result.Rows[0], "scheme", g)
	assert.Equal(t, "bearer", scheme.Str)
}

func TestExecute_ComponentsParameters(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("components.parameters", g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "LimitParam", name.Str)
}

func TestExecute_ComponentsResponses(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("components.responses", g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "NotFound", name.Str)

	// status_code should be empty for component responses (not leaked from key)
	sc := oq.FieldValuePublic(result.Rows[0], "status_code", g)
	assert.Empty(t, sc.Str)
}

func TestExecute_GroupByWithNameField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// group_by(status_code; operation) collects operation names instead of status codes
	result, err := oq.Execute("operations | responses | group_by(status_code; operation)", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups)

	for _, grp := range result.Groups {
		for _, n := range grp.Names {
			assert.NotEqual(t, grp.Key, n, "group names should be operation names, not status codes")
		}
	}
}

func TestExecute_DeprecatedParameters(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | parameters | select(deprecated) | pick name, operation", g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "format", name.Str)
}

func TestExecute_SSEContentType(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | responses | content-types | select(media_type == "text/event-stream") | operation | unique`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "streamEvents", name.Str)
}

func TestExecute_MultipleContentTypes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// createPet request body has both application/json and multipart/form-data
	result, err := oq.Execute(`operations | select(name == "createPet") | request-body | content-types | pick media_type`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 2)

	types := make([]string, len(result.Rows))
	for i, row := range result.Rows {
		types[i] = oq.FieldValuePublic(row, "media_type", g).Str
	}
	assert.Contains(t, types, "application/json")
	assert.Contains(t, types, "multipart/form-data")
}

func TestExecute_ResponseHeaders(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// createPet 201 response has X-Request-Id header
	result, err := oq.Execute(`operations | select(name == "createPet") | responses | select(status_code == "201") | headers`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "X-Request-Id", name.Str)

	req := oq.FieldValuePublic(result.Rows[0], "required", g)
	assert.True(t, req.Bool)
}

func TestExecute_UniqueContentTypes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Unique after pick should deduplicate by projected values
	result, err := oq.Execute("operations | responses | content-types | pick media_type | unique", g)
	require.NoError(t, err)

	seen := make(map[string]bool)
	for _, row := range result.Rows {
		mt := oq.FieldValuePublic(row, "media_type", g).Str
		assert.False(t, seen[mt], "duplicate media_type %q after unique", mt)
		seen[mt] = true
	}
	// Should have application/json and text/event-stream
	assert.True(t, seen["application/json"])
	assert.True(t, seen["text/event-stream"])
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
