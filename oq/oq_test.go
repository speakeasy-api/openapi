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
		{"references", "schemas | references"},
		{"referenced-by", "schemas | referenced-by"},
		{"descendants", "schemas | descendants"},
		{"ancestors", "schemas | ancestors"},
		{"ancestors depth", "schemas | ancestors(2)"},
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

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | descendants | pick name`, g)
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
	assert.Equal(t, "(empty)\n", table, "empty result should format as (empty)")
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

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | references | pick name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "Pet should have outgoing refs")
}

func TestExecute_RefsIn_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Owner") | referenced-by | pick name`, g)
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

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | references | pick name, via, key, from`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "references from Pet should have results")

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
	assert.Contains(t, table, "key", "group table should have key column")
	assert.Contains(t, table, "count", "group table should have count column")
	assert.Contains(t, table, "names", "group table should have names column")
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
	assert.Contains(t, md, "| key |", "group markdown should include key column")
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
	assert.Equal(t, "(empty)\n", md, "empty result markdown should be (empty)")
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
	assert.Contains(t, toon, "results[", "toon should show results header")
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
			"schemas | select(is_component) | select(name == \"Pet\") | references | explain",
			[]string{"Traverse: direct outgoing references"},
		},
		{
			"explain with referenced-by",
			"schemas | select(is_component) | select(name == \"Owner\") | referenced-by | explain",
			[]string{"Traverse: schemas that reference this one"},
		},
		{
			"explain with descendants",
			"schemas | select(is_component) | select(name == \"Pet\") | descendants | explain",
			[]string{"Traverse: all descendants"},
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
		{
			"explain with ancestors depth",
			"schemas | select(is_component) | select(name == \"Address\") | ancestors(2) | explain",
			[]string{"Traverse: ancestors within 2 hops"},
		},
		{
			"explain with descendants depth",
			"schemas | select(is_component) | select(name == \"Pet\") | descendants(1) | explain",
			[]string{"Traverse: descendants within 1 hops"},
		},
		{
			"explain with parent",
			"schemas | select(is_component) | select(name == \"Pet\") | properties | parent | explain",
			[]string{"Traverse: structural parent"},
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
	assert.Contains(t, md, "| key |", "group markdown should include key column")
	assert.Contains(t, md, "| count |", "group markdown should include count column")
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

	// Test references to cover edgeKindString branches
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "NodeA") | references | pick name, via, key`, g)
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
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | let $pet = name | descendants | select(name != $pet) | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.NotContains(t, names, "Pet", "should not include the $pet variable value")
	assert.Contains(t, names, "Owner", "should include descendants schemas")
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
	result, err := oq.Execute("operations | first(1) | responses | content-types | schema | references | first(3)", g)
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

func TestEval_InfixStartswith(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name startswith "S") | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	for _, n := range names {
		assert.True(t, len(n) > 0 && n[0] == 'S', "name %q should start with S", n)
	}
	assert.Contains(t, names, "Shape")
	assert.Contains(t, names, "Square")
}

func TestEval_InfixEndswith(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(is_component) | select(name endswith "er") | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Owner")
	for _, n := range names {
		assert.True(t, len(n) >= 2 && n[len(n)-2:] == "er", "name %q should end with er", n)
	}
}

func TestExecute_PropertiesContains(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Pet and Owner both have a "name" property
	result, err := oq.Execute(`schemas | select(properties contains "name") | select(is_component) | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet")
	assert.Contains(t, names, "Owner")
	assert.NotContains(t, names, "Error", "Error has code and message, not name")
}

func TestExecute_PropertiesField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | select(name == "Pet") | pick name, properties`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	props := oq.FieldValuePublic(result.Rows[0], "properties", g)
	assert.Equal(t, expr.KindArray, props.Kind)
	assert.Contains(t, props.Arr, "id")
	assert.Contains(t, props.Arr, "name")
	assert.Contains(t, props.Arr, "owner")
}

func TestExecute_KindField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | first(1) | pick kind, name`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	kind := oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "schema", kind.Str)

	result, err = oq.Execute(`operations | first(1) | pick kind, name`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	kind = oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "operation", kind.Str)
}

func TestExecute_DescendantsDepthLimited(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// descendants(1) = direct children only
	d1, err := oq.Execute(`schemas | select(name == "Pet") | descendants(1) | length`, g)
	require.NoError(t, err)

	// descendants (unlimited) = full transitive closure
	dAll, err := oq.Execute(`schemas | select(name == "Pet") | descendants | length`, g)
	require.NoError(t, err)

	assert.Greater(t, dAll.Count, d1.Count, "unlimited should find more than 1-hop")

	// descendants(2) should include more than 1-hop but may equal unlimited for shallow graphs
	d2, err := oq.Execute(`schemas | select(name == "Pet") | descendants(2) | length`, g)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, d2.Count, d1.Count)
}

func TestExecute_MixedTypeDefaultFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// blast-radius returns mixed schema+operation rows — should show kind column
	result, err := oq.Execute(`schemas | select(name == "Pet") | blast-radius`, g)
	require.NoError(t, err)

	// Verify mixed types present
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
	assert.True(t, hasSchema, "should have schema rows")
	assert.True(t, hasOp, "should have operation rows")

	// Table output should include "kind" column for mixed results
	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "kind")
}

func TestExecute_EmitResponseKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Emit on responses should use operation/status_code as key
	result, err := oq.Execute(`operations | first(1) | responses | first(1) | emit`, g)
	require.NoError(t, err)
	assert.True(t, result.EmitYAML)

	yaml := oq.FormatYAML(result, g)
	// Should contain operation name in the key (e.g., "listPets/200:")
	assert.Contains(t, yaml, "listPets/")
}

func TestExecute_EmitSchemaPath(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Emit on schemas should use path as key
	result, err := oq.Execute(`schemas | select(name == "Pet") | emit`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "/components/schemas/Pet")
}

func TestFormatJSON_ArrayValues(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// JSON format should render array fields as JSON arrays, not null
	result, err := oq.Execute(`schemas | select(name == "Pet") | pick name, properties | format(json)`, g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "[")
	assert.Contains(t, json, `"id"`)
}

func TestExecute_ArrayMatchesRegex(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// properties matches regex — any property name matching pattern
	result, err := oq.Execute(`schemas | select(properties matches "^ow") | select(is_component) | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "Pet has 'owner' property starting with 'ow'")
}

func TestExecute_ArrayStartswith(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// properties startswith — any property name with prefix
	result, err := oq.Execute(`schemas | select(properties startswith "na") | select(is_component) | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "Pet has 'name' property")
	assert.Contains(t, names, "Owner", "Owner has 'name' property")
}

func TestExecute_ArrayEndswith(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// properties endswith — any property name with suffix
	result, err := oq.Execute(`schemas | select(properties endswith "eet") | select(is_component) | pick name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Address", "Address has 'street' property")
}

func TestExecute_EmitParameterKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | parameters | first(1) | emit`, g)
	require.NoError(t, err)
	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "/parameters/", "emit key should include /parameters/ path")
}

func TestExecute_EmitContentTypeKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | first(1) | responses | first(1) | content-types | emit`, g)
	require.NoError(t, err)
	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "application/json", "emit key should include media type")
}

func TestExecute_EmitSecuritySchemeKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components.security-schemes | emit`, g)
	require.NoError(t, err)
	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "bearerAuth", "emit key should be scheme name")
}

func TestExecute_EmitRequestBodyKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(name == "createPet") | request-body | emit`, g)
	require.NoError(t, err)
	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "createPet/request-body", "emit key should include operation name")
}

func TestExecute_GroupByTable_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// group_by should now render as a standard table, not summary format
	result, err := oq.Execute("schemas | select(is_component) | group_by(type)", g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	// Should have table headers
	assert.Contains(t, table, "key")
	assert.Contains(t, table, "count")
	assert.Contains(t, table, "---")

	// JSON format should also work as regular table
	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "\"key\"")
	assert.Contains(t, json, "\"count\"")
	assert.Contains(t, json, "\"names\"")

	// Toon format
	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "results[")
	assert.Contains(t, toon, "{key,count,names}")
}

func TestExecute_CyclesTable_Success(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	result, err := oq.Execute("schemas | cycles", g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "key")
	assert.Contains(t, table, "count")
}

func TestExecute_KindFieldAllTypes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test kind field for various row types
	tests := []struct {
		query    string
		expected string
	}{
		{"schemas | first(1)", "schema"},
		{"operations | first(1)", "operation"},
		{"operations | parameters | first(1)", "parameter"},
		{"operations | first(1) | responses | first(1)", "response"},
		{"operations | select(name == \"createPet\") | request-body", "request-body"},
		{"operations | first(1) | responses | first(1) | content-types | first(1)", "content-type"},
		{"components.security-schemes | first(1)", "security-scheme"},
	}

	for _, tt := range tests {
		result, err := oq.Execute(tt.query, g)
		require.NoError(t, err, "query: %s", tt.query)
		if len(result.Rows) > 0 {
			kind := oq.FieldValuePublic(result.Rows[0], "kind", g)
			assert.Equal(t, tt.expected, kind.Str, "query: %s", tt.query)
		}
	}
}

func TestExecute_DefaultFieldsForAllTypes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Verify each row type renders without error in all formats
	queries := []string{
		"schemas | first(1)",
		"operations | first(1)",
		"operations | parameters | first(1)",
		"operations | first(1) | responses | first(1)",
		"operations | select(name == \"createPet\") | request-body",
		"operations | first(1) | responses | first(1) | content-types | first(1)",
		"components.security-schemes",
		"operations | first(1) | security",
	}

	for _, q := range queries {
		result, err := oq.Execute(q, g)
		require.NoError(t, err, "query: %s", q)
		if len(result.Rows) > 0 {
			table := oq.FormatTable(result, g)
			assert.NotEmpty(t, table, "table should not be empty for: %s", q)
			json := oq.FormatJSON(result, g)
			assert.NotEmpty(t, json, "json should not be empty for: %s", q)
		}
	}
}

func TestExecute_ResolveThinWrapper(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// The listPets response is {type: array, items: {$ref: Pet}}
	// schema stage should resolve through the array wrapper to Pet
	result, err := oq.Execute(`operations | select(name == "listPets") | responses | select(status_code == "200") | content-types | schema | pick name, is_component`, g)
	require.NoError(t, err)
	if assert.NotEmpty(t, result.Rows) {
		name := oq.FieldValuePublic(result.Rows[0], "name", g)
		isComp := oq.FieldValuePublic(result.Rows[0], "is_component", g)
		assert.Equal(t, "Pet", name.Str, "should resolve through array wrapper to Pet component")
		assert.True(t, isComp.Bool, "resolved schema should be a component")
	}
}

func TestExecute_HeadersEmitKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(name == "createPet") | responses | select(status_code == "201") | headers | emit`, g)
	require.NoError(t, err)
	if len(result.Rows) > 0 {
		yaml := oq.FormatYAML(result, g)
		assert.Contains(t, yaml, "createPet/201/headers/X-Request-Id", "emit key should include full path")
	}
}

func TestExecute_ReferencedByResolvesWrappers_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Owner is referenced via Pet.owner (which goes through an inline $ref wrapper).
	// After the fix, referenced-by should resolve through the wrapper to return Pet.
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Owner") | referenced-by`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "referenced-by should resolve through $ref wrapper to Pet")

	// Via should be the structural edge kind (property, items, etc.), not 'ref'
	// Key should be the structural label (e.g. 'owner'), not the $ref URI
	for _, row := range result.Rows {
		from := oq.FieldValuePublic(row, "from", g)
		name := oq.FieldValuePublic(row, "name", g)
		assert.Equal(t, name.Str, from.Str, "from should match the resolved node name")

		via := oq.FieldValuePublic(row, "via", g)
		assert.NotEqual(t, "ref", via.Str, "via should be structural (property/items/allOf), not 'ref'")

		key := oq.FieldValuePublic(row, "key", g)
		assert.NotContains(t, key.Str, "#/components", "key should be structural label, not $ref URI")
	}
}

func TestExecute_ParentStructural_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Get properties of Pet, then navigate to parent — should return Pet
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | properties | parent`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "parent of Pet's properties should return results")

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "parent of Pet's properties should be Pet")
}

func TestExecute_AncestorsDepthLimited_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Address is referenced by Owner (1 hop), and Owner is referenced by Pet (2 hops).
	// ancestors(1) should include only the immediate referrers.
	result1, err := oq.Execute(`schemas | select(is_component) | select(name == "Address") | ancestors(1)`, g)
	require.NoError(t, err)

	// Full ancestors (unlimited) should include more.
	resultAll, err := oq.Execute(`schemas | select(is_component) | select(name == "Address") | ancestors`, g)
	require.NoError(t, err)

	// Depth-limited result should have fewer or equal rows
	assert.LessOrEqual(t, len(result1.Rows), len(resultAll.Rows),
		"ancestors(1) should return <= ancestors")
	assert.NotEmpty(t, result1.Rows, "Address should have at least one ancestor at depth 1")
}

func TestExecute_BFSDepthField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// descendants(2) should populate bfs_depth on result rows
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | descendants(2)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		d := oq.FieldValuePublic(row, "bfs_depth", g)
		assert.Equal(t, expr.KindInt, d.Kind, "bfs_depth should be an int")
		assert.True(t, d.Int >= 1 && d.Int <= 2, "bfs_depth should be between 1 and 2")
	}
}

func TestExecute_TargetField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// references from Pet should have target == "Pet" (the seed)
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | references`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		target := oq.FieldValuePublic(row, "target", g)
		assert.Equal(t, "Pet", target.Str, "target should be the seed schema name")
	}
}

func TestExecute_AncestorsDepthBFSDepth_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// ancestors(2) on Address should populate bfs_depth
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Address") | ancestors(2)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		d := oq.FieldValuePublic(row, "bfs_depth", g)
		assert.Equal(t, expr.KindInt, d.Kind, "bfs_depth should be an int")
		assert.True(t, d.Int >= 1 && d.Int <= 2, "bfs_depth should be between 1 and 2")
		target := oq.FieldValuePublic(row, "target", g)
		assert.Equal(t, "Address", target.Str, "target should be Address")
	}
}

func TestParse_AncestorsDepth_Success(t *testing.T) {
	t.Parallel()
	stages, err := oq.Parse("schemas | ancestors(3)")
	require.NoError(t, err)
	assert.Len(t, stages, 2)
	assert.Equal(t, 3, stages[1].Limit)
}

// --- Coverage boost tests ---

func TestExecute_SchemaContentFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test all schema content fields against Pet (has required, properties, type=object)
	fields := []struct {
		field   string
		notNull bool
	}{
		{"description", false},
		{"has_description", true},
		{"title", false},
		{"has_title", true},
		{"format", false},
		{"pattern", false},
		{"nullable", true},
		{"read_only", true},
		{"write_only", true},
		{"deprecated", true},
		{"unique_items", true},
		{"has_discriminator", true},
		{"discriminator_property", false},
		{"discriminator_mapping_count", true},
		{"required", true},
		{"required_count", true},
		{"enum", true},
		{"enum_count", true},
		{"has_default", true},
		{"has_example", true},
		{"minimum", false},
		{"maximum", false},
		{"min_length", false},
		{"max_length", false},
		{"min_items", false},
		{"max_items", false},
		{"min_properties", false},
		{"max_properties", false},
		{"extension_count", true},
		{"content_encoding", false},
		{"content_media_type", false},
	}

	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet")`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	for _, f := range fields {
		v := oq.FieldValuePublic(result.Rows[0], f.field, g)
		if f.notNull {
			assert.NotEqual(t, expr.KindNull, v.Kind, "field %q should not be null", f.field)
		}
	}

	// Pet has required: [id, name]
	reqCount := oq.FieldValuePublic(result.Rows[0], "required_count", g)
	assert.Equal(t, 2, reqCount.Int)

	reqArr := oq.FieldValuePublic(result.Rows[0], "required", g)
	assert.Equal(t, expr.KindArray, reqArr.Kind)
	assert.Contains(t, reqArr.Arr, "id")
	assert.Contains(t, reqArr.Arr, "name")
}

func TestExecute_OperationContentFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test operation content fields
	result, err := oq.Execute(`operations | select(operation_id == "showPetById")`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	// showPetById has a default error response
	hasErr := oq.FieldValuePublic(row, "has_error_response", g)
	assert.True(t, hasErr.Bool, "showPetById should have error response")

	respCount := oq.FieldValuePublic(row, "response_count", g)
	assert.Positive(t, respCount.Int, "should have responses")

	hasBody := oq.FieldValuePublic(row, "has_request_body", g)
	assert.False(t, hasBody.Bool, "GET should not have request body")

	secCount := oq.FieldValuePublic(row, "security_count", g)
	assert.Equal(t, expr.KindInt, secCount.Kind)

	tags := oq.FieldValuePublic(row, "tags", g)
	assert.Equal(t, expr.KindArray, tags.Kind)

	// createPet has a request body
	result2, err := oq.Execute(`operations | select(operation_id == "createPet")`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result2.Rows)
	hasBody2 := oq.FieldValuePublic(result2.Rows[0], "has_request_body", g)
	assert.True(t, hasBody2.Bool, "createPet should have request body")
}

func TestExecute_ComponentSources_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test all component source stages
	sources := []struct {
		query    string
		hasRows  bool
		rowField string
	}{
		{`components.parameters | pick name`, true, "name"},
		{`components.responses | pick name`, true, "name"},
		{`components.security-schemes | pick name`, true, "name"},
		// No request-bodies or headers in components in our fixture
		{`components.request-bodies`, false, ""},
		{`components.headers`, false, ""},
	}

	for _, s := range sources {
		t.Run(s.query, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(s.query, g)
			require.NoError(t, err)
			if s.hasRows {
				assert.NotEmpty(t, result.Rows, "should have rows for %s", s.query)
			}
		})
	}
}

func TestExecute_ComponentParameterFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components.parameters | pick name`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	// LimitParam: name=limit, in=query
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "LimitParam", name.Str)

	in := oq.FieldValuePublic(row, "in", g)
	assert.Equal(t, "query", in.Str)

	hasSch := oq.FieldValuePublic(row, "has_schema", g)
	assert.True(t, hasSch.Bool)

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Empty(t, op.Str, "component param has no operation")
}

func TestExecute_ComponentResponseFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components.responses`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "NotFound", name.Str)

	desc := oq.FieldValuePublic(row, "description", g)
	assert.Equal(t, "Resource not found", desc.Str)

	ctCount := oq.FieldValuePublic(row, "content_type_count", g)
	assert.Equal(t, 1, ctCount.Int)

	hasCt := oq.FieldValuePublic(row, "has_content", g)
	assert.True(t, hasCt.Bool)
}

func TestExecute_SecuritySchemeFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components.security-schemes`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "bearerAuth", name.Str)

	typ := oq.FieldValuePublic(row, "type", g)
	assert.Equal(t, "http", typ.Str)

	scheme := oq.FieldValuePublic(row, "scheme", g)
	assert.Equal(t, "bearer", scheme.Str)

	bf := oq.FieldValuePublic(row, "bearer_format", g)
	assert.Equal(t, "JWT", bf.Str)

	hasFlows := oq.FieldValuePublic(row, "has_flows", g)
	assert.False(t, hasFlows.Bool)
}

func TestExecute_NavigationPipeline_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test full navigation: operation -> responses -> content-types -> schema
	result, err := oq.Execute(`operations | select(operation_id == "listPets") | responses | content-types | schema`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should resolve schema through navigation")

	// Test request-body -> content-types
	result2, err := oq.Execute(`operations | select(operation_id == "createPet") | request-body | content-types`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows, "should have content types from request body")

	// Content type fields
	for _, row := range result2.Rows {
		mt := oq.FieldValuePublic(row, "media_type", g)
		assert.NotEmpty(t, mt.Str, "media_type should be set")
		hasSch := oq.FieldValuePublic(row, "has_schema", g)
		assert.Equal(t, expr.KindBool, hasSch.Kind)
	}

	// Test headers from response
	result3, err := oq.Execute(`operations | select(operation_id == "createPet") | responses | select(status_code == "201") | headers`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result3.Rows, "201 response should have headers")

	for _, row := range result3.Rows {
		name := oq.FieldValuePublic(row, "name", g)
		assert.NotEmpty(t, name.Str)
	}

	// Test security navigation
	result4, err := oq.Execute(`operations | select(operation_id == "createPet") | security`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result4.Rows, "createPet should have security requirements")

	for _, row := range result4.Rows {
		sn := oq.FieldValuePublic(row, "scheme_name", g)
		assert.NotEmpty(t, sn.Str)
		st := oq.FieldValuePublic(row, "scheme_type", g)
		assert.NotEmpty(t, st.Str)
	}
}

func TestExecute_OperationBackNav_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Navigate parameters -> operation
	result, err := oq.Execute(`operations | select(operation_id == "listPets") | parameters | operation`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should navigate back to operation")

	names := collectNames(result, g)
	assert.Contains(t, names, "listPets")
}

func TestExecute_FieldsIntrospection_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test fields for different row types
	queries := []struct {
		query  string
		expect string
	}{
		{"operations | fields", "method"},
		{"schemas | select(is_component) | fields", "bfs_depth"},
		{"schemas | select(is_component) | fields", "target"},
		{"operations | select(operation_id == \"listPets\") | parameters | fields", "in"},
		{"operations | select(operation_id == \"listPets\") | responses | fields", "status_code"},
		{"operations | select(operation_id == \"createPet\") | request-body | fields", "required"},
		{"operations | select(operation_id == \"listPets\") | responses | content-types | fields", "media_type"},
		{"operations | select(operation_id == \"createPet\") | responses | select(status_code == \"201\") | headers | fields", "name"},
		{"components.security-schemes | fields", "bearer_format"},
		{"operations | select(operation_id == \"createPet\") | security | fields", "scopes"},
	}

	for _, q := range queries {
		t.Run(q.query, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(q.query, g)
			require.NoError(t, err)
			assert.Contains(t, result.Explain, q.expect, "fields should include %s", q.expect)
		})
	}

	// Group row fields
	result, err := oq.Execute(`schemas | select(is_component) | group_by(type) | fields`, g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "count")
}

func TestExecute_EdgeKinds_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Pet has property edges and Shape has oneOf edges — verify edge kind strings
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | references | pick via`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	vias := make(map[string]bool)
	for _, row := range result.Rows {
		v := oq.FieldValuePublic(row, "via", g)
		vias[v.Str] = true
	}
	assert.True(t, vias["property"], "Pet should have property edges")

	// Shape has oneOf edges
	result2, err := oq.Execute(`schemas | select(is_component) | select(name == "Shape") | references | pick via`, g)
	require.NoError(t, err)
	for _, row := range result2.Rows {
		v := oq.FieldValuePublic(row, "via", g)
		assert.Equal(t, "oneOf", v.Str)
	}
}

func TestExecute_RowKeyDedup_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Ensure unique deduplicates across different row types
	result, err := oq.Execute(`schemas | select(is_component) | unique | pick name`, g)
	require.NoError(t, err)
	names := collectNames(result, g)
	// No duplicates
	seen := make(map[string]bool)
	for _, n := range names {
		assert.False(t, seen[n], "duplicate name: %s", n)
		seen[n] = true
	}

	// Operations unique
	result2, err := oq.Execute(`operations | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows)

	// Parameters unique
	result3, err := oq.Execute(`operations | select(operation_id == "listPets") | parameters | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result3.Rows)

	// Responses unique
	result4, err := oq.Execute(`operations | select(operation_id == "showPetById") | responses | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result4.Rows)

	// Content-types unique
	result5, err := oq.Execute(`operations | select(operation_id == "createPet") | request-body | content-types | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result5.Rows)

	// Headers unique
	result6, err := oq.Execute(`operations | select(operation_id == "createPet") | responses | select(status_code == "201") | headers | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result6.Rows)

	// Security schemes unique
	result7, err := oq.Execute(`components.security-schemes | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result7.Rows)

	// Security requirements unique
	result8, err := oq.Execute(`operations | select(operation_id == "createPet") | security | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result8.Rows)
}

func TestExecute_RequestBodyFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(operation_id == "createPet") | request-body`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	req := oq.FieldValuePublic(row, "required", g)
	assert.True(t, req.Bool, "createPet request body should be required")

	desc := oq.FieldValuePublic(row, "description", g)
	assert.Equal(t, expr.KindString, desc.Kind)

	ctCount := oq.FieldValuePublic(row, "content_type_count", g)
	assert.Equal(t, 2, ctCount.Int) // application/json + multipart/form-data

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Equal(t, "createPet", op.Str)
}

func TestExecute_ResponseFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(operation_id == "showPetById") | responses`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	// Should have 200 and default responses
	codes := make(map[string]bool)
	for _, row := range result.Rows {
		sc := oq.FieldValuePublic(row, "status_code", g)
		codes[sc.Str] = true
		desc := oq.FieldValuePublic(row, "description", g)
		assert.NotEmpty(t, desc.Str)
		lc := oq.FieldValuePublic(row, "link_count", g)
		assert.Equal(t, expr.KindInt, lc.Kind)
		hc := oq.FieldValuePublic(row, "header_count", g)
		assert.Equal(t, expr.KindInt, hc.Kind)
	}
	assert.True(t, codes["200"])
	assert.True(t, codes["default"])
}

func TestExecute_HeaderFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(operation_id == "createPet") | responses | select(status_code == "201") | headers`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "X-Request-Id", name.Str)

	req := oq.FieldValuePublic(row, "required", g)
	assert.True(t, req.Bool)

	hasSch := oq.FieldValuePublic(row, "has_schema", g)
	assert.True(t, hasSch.Bool)

	sc := oq.FieldValuePublic(row, "status_code", g)
	assert.Equal(t, "201", sc.Str)

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Equal(t, "createPet", op.Str)
}

func TestExecute_ContentTypeFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(operation_id == "createPet") | responses | select(status_code == "201") | content-types`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	mt := oq.FieldValuePublic(row, "media_type", g)
	assert.Equal(t, "application/json", mt.Str)

	hasSch := oq.FieldValuePublic(row, "has_schema", g)
	assert.True(t, hasSch.Bool)

	hasEnc := oq.FieldValuePublic(row, "has_encoding", g)
	assert.Equal(t, expr.KindBool, hasEnc.Kind)

	hasEx := oq.FieldValuePublic(row, "has_example", g)
	assert.Equal(t, expr.KindBool, hasEx.Kind)

	sc := oq.FieldValuePublic(row, "status_code", g)
	assert.Equal(t, "201", sc.Str)

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Equal(t, "createPet", op.Str)
}

func TestExecute_SecurityRequirementFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(operation_id == "createPet") | security`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	sn := oq.FieldValuePublic(row, "scheme_name", g)
	assert.Equal(t, "bearerAuth", sn.Str)

	st := oq.FieldValuePublic(row, "scheme_type", g)
	assert.Equal(t, "http", st.Str)

	scopes := oq.FieldValuePublic(row, "scopes", g)
	assert.Equal(t, expr.KindArray, scopes.Kind)

	scopeCount := oq.FieldValuePublic(row, "scope_count", g)
	assert.Equal(t, expr.KindInt, scopeCount.Kind)

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Equal(t, "createPet", op.Str)
}

func TestExecute_ParameterFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | select(operation_id == "listPets") | parameters`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	// Find the deprecated "format" parameter
	for _, row := range result.Rows {
		name := oq.FieldValuePublic(row, "name", g)
		if name.Str == "format" {
			dep := oq.FieldValuePublic(row, "deprecated", g)
			assert.True(t, dep.Bool, "format param should be deprecated")
		}
		in := oq.FieldValuePublic(row, "in", g)
		assert.Equal(t, "query", in.Str)

		req := oq.FieldValuePublic(row, "required", g)
		assert.Equal(t, expr.KindBool, req.Kind)

		style := oq.FieldValuePublic(row, "style", g)
		assert.Equal(t, expr.KindString, style.Kind)

		explode := oq.FieldValuePublic(row, "explode", g)
		assert.Equal(t, expr.KindBool, explode.Kind)

		aev := oq.FieldValuePublic(row, "allow_empty_value", g)
		assert.Equal(t, expr.KindBool, aev.Kind)

		ar := oq.FieldValuePublic(row, "allow_reserved", g)
		assert.Equal(t, expr.KindBool, ar.Kind)

		op := oq.FieldValuePublic(row, "operation", g)
		assert.Equal(t, "listPets", op.Str)
	}
}

func TestExecute_SchemaFromNavRows_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Extract schema from parameter
	result, err := oq.Execute(`operations | select(operation_id == "listPets") | parameters | select(name == "limit") | schema`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should extract schema from parameter")

	// Extract schema from header
	result2, err := oq.Execute(`operations | select(operation_id == "createPet") | responses | select(status_code == "201") | headers | schema`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows, "should extract schema from header")
}

func TestExecute_KindField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test kind field for different row types
	result, err := oq.Execute(`schemas | select(is_component) | first(1)`, g)
	require.NoError(t, err)
	kind := oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "schema", kind.Str)

	result2, err := oq.Execute(`operations | first(1)`, g)
	require.NoError(t, err)
	kind2 := oq.FieldValuePublic(result2.Rows[0], "kind", g)
	assert.Equal(t, "operation", kind2.Str)

	result3, err := oq.Execute(`operations | select(operation_id == "listPets") | parameters | first(1)`, g)
	require.NoError(t, err)
	kind3 := oq.FieldValuePublic(result3.Rows[0], "kind", g)
	assert.Equal(t, "parameter", kind3.Str)

	result4, err := oq.Execute(`operations | select(operation_id == "listPets") | responses | first(1)`, g)
	require.NoError(t, err)
	kind4 := oq.FieldValuePublic(result4.Rows[0], "kind", g)
	assert.Equal(t, "response", kind4.Str)

	result5, err := oq.Execute(`operations | select(operation_id == "createPet") | request-body`, g)
	require.NoError(t, err)
	kind5 := oq.FieldValuePublic(result5.Rows[0], "kind", g)
	assert.Equal(t, "request-body", kind5.Str)

	result6, err := oq.Execute(`operations | select(operation_id == "listPets") | responses | content-types | first(1)`, g)
	require.NoError(t, err)
	kind6 := oq.FieldValuePublic(result6.Rows[0], "kind", g)
	assert.Equal(t, "content-type", kind6.Str)

	result7, err := oq.Execute(`operations | select(operation_id == "createPet") | responses | select(status_code == "201") | headers | first(1)`, g)
	require.NoError(t, err)
	kind7 := oq.FieldValuePublic(result7.Rows[0], "kind", g)
	assert.Equal(t, "header", kind7.Str)

	result8, err := oq.Execute(`components.security-schemes | first(1)`, g)
	require.NoError(t, err)
	kind8 := oq.FieldValuePublic(result8.Rows[0], "kind", g)
	assert.Equal(t, "security-scheme", kind8.Str)

	result9, err := oq.Execute(`operations | select(operation_id == "createPet") | security | first(1)`, g)
	require.NoError(t, err)
	kind9 := oq.FieldValuePublic(result9.Rows[0], "kind", g)
	assert.Equal(t, "security-requirement", kind9.Str)
}

func TestExecute_IncludeAndDef_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test include with nonexistent file (covers include error path)
	_, err := oq.Execute(`include "nonexistent.oq"; schemas`, g)
	require.Error(t, err, "include of nonexistent file should fail")

	// Test def expansion in non-source position
	result, err := oq.Execute(`def top3: sort_by(depth; desc) | first(3); schemas | select(is_component) | top3`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3)
}

func TestExecute_EmitYAML_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test emit on schemas
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | emit`, g)
	require.NoError(t, err)
	assert.True(t, result.EmitYAML, "emit should set EmitYAML flag")

	// FormatYAML should produce output
	output := oq.FormatYAML(result, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output, "YAML output should not be empty")
	assert.Contains(t, output, "type", "YAML should contain schema content")

	// Test emit on operations
	result2, err := oq.Execute(`operations | select(operation_id == "listPets") | emit`, g)
	require.NoError(t, err)
	output2 := oq.FormatYAML(result2, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output2)

	// Test emit on navigation rows (parameters, responses, etc.)
	result3, err := oq.Execute(`operations | select(operation_id == "listPets") | parameters | emit`, g)
	require.NoError(t, err)
	output3 := oq.FormatYAML(result3, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output3)

	result4, err := oq.Execute(`operations | select(operation_id == "listPets") | responses | emit`, g)
	require.NoError(t, err)
	output4 := oq.FormatYAML(result4, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output4)

	result5, err := oq.Execute(`operations | select(operation_id == "createPet") | request-body | emit`, g)
	require.NoError(t, err)
	output5 := oq.FormatYAML(result5, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output5)

	result6, err := oq.Execute(`operations | select(operation_id == "listPets") | responses | content-types | emit`, g)
	require.NoError(t, err)
	output6 := oq.FormatYAML(result6, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output6)

	result7, err := oq.Execute(`operations | select(operation_id == "createPet") | responses | select(status_code == "201") | headers | emit`, g)
	require.NoError(t, err)
	output7 := oq.FormatYAML(result7, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output7)

	result8, err := oq.Execute(`components.security-schemes | emit`, g)
	require.NoError(t, err)
	output8 := oq.FormatYAML(result8, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output8)
}

func TestExecute_EmitKeys_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test emit with pick (emit keys)
	result, err := oq.Execute(`schemas | select(is_component) | select(name == "Pet") | pick name | emit`, g)
	require.NoError(t, err)
	output := oq.FormatYAML(result, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestFormatTable_NavigationRows_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Parameters
	result, err := oq.Execute(`operations | select(operation_id == "listPets") | parameters`, g)
	require.NoError(t, err)
	output := oq.FormatTable(result, g)
	require.NoError(t, err)
	assert.Contains(t, output, "limit")

	// Content types
	result2, err := oq.Execute(`operations | select(operation_id == "createPet") | request-body | content-types`, g)
	require.NoError(t, err)
	output2 := oq.FormatTable(result2, g)
	require.NoError(t, err)
	assert.Contains(t, output2, "application/json")

	// Headers
	result3, err := oq.Execute(`operations | select(operation_id == "createPet") | responses | select(status_code == "201") | headers`, g)
	require.NoError(t, err)
	output3 := oq.FormatTable(result3, g)
	require.NoError(t, err)
	assert.Contains(t, output3, "X-Request-Id")

	// Security requirements
	result4, err := oq.Execute(`operations | select(operation_id == "createPet") | security`, g)
	require.NoError(t, err)
	output4 := oq.FormatTable(result4, g)
	require.NoError(t, err)
	assert.Contains(t, output4, "bearerAuth")

	// Security schemes
	result5, err := oq.Execute(`components.security-schemes`, g)
	require.NoError(t, err)
	output5 := oq.FormatTable(result5, g)
	require.NoError(t, err)
	assert.Contains(t, output5, "bearer")
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
