package oq_test

import (
	"fmt"
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
		{"components source", "schemas | where(isComponent)"},
		{"inline source", "schemas | where(isInline)"},
		{"operations source", "operations"},
		{"sort-by", "schemas | sort-by(depth, desc)"},
		{"take", "schemas | take(5)"},
		{"where", "schemas | where(depth > 3)"},
		{"select", "schemas | select name, depth"},
		{"length", "schemas | length"},
		{"unique", "schemas | unique"},
		{"group-by", "schemas | group-by(hash)"},
		{"refs(out)", "schemas | refs(out)"},
		{"refs(in) 1-hop", "schemas | refs(in)"},
		{"refs(out)", "schemas | refs(out)"},
		{"refs(in)", "schemas | refs(in)"},
		{"refs(in) closure", "schemas | refs(in, *)"},
		{"properties", "schemas | properties"},
		{"members", "schemas | members"},
		{"items", "schemas | items"},
		{"to-operations", "schemas | to-operations"},
		{"schemas from ops", "operations | to-schemas"},
		{"blast-radius", "schemas | where(isComponent) | where(name == \"Pet\") | blast-radius"},
		{"refs", "schemas | where(isComponent) | where(name == \"Pet\") | refs"},
		{"orphans", "schemas | where(isComponent) | orphans"},
		{"leaves", "schemas | where(isComponent) | leaves"},
		{"cycles", "schemas | cycles"},
		{"clusters", "schemas | where(isComponent) | clusters"},
		{"cross-tag", "schemas | cross-tag"},
		{"shared-refs", "operations | take(2) | shared-refs"},
		{"full pipeline", "schemas | where(isComponent) | where(depth > 0) | sort-by(depth, desc) | take(5) | select name, depth"},
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

	result, err := oq.Execute("schemas | where(isComponent) | select name", g)
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

	result, err := oq.Execute(`schemas | where(isComponent) | where(type == "object") | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "should include Pet schema")
	assert.Contains(t, names, "Owner", "should include Owner schema")
}

func TestExecute_WhereInDegree_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Unused schema has no incoming references (from other schemas in components)
	result, err := oq.Execute(`schemas | where(isComponent) | where(inDegree == 0) | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Unused should have no references from other schemas
	assert.Contains(t, names, "Unused", "should include Unused schema with inDegree 0")
}

func TestExecute_Sort_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | sort-by(propertyCount, desc) | take(3) | select name, propertyCount", g)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Rows), 3, "should return at most 3 rows")
}

func TestExecute_Reachable_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs(out, *) | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Pet references Owner, Owner references Address
	assert.Contains(t, names, "Owner", "Pet should reach Owner")
	assert.Contains(t, names, "Address", "Pet should reach Address")
}

func TestExecute_Ancestors_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Address") | refs(in, *) | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	// Address is referenced by Owner, which is referenced by Pet
	assert.Contains(t, names, "Owner", "Address ancestors should include Owner")
}

func TestExecute_Properties_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | properties | select name`, g)
	require.NoError(t, err)
	// Pet has 4 properties: id, name, tag, owner
	assert.NotEmpty(t, result.Rows, "Pet should have properties")
}

func TestExecute_UnionMembers_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Shape") | members | select name`, g)
	require.NoError(t, err)
	// Shape has oneOf with Circle and Square
	names := collectNames(result, g)
	assert.Contains(t, names, "Circle", "Shape union members should include Circle")
	assert.Contains(t, names, "Square", "Shape union members should include Square")
}

func TestExecute_Operations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | select name, method, path", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operations")
}

func TestExecute_OperationSchemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(operationId == "listPets") | to-schemas | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "listPets operation should reference Pet schema")
}

func TestExecute_GroupBy_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | group-by(type)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "should have groups")
}

func TestExecute_Unique_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | unique", g)
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

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | to-operations | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operations using Pet schema")
}

func TestFormatTable_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | take(3) | select name, type", g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.Contains(t, table, "name", "table should include name column")
	assert.Contains(t, table, "type", "table should include type column")
	assert.NotEmpty(t, table, "table should not be empty")
}

func TestFormatJSON_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | take(3) | select name, type", g)
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

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "NonExistent")`, g)
	require.NoError(t, err)

	table := oq.FormatTable(result, g)
	assert.Equal(t, "(empty)\n", table, "empty result should format as (empty)")
}

func TestExecute_MatchesExpression_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name matches ".*Error.*") | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Error", "regex match should return Error schema")
}

func TestExecute_SortAsc_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | sort-by(name) | select name", g)
	require.NoError(t, err)

	names := collectNames(result, g)
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "names should be in ascending order")
	}
}

func TestExecute_Explain_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | where(depth > 5) | sort-by(depth, desc) | take(10) | explain", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Source: schemas", "explain should show source")
	assert.Contains(t, result.Explain, "Filter: where(depth > 5)", "explain should show filter stage")
	assert.Contains(t, result.Explain, "Sort: sort-by(depth, desc)", "explain should show sort stage")
	assert.Contains(t, result.Explain, "Limit: take(10)", "explain should show limit stage")
}

func TestExecute_Fields_Schemas_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | fields", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "name", "fields output should list name")
	assert.Contains(t, result.Explain, "depth", "fields output should list depth")
	assert.Contains(t, result.Explain, "propertyCount", "fields output should list propertyCount")
	assert.Contains(t, result.Explain, "isComponent", "fields output should list isComponent")
}

func TestExecute_Fields_Operations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | fields", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "method", "fields output should list method")
	assert.Contains(t, result.Explain, "operationId", "fields output should list operationId")
	assert.Contains(t, result.Explain, "schemaCount", "fields output should list schemaCount")
	assert.Contains(t, result.Explain, "tag", "fields output should list tag")
	assert.Contains(t, result.Explain, "deprecated", "fields output should list deprecated")
}

func TestExecute_Sample_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | sample 3", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3, "sample should return exactly 3 rows")

	// Running sample again should produce the same result (deterministic)
	result2, err := oq.Execute("schemas | where(isComponent) | sample 3", g)
	require.NoError(t, err)
	assert.Len(t, result2.Rows, len(result.Rows), "sample should be deterministic")
}

func TestExecute_Path_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | path Pet Address | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "path from Pet to Address should have results")

	names := collectNames(result, g)
	// Path should include Pet, something in between, and Address
	assert.Equal(t, "Pet", names[0], "path should start at Pet")
	assert.Equal(t, "Address", names[len(names)-1], "path should end at Address")
}

func TestExecute_Path_EdgeAnnotations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Path from Pet to Address should have edge annotations on intermediate/final nodes
	result, err := oq.Execute(`schemas | path(Pet, Address) | select name, via, key, from, bfsDepth`, g)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Rows), 3, "path should have at least 3 nodes")

	// First node (Pet) has no edge annotation
	first := result.Rows[0]
	assert.Empty(t, oq.FieldValuePublic(first, "via", g).Str, "first node should have no via")
	assert.Equal(t, 0, oq.FieldValuePublic(first, "bfsDepth", g).Int, "first node should have bfsDepth 0")

	// Second node should have edge annotations from Pet
	second := result.Rows[1]
	assert.NotEmpty(t, oq.FieldValuePublic(second, "via", g).Str, "second node should have via")
	assert.Equal(t, "Pet", oq.FieldValuePublic(second, "from", g).Str, "second node should come from Pet")
	assert.Equal(t, 1, oq.FieldValuePublic(second, "bfsDepth", g).Int, "second node should have bfsDepth 1")

	// Last node (Address) should have edge annotations
	last := result.Rows[len(result.Rows)-1]
	assert.NotEmpty(t, oq.FieldValuePublic(last, "via", g).Str, "last node should have via")
	assert.Equal(t, "Address", oq.FieldValuePublic(last, "name", g).Str)
}

func TestExecute_Path_NotFound_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Unused has no outgoing edges to reach Pet
	result, err := oq.Execute(`schemas | path Unused Pet | select name`, g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows, "no path should exist from Unused to Pet")
}

func TestExecute_Top_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | highest 3 propertyCount | select name, propertyCount", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3, "highest should return exactly 3 rows")

	// Verify descending order
	for i := 1; i < len(result.Rows); i++ {
		prev := oq.FieldValuePublic(result.Rows[i-1], "propertyCount", g)
		curr := oq.FieldValuePublic(result.Rows[i], "propertyCount", g)
		assert.GreaterOrEqual(t, prev.Int, curr.Int, "highest should be in descending order")
	}
}

func TestExecute_Bottom_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | lowest 3 propertyCount | select name, propertyCount", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3, "lowest should return exactly 3 rows")

	// Verify ascending order
	for i := 1; i < len(result.Rows); i++ {
		prev := oq.FieldValuePublic(result.Rows[i-1], "propertyCount", g)
		curr := oq.FieldValuePublic(result.Rows[i], "propertyCount", g)
		assert.LessOrEqual(t, prev.Int, curr.Int, "lowest should be in ascending order")
	}
}

func TestExecute_Format_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | take(3) | format json", g)
	require.NoError(t, err)
	assert.Equal(t, "json", result.FormatHint, "format hint should be json")
}

func TestFormatMarkdown_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | take(3) | select name, type", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| name", "markdown should include name column header")
	assert.Contains(t, md, "| --- |", "markdown should include separator row")
}

func TestExecute_OperationTag_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | select name, tag, parameterCount", g)
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
		{"first bare", "schemas | take(5)"},
		{"sample", "schemas | sample 10"},
		{"path", `schemas | path "User" "Order"`},
		{"path unquoted", "schemas | path User Order"},
		{"highest", "schemas | highest 5 depth"},
		{"lowest", "schemas | lowest 5 depth"},
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

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs(out) | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "Pet should have outgoing refs")
}

func TestExecute_RefsIn_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Owner") | refs(in) | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "Owner should have incoming refs")
}

func TestExecute_Items_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// listPets response includes an array with items
	result, err := oq.Execute(`schemas | where(type == "array") | items | select name`, g)
	require.NoError(t, err)
	// May or may not have results depending on spec, but should not error
	assert.NotNil(t, result, "result should not be nil")
}

func TestExecute_EdgeAnnotations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs(out) | select name, via, key, from`, g)
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

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | blast-radius`, g)
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

func TestExecute_Refs_Bidi_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "refs (bidi) should return rows")

	// Bidi refs should have direction annotations and edge metadata
	for _, row := range result.Rows {
		dir := oq.FieldValuePublic(row, "direction", g)
		assert.Contains(t, []string{"→", "←"}, dir.Str, "bidi refs should set direction")
		via := oq.FieldValuePublic(row, "via", g)
		assert.NotEmpty(t, via.Str, "refs should set via")
	}
}

func TestExecute_Orphans_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | orphans | select name`, g)
	require.NoError(t, err)
	// Result may be empty if all schemas are referenced, that's fine
	assert.NotNil(t, result, "result should not be nil")
}

func TestExecute_Leaves_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | leaves | select name`, g)
	require.NoError(t, err)
	// Leaves are schemas with no outgoing $ref to other component schemas.
	// Schemas like Address, Circle, Square, Error, Unused should be leaves
	// (they only have primitive property children, no $ref edges).
	names := collectNames(result, g)
	assert.Contains(t, names, "Address", "Address should be a leaf")
	assert.Contains(t, names, "Circle", "Circle should be a leaf")
	assert.NotContains(t, names, "Pet", "Pet should NOT be a leaf (refs Owner)")
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

	result, err := oq.Execute(`schemas | where(isComponent) | clusters`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "should have clusters")

	// Total names across all clusters should equal component count
	total := 0
	for _, grp := range result.Groups {
		total += grp.Count
	}
	// Count component schemas
	compCount, err := oq.Execute(`schemas | where(isComponent) | length`, g)
	require.NoError(t, err)
	assert.Equal(t, compCount.Count, total, "cluster totals should equal component count")
}

func TestExecute_TagBoundary_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | cross-tag | select name, tagCount`, g)
	require.NoError(t, err)
	// All returned rows should have tagCount > 1
	for _, row := range result.Rows {
		tc := oq.FieldValuePublic(row, "tagCount", g)
		assert.Greater(t, tc.Int, 1, "cross-tag schemas should have tagCount > 1")
	}
}

func TestExecute_SharedRefs_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | shared-refs | select name`, g)
	require.NoError(t, err)
	// Schemas shared by ALL operations
	assert.NotNil(t, result, "result should not be nil")
}

func TestExecute_OpCount_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | sort-by(opCount, desc) | take(3) | select name, opCount`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have schemas sorted by opCount")
}

func TestFormatTable_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | group-by(type)", g)
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

	result, err := oq.Execute("schemas | where(isComponent) | group-by(type)", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "\"key\"", "group JSON should include key field")
	assert.Contains(t, json, "\"count\"", "group JSON should include count field")
}

func TestFormatMarkdown_Groups_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | group-by(type)", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.Contains(t, md, "| key |", "group markdown should include key column")
}

func TestExecute_InlineSource_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isInline) | length", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount, "should be a count result")
}

func TestExecute_SchemaFields_Coverage(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Select all schema fields to cover fieldValue branches
	result, err := oq.Execute("schemas | where(isComponent) | take(1) | select name, type, depth, inDegree, outDegree, unionWidth, propertyCount, isComponent, isInline, isCircular, hasRef, hash, location", g)
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
	result, err := oq.Execute("operations | take(1) | select name, method, path, operationId, schemaCount, componentCount, tag, parameterCount, deprecated, description, summary", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operation rows")
}

func TestFormatJSON_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "NonExistent")`, g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.Equal(t, "[]", json, "empty result JSON should be []")
}

func TestFormatMarkdown_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "NonExistent")`, g)
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

	result, err := oq.Execute("schemas | where(isComponent) | take(3) | select name, type", g)
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

	result, err := oq.Execute("schemas | where(isComponent) | group-by(type)", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "results[", "toon should show results header")
	assert.Contains(t, toon, "{key,count,names}:", "toon should show group fields")
}

func TestFormatToon_Empty_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "NonExistent")`, g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Equal(t, "results[0]:\n", toon, "empty toon should show results[0]")
}

func TestFormatToon_Escaping_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Paths contain special chars like / that don't need escaping,
	// but hash values and paths are good coverage
	result, err := oq.Execute("schemas | where(isComponent) | take(1) | select name, hash, location", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.Contains(t, toon, "results[1]{name,hash,location}:", "toon should show result count and selected fields")
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
			"schemas | where(isComponent) | unique | length | explain",
			[]string{"Unique:", "Count:"},
		},
		{
			"explain with group-by",
			"schemas | where(isComponent) | group-by(type) | explain",
			[]string{"Group: group-by("},
		},
		{
			"explain with refs(out) 1-hop",
			"schemas | where(isComponent) | where(name == \"Pet\") | refs(out) | explain",
			[]string{"Traverse: refs(out) 1 hop"},
		},
		{
			"explain with refs(in) 1-hop",
			"schemas | where(isComponent) | where(name == \"Owner\") | refs(in) | explain",
			[]string{"Traverse: refs(in) 1 hop"},
		},
		{
			"explain with refs(out) full",
			"schemas | where(isComponent) | where(name == \"Pet\") | refs(out, *) | explain",
			[]string{"Traverse: refs(out, *) full closure"},
		},
		{
			"explain with refs(in) full",
			"schemas | where(isComponent) | where(name == \"Address\") | refs(in, *) | explain",
			[]string{"Traverse: refs(in, *) full closure"},
		},
		{
			"explain with properties",
			"schemas | where(isComponent) | where(name == \"Pet\") | properties | explain",
			[]string{"Traverse: property children"},
		},
		{
			"explain with members",
			"schemas | where(isComponent) | where(name == \"Shape\") | members | explain",
			[]string{"Expand: union members"},
		},
		{
			"explain with items",
			"schemas | where(type == \"array\") | items | explain",
			[]string{"Traverse: array items"},
		},
		{
			"explain with to-operations",
			"schemas | where(isComponent) | where(name == \"Pet\") | to-operations | explain",
			[]string{"Navigate: schemas to operations"},
		},
		{
			"explain with schemas from ops",
			"operations | to-schemas | explain",
			[]string{"Navigate: operations to schemas"},
		},
		{
			"explain with sample",
			"schemas | where(isComponent) | sample 3 | explain",
			[]string{"Sample: random 3"},
		},
		{
			"explain with path",
			"schemas | path Pet Address | explain",
			[]string{"Path: shortest path from Pet to Address"},
		},
		{
			"explain with highest",
			"schemas | where(isComponent) | highest 3 depth | explain",
			[]string{"Highest: 3 by depth"},
		},
		{
			"explain with lowest",
			"schemas | where(isComponent) | lowest 3 depth | explain",
			[]string{"Lowest: 3 by depth"},
		},
		{
			"explain with format",
			"schemas | where(isComponent) | format json | explain",
			[]string{"Format: json"},
		},
		{
			"explain with blast-radius",
			"schemas | where(isComponent) | where(name == \"Pet\") | blast-radius | explain",
			[]string{"Traverse: blast radius"},
		},
		{
			"explain with refs",
			"schemas | where(isComponent) | where(name == \"Pet\") | refs | explain",
			[]string{"Traverse: refs(bidi) 1 hop"},
		},
		{
			"explain with orphans",
			"schemas | where(isComponent) | orphans | explain",
			[]string{"Filter: schemas with no incoming"},
		},
		{
			"explain with leaves",
			"schemas | where(isComponent) | leaves | explain",
			[]string{"Filter: schemas with no $ref to component"},
		},
		{
			"explain with cycles",
			"schemas | cycles | explain",
			[]string{"Analyze: strongly connected"},
		},
		{
			"explain with clusters",
			"schemas | where(isComponent) | clusters | explain",
			[]string{"Analyze: weakly connected"},
		},
		{
			"explain with cross-tag",
			"schemas | cross-tag | explain",
			[]string{"Filter: schemas used by operations across multiple"},
		},
		{
			"explain with shared-refs",
			"operations | shared-refs | explain",
			[]string{"Analyze: schemas shared"},
		},
		{
			"explain with refs(in) 1-hop",
			"schemas | where(isComponent) | where(name == \"Address\") | refs(in) | explain",
			[]string{"Traverse: refs(in) 1 hop"},
		},
		{
			"explain with refs(out) 1-hop",
			"schemas | where(isComponent) | where(name == \"Pet\") | refs(out) | explain",
			[]string{"Traverse: refs(out) 1 hop"},
		},
		{
			"explain with parent",
			"schemas | where(isComponent) | where(name == \"Pet\") | properties | parent | explain",
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
	result, err := oq.Execute("operations | take(1) | select name, tag, parameterCount, deprecated, description, summary", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have operation rows")

	// Test edge fields on non-traversal rows (should be empty strings)
	result, err = oq.Execute("schemas | where(isComponent) | take(1) | select name, via, key, from", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have schema rows")
	viaVal := oq.FieldValuePublic(result.Rows[0], "via", g)
	assert.Empty(t, viaVal.Str, "via should be empty for non-traversal rows")

	// Test tagCount field
	result, err = oq.Execute("schemas | where(isComponent) | take(1) | select name, tagCount", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have rows for tagCount test")

	// Test opCount field
	result, err = oq.Execute("schemas | where(isComponent) | take(1) | select name, opCount", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have rows for opCount test")

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
	result, err := oq.Execute("operations | shared-refs | select name", g)
	require.NoError(t, err)
	assert.NotNil(t, result, "result should not be nil")
}

func TestFormatToon_SpecialChars(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test TOON format with bool and int fields to cover toonValue branches
	result, err := oq.Execute("schemas | where(isComponent) | take(1) | select name, depth, isComponent", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon, "toon output should not be empty")
	assert.Contains(t, toon, "results[1]", "toon should show one result")
}

func TestFormatJSON_Operations(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | take(2) | select name, method, path", g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.True(t, strings.HasPrefix(json, "["), "JSON output should start with [")
	assert.Contains(t, json, "\"name\"", "JSON should include name field")
	assert.Contains(t, json, "\"method\"", "JSON should include method field")
}

func TestFormatMarkdown_Operations(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | take(2) | select name, method", g)
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
		{"refs non-integer", "schemas | refs(abc)"},
		{"top missing field", "schemas | top 5"},
		{"bottom missing field", "schemas | bottom 5"},
		{"path missing args", "schemas | path"},
		{"path one arg", "schemas | path Pet"},
		{"select empty expr", "schemas | where()"},
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
		{"sort-by", "schemas | sort-by(name)"},
		{"select single field", "schemas | select name"},
		{"select many fields", "schemas | select name, type, depth, inDegree"},
		{"select with string", `schemas | where(name == "Pet")`},
		{"select with bool", "schemas | where(isComponent)"},
		{"select with not", "schemas | where(not isInline)"},
		{"select with has", "schemas | where(has(hash))"},
		{"select with matches", `schemas | where(name matches ".*Pet.*")`},
		{"path quoted", `schemas | path "Pet" "Address"`},
		{"shared-refs stage", "operations | take(2) | shared-refs"},
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
	result, err := oq.Execute(`schemas | where(isComponent) | where(depth > 0 and isComponent)`, g)
	require.NoError(t, err)
	assert.NotNil(t, result, "result should not be nil")

	result, err = oq.Execute(`schemas | where(isComponent) | where(depth > 100 or isComponent)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "or should match isComponent=true schemas")
}

func TestExecute_SortStringField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Sort by string field
	result, err := oq.Execute("schemas | where(isComponent) | sort-by(type) | select name, type", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have schemas sorted by type")
}

func TestExecute_GroupBy_Type_Details(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | group-by(type)", g)
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

	result, err := oq.Execute("schemas | where(isComponent) | group-by(type)", g)
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

func TestExecute_Leaves_NoComponentRefs(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | leaves | select name", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have leaf schemas")

	// Leaf schemas should not reference any component schemas
	for _, row := range result.Rows {
		refs, err := oq.Execute(
			fmt.Sprintf(`schemas | where(name == "%s") | refs(out, *) | where(isComponent)`,
				oq.FieldValuePublic(row, "name", g).Str), g)
		require.NoError(t, err)
		assert.Empty(t, refs.Rows, "leaf %s should not reference any component schemas",
			oq.FieldValuePublic(row, "name", g).Str)
	}
}

func TestExecute_OperationsTraversals(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Operations going to schemas and back
	result, err := oq.Execute("operations | take(1) | to-schemas | select name", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "operation schemas should have results")

	// Schema to operations roundtrip
	result, err = oq.Execute("schemas | where(isComponent) | where(name == \"Pet\") | to-operations | select name", g)
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
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "NodeA") | refs(out) | select name, via, key`, g)
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

	result, err := oq.Execute("schemas | where(isComponent) | where(isCircular) | select name", g)
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
	result, err := oq.Execute("operations | select name, deprecated, summary, description, tag, parameterCount", g)
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
	result, err := oq.Execute("schemas | where(isComponent) | take(1) | select name, depth, isCircular", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon, "toon output should not be empty")
}

func TestExecute_ToonEscape_SpecialChars(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// path fields contain "/" which doesn't need quoting, but let's cover the formatter
	result, err := oq.Execute("schemas | take(3) | select location", g)
	require.NoError(t, err)

	toon := oq.FormatToon(result, g)
	assert.NotEmpty(t, toon, "toon output should not be empty")
}

func TestFormatToon_Explain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(depth > 0) | explain", g)
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
		{"select filter", `schemas | where(depth > 3)`},
		{"select fields", "schemas | select name, depth"},
		{"sort-by asc", "schemas | sort-by(depth)"},
		{"sort-by desc", "schemas | sort-by(depth, desc)"},
		{"take", "schemas | take(5)"},
		{"last", "schemas | last(5)"},
		{"length", "schemas | length"},
		{"group-by", "schemas | group-by(type)"},
		{"sample call", "schemas | sample(3)"},
		{"refs bare", "schemas | refs"},
		{"refs closure", "schemas | refs(*)"},
		{"path call", "schemas | path(Pet, Address)"},
		{"highest call", "schemas | highest(3, depth)"},
		{"lowest call", "schemas | lowest(3, depth)"},
		{"format call", "schemas | format(json)"},
		{"let binding", `schemas | where(name == "Pet") | let $pet = name`},
		{"full new pipeline", `schemas | where(isComponent) | where(depth > 5) | sort-by(depth, desc) | take(10) | select name, depth`},
		{"def inline", `def hot: where(inDegree > 0); schemas | where(isComponent) | hot`},
		{"def with params", `def impact($name): where(name == $name); schemas | where(isComponent) | impact("Pet")`},
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

	result, err := oq.Execute(`schemas | where(isComponent) | where(type == "object") | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "select filter should match Pet")
	assert.Contains(t, names, "Owner", "select filter should match Owner")
}

func TestExecute_SortBy_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | sort-by(propertyCount, desc) | take(3) | select name, propertyCount", g)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Rows), 3, "should return at most 3 rows")
}

func TestExecute_First_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | take(3)", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3, "first should return exactly 3 rows")
}

func TestExecute_Last_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | last(2)", g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 2, "last should return exactly 2 rows")
}

func TestExecute_Length_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | length", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount, "length should be a count result")
	assert.Positive(t, result.Count, "count should be positive")
}

func TestExecute_GroupBy_NewSyntax_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | group-by(type)", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Groups, "should have groups")
}

func TestExecute_LetBinding_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// let $pet = name, then use $pet in subsequent filter
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | let $pet = name | refs(out, *) | where(name != $pet) | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.NotContains(t, names, "Pet", "should not include the $pet variable value")
	assert.Contains(t, names, "Owner", "should include refs(out) schemas")
}

func TestExecute_DefExpansion_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`def hot: where(inDegree > 0); schemas | where(isComponent) | hot | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "def expansion should produce results")

	// All results should have inDegree > 0
	for _, row := range result.Rows {
		v := oq.FieldValuePublic(row, "inDegree", g)
		assert.Positive(t, v.Int, "hot filter should require inDegree > 0")
	}
}

func TestExecute_DefWithParams_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`def impact($name): where(name == $name) | blast-radius; schemas | where(isComponent) | impact("Pet")`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "parameterized def should produce results")
}

func TestExecute_AlternativeOperator_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// name // "none" — name is always set, so should not be "none"
	result, err := oq.Execute(`schemas | where(isComponent) | where(name // "none" != "none") | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "alternative operator should work")
}

func TestExecute_IfThenElse_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(if isComponent then depth >= 0 else true end) | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "if-then-else should work in select")
}

func TestExecute_ExplainNewSyntax_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(depth > 5) | sort-by(depth, desc) | take(10) | select name, depth | explain`, g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Filter: where(depth > 5)", "explain should show select filter")
	assert.Contains(t, result.Explain, "Sort: sort-by(depth, desc)", "explain should show sort-by")
	assert.Contains(t, result.Explain, "Limit: take(10)", "explain should show first")
	assert.Contains(t, result.Explain, "Project: select name, depth", "explain should show pick")
}

func TestExecute_ExplainLast_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | where(isComponent) | last(3) | explain", g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Limit: last(3)", "explain should show last")
}

func TestExecute_ExplainLet_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | let $pet = name | explain`, g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "Bind: let $pet = name", "explain should show let binding")
}

func TestParse_NewSyntax_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"where call empty", "schemas | where()"},
		{"sort-by no parens", "schemas | sort-by depth"},
		{"group-by no parens", "schemas | group-by type"},
		{"let no dollar", "schemas | let x = name"},
		{"let no equals", "schemas | let $x name"},
		{"let empty expr", "schemas | let $x ="},
		{"def missing colon", "def hot where(depth > 0); schemas | hot"},
		{"def missing semicolon", "def hot: where(depth > 0) schemas | hot"},
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

	result, err := oq.Execute("operations | take(1) | responses", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should find responses")

	for _, row := range result.Rows {
		assert.Equal(t, oq.ResponseResult, row.Kind)
		sc := oq.FieldValuePublic(row, "statusCode", g)
		assert.NotEmpty(t, sc.Str, "response should have statusCode")
	}
}

func TestExecute_ContentTypes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("operations | take(1) | responses | content-types", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should find content types")

	for _, row := range result.Rows {
		assert.Equal(t, oq.ContentTypeResult, row.Kind)
		mt := oq.FieldValuePublic(row, "mediaType", g)
		assert.NotEmpty(t, mt.Str, "content type should have mediaType")
	}
}

func TestExecute_RequestBody(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// createPet has a request body
	result, err := oq.Execute(`operations | where(name == "createPet") | request-body`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 1, "createPet should have one request body")
	assert.Equal(t, oq.RequestBodyResult, result.Rows[0].Kind)

	// request-body | content-types
	ct, err := oq.Execute(`operations | where(name == "createPet") | request-body | content-types`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, ct.Rows, "request body should have content types")
}

func TestExecute_SchemaResolvesRef(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// The schema stage should resolve $ref wrappers to the component they reference
	result, err := oq.Execute("operations | take(1) | responses | content-types | to-schema", g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should resolve schemas")

	for _, row := range result.Rows {
		assert.Equal(t, oq.SchemaResult, row.Kind)
		// After resolving $ref, the schema should not be a bare $ref wrapper
		hasRef := oq.FieldValuePublic(row, "hasRef", g)
		isComp := oq.FieldValuePublic(row, "isComponent", g)
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
	result, err := oq.Execute("operations | parameters | take(1) | to-schema", g)
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

	// With select mediaType + unique, should deduplicate by the projected value
	deduped, err := oq.Execute("operations | responses | content-types | select mediaType | unique", g)
	require.NoError(t, err)
	assert.Less(t, len(deduped.Rows), len(all.Rows), "unique after select should reduce rows")

	// All remaining rows should have distinct mediaType values
	seen := make(map[string]bool)
	for _, row := range deduped.Rows {
		mt := oq.FieldValuePublic(row, "mediaType", g)
		assert.False(t, seen[mt.Str], "mediaType %q should not be duplicated", mt.Str)
		seen[mt.Str] = true
	}
}

func TestExecute_UniqueWithoutPick_UsesRowKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Without pick, unique should use row identity (original behavior)
	result, err := oq.Execute("schemas | where(isComponent) | unique | length", g)
	require.NoError(t, err)
	assert.True(t, result.IsCount)

	all, err := oq.Execute("schemas | where(isComponent) | length", g)
	require.NoError(t, err)
	assert.Equal(t, all.Count, result.Count, "unique on already-unique rows should keep all")
}

func TestExecute_NavStageOnWrongType_EmptyNotError(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// parameters on schemas should be empty, not error
	result, err := oq.Execute("schemas | take(1) | parameters", g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)

	// headers on operations (need responses first) should be empty
	result, err = oq.Execute("operations | take(1) | headers", g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)

	// content-types on schemas should be empty
	result, err = oq.Execute("schemas | take(1) | content-types", g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)
}

func TestExecute_ComponentsSources(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// components | where(kind == "schema") = schemas | where(isComponent)
	compSchemas, err := oq.Execute(`components | where(kind == "schema") | length`, g)
	require.NoError(t, err)
	filteredSchemas, err := oq.Execute("schemas | where(isComponent) | length", g)
	require.NoError(t, err)
	assert.Equal(t, filteredSchemas.Count, compSchemas.Count)

	// components | where(kind == "parameter") should work (may be 0 for petstore)
	_, err = oq.Execute(`components | where(kind == "parameter") | length`, g)
	require.NoError(t, err)

	// components | where(kind == "response") should work
	_, err = oq.Execute(`components | where(kind == "response") | length`, g)
	require.NoError(t, err)
}

func TestExecute_NavigationFullChain(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Full navigation chain: operation → responses → content-types → schema → graph traversal
	result, err := oq.Execute("operations | take(1) | responses | content-types | to-schema | refs(out) | take(3)", g)
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

	// Content-type rows should inherit statusCode from their parent response
	result, err := oq.Execute("operations | take(1) | responses | content-types | select statusCode, mediaType, operation", g)
	require.NoError(t, err)
	for _, row := range result.Rows {
		sc := oq.FieldValuePublic(row, "statusCode", g)
		assert.NotEmpty(t, sc.Str, "content-type should inherit statusCode")
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
		{"to-schema", "operations | parameters | to-schema"},
		{"operation", "operations | parameters | operation"},
		{"security", "operations | security"},
		{"components", "components | length"},
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
	result, err := oq.Execute(`operations | where(name == "listPets") | security`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1, "listPets should inherit global security")
	assert.Equal(t, oq.SecurityRequirementResult, result.Rows[0].Kind)

	schemeName := oq.FieldValuePublic(result.Rows[0], "schemeName", g)
	assert.Equal(t, "bearerAuth", schemeName.Str)

	schemeType := oq.FieldValuePublic(result.Rows[0], "schemeType", g)
	assert.Equal(t, "http", schemeType.Str)
}

func TestExecute_SecurityPerOperation(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// createPet has per-operation security with scopes
	result, err := oq.Execute(`operations | where(name == "createPet") | security`, g)
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
	result, err := oq.Execute(`operations | where(name == "streamEvents") | security`, g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows, "streamEvents should have no security (explicit opt-out)")
}

func TestExecute_ComponentsSecuritySchemes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components | where(kind == "security-scheme")`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "bearerAuth", name.Str)

	schemeType := oq.FieldValuePublic(result.Rows[0], "schemeType", g)
	assert.Equal(t, "http", schemeType.Str)

	scheme := oq.FieldValuePublic(result.Rows[0], "scheme", g)
	assert.Equal(t, "bearer", scheme.Str)
}

func TestExecute_ComponentsParameters(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components | where(kind == "parameter")`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "LimitParam", name.Str)
}

func TestExecute_ComponentsResponses(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components | where(kind == "response")`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	// Component responses have name (component key) and empty statusCode
	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "NotFound", name.Str)

	sc := oq.FieldValuePublic(result.Rows[0], "statusCode", g)
	assert.Empty(t, sc.Str, "component responses should not have statusCode")
}

func TestExecute_GroupByWithNameField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// group-by(statusCode, operation) collects operation names instead of status codes
	result, err := oq.Execute("operations | responses | group-by(statusCode, operation)", g)
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

	result, err := oq.Execute("operations | parameters | where(deprecated) | select name, operation", g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "format", name.Str)
}

func TestExecute_SSEContentType(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | responses | content-types | where(mediaType == "text/event-stream") | operation | unique`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "streamEvents", name.Str)
}

func TestExecute_MultipleContentTypes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// createPet request body has both application/json and multipart/form-data
	result, err := oq.Execute(`operations | where(name == "createPet") | request-body | content-types | select mediaType`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 2)

	types := make([]string, len(result.Rows))
	for i, row := range result.Rows {
		types[i] = oq.FieldValuePublic(row, "mediaType", g).Str
	}
	assert.Contains(t, types, "application/json")
	assert.Contains(t, types, "multipart/form-data")
}

func TestExecute_ResponseHeaders(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// createPet 201 response has X-Request-Id header
	result, err := oq.Execute(`operations | where(name == "createPet") | responses | where(statusCode == "201") | headers`, g)
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

	// Unique after select should deduplicate by projected values
	result, err := oq.Execute("operations | responses | content-types | select mediaType | unique", g)
	require.NoError(t, err)

	seen := make(map[string]bool)
	for _, row := range result.Rows {
		mt := oq.FieldValuePublic(row, "mediaType", g).Str
		assert.False(t, seen[mt], "duplicate mediaType %q after unique", mt)
		seen[mt] = true
	}
	// Should have application/json and text/event-stream
	assert.True(t, seen["application/json"])
	assert.True(t, seen["text/event-stream"])
}

func TestEval_InfixStartswith(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | where(name startswith "S") | select name`, g)
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

	result, err := oq.Execute(`schemas | where(isComponent) | where(name endswith "er") | select name`, g)
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
	result, err := oq.Execute(`schemas | where(properties contains "name") | where(isComponent) | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet")
	assert.Contains(t, names, "Owner")
	assert.NotContains(t, names, "Error", "Error has code and message, not name")
}

func TestExecute_PropertiesField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(name == "Pet") | select name, properties`, g)
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

	result, err := oq.Execute(`schemas | take(1) | select kind, name`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	kind := oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "schema", kind.Str)

	result, err = oq.Execute(`operations | take(1) | select kind, name`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	kind = oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "operation", kind.Str)
}

func TestExecute_RefsOutClosure(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// refs(out) = 1-hop only
	d1, err := oq.Execute(`schemas | where(name == "Pet") | refs(out) | length`, g)
	require.NoError(t, err)

	// refs(out, *) = full transitive closure
	dAll, err := oq.Execute(`schemas | where(name == "Pet") | refs(out, *) | length`, g)
	require.NoError(t, err)

	assert.Greater(t, dAll.Count, d1.Count, "closure should find more than 1-hop")
}

func TestExecute_MixedTypeDefaultFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// blast-radius returns mixed schema+operation rows — should show kind column
	result, err := oq.Execute(`schemas | where(name == "Pet") | blast-radius`, g)
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

	// Emit on responses should use operation/statusCode as key
	result, err := oq.Execute(`operations | take(1) | responses | take(1) | to-yaml`, g)
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
	result, err := oq.Execute(`schemas | where(name == "Pet") | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "/components/schemas/Pet")
}

func TestFormatJSON_ArrayValues(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// JSON format should render array fields as JSON arrays, not null
	result, err := oq.Execute(`schemas | where(name == "Pet") | select name, properties | format(json)`, g)
	require.NoError(t, err)

	json := oq.FormatJSON(result, g)
	assert.Contains(t, json, "[")
	assert.Contains(t, json, `"id"`)
}

func TestExecute_ArrayMatchesRegex(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// properties matches regex — any property name matching pattern
	result, err := oq.Execute(`schemas | where(properties matches "^ow") | where(isComponent) | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "Pet has 'owner' property starting with 'ow'")
}

func TestExecute_ArrayStartswith(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// properties startswith — any property name with prefix
	result, err := oq.Execute(`schemas | where(properties startswith "na") | where(isComponent) | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "Pet has 'name' property")
	assert.Contains(t, names, "Owner", "Owner has 'name' property")
}

func TestExecute_ArrayEndswith(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// properties endswith — any property name with suffix
	result, err := oq.Execute(`schemas | where(properties endswith "eet") | where(isComponent) | select name`, g)
	require.NoError(t, err)

	names := collectNames(result, g)
	assert.Contains(t, names, "Address", "Address has 'street' property")
}

func TestExecute_EmitParameterKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | parameters | take(1) | to-yaml`, g)
	require.NoError(t, err)
	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "/parameters/", "to-yaml key should include /parameters/ path")
}

func TestExecute_EmitContentTypeKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | take(1) | responses | take(1) | content-types | to-yaml`, g)
	require.NoError(t, err)
	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "application/json", "to-yaml key should include media type")
}

func TestExecute_EmitSecuritySchemeKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components | where(kind == "security-scheme") | to-yaml`, g)
	require.NoError(t, err)
	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "bearerAuth", "to-yaml key should be scheme name")
}

func TestExecute_EmitRequestBodyKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(name == "createPet") | request-body | to-yaml`, g)
	require.NoError(t, err)
	yaml := oq.FormatYAML(result, g)
	assert.Contains(t, yaml, "createPet/request-body", "to-yaml key should include operation name")
}

func TestExecute_GroupByTable_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// group-by should now render as a standard table, not summary format
	result, err := oq.Execute("schemas | where(isComponent) | group-by(type)", g)
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
		{"schemas | take(1)", "schema"},
		{"operations | take(1)", "operation"},
		{"operations | parameters | take(1)", "parameter"},
		{"operations | take(1) | responses | take(1)", "response"},
		{"operations | where(name == \"createPet\") | request-body", "request-body"},
		{"operations | take(1) | responses | take(1) | content-types | take(1)", "content-type"},
		{`components | where(kind == "security-scheme") | take(1)`, "security-scheme"},
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
		"schemas | take(1)",
		"operations | take(1)",
		"operations | parameters | take(1)",
		"operations | take(1) | responses | take(1)",
		"operations | where(name == \"createPet\") | request-body",
		"operations | take(1) | responses | take(1) | content-types | take(1)",
		`components | where(kind == "security-scheme")`,
		"operations | take(1) | security",
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
	result, err := oq.Execute(`operations | where(name == "listPets") | responses | where(statusCode == "200") | content-types | to-schema | select name, isComponent`, g)
	require.NoError(t, err)
	if assert.NotEmpty(t, result.Rows) {
		name := oq.FieldValuePublic(result.Rows[0], "name", g)
		isComp := oq.FieldValuePublic(result.Rows[0], "isComponent", g)
		assert.Equal(t, "Pet", name.Str, "should resolve through array wrapper to Pet component")
		assert.True(t, isComp.Bool, "resolved schema should be a component")
	}
}

func TestExecute_HeadersEmitKey(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(name == "createPet") | responses | where(statusCode == "201") | headers | to-yaml`, g)
	require.NoError(t, err)
	if len(result.Rows) > 0 {
		yaml := oq.FormatYAML(result, g)
		assert.Contains(t, yaml, "createPet/201/headers/X-Request-Id", "to-yaml key should include full path")
	}
}

func TestExecute_ReferencedByResolvesWrappers_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Owner is referenced via Pet.owner (which goes through an inline $ref wrapper).
	// After the fix, referenced-by should resolve through the wrapper to return Pet.
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Owner") | refs(in)`, g)
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
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | properties | parent`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "parent of Pet's properties should return results")

	names := collectNames(result, g)
	assert.Contains(t, names, "Pet", "parent of Pet's properties should be Pet")
}

func TestExecute_AncestorsDepthLimited_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// refs(in) = 1-hop
	result1, err := oq.Execute(`schemas | where(isComponent) | where(name == "Address") | refs(in)`, g)
	require.NoError(t, err)

	// refs(in, *) = full closure
	resultAll, err := oq.Execute(`schemas | where(isComponent) | where(name == "Address") | refs(in, *)`, g)
	require.NoError(t, err)

	assert.LessOrEqual(t, len(result1.Rows), len(resultAll.Rows),
		"refs(in) should return <= refs(in, *)")
	assert.NotEmpty(t, result1.Rows, "Address should have at least one ancestor")
}

func TestExecute_BFSDepthField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// refs(out) should populate edge annotations on result rows
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs(out)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		via := oq.FieldValuePublic(row, "via", g)
		assert.NotEmpty(t, via.Str, "via should be populated on refs(out) rows")
	}
}

func TestExecute_TargetField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// refs(out) from Pet should have target == "Pet" (the seed)
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs(out)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		target := oq.FieldValuePublic(row, "seed", g)
		assert.Equal(t, "Pet", target.Str, "target should be the seed schema name")
	}
}

func TestExecute_RefsInAnnotations_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// refs(in) on Owner should populate edge annotations
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Owner") | refs(in)`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		via := oq.FieldValuePublic(row, "via", g)
		assert.NotEmpty(t, via.Str, "via should be populated on refs(in) rows")
	}
}

func TestParse_RefsOutStar_Success(t *testing.T) {
	t.Parallel()
	stages, err := oq.Parse("schemas | refs(out, *)")
	require.NoError(t, err)
	assert.Len(t, stages, 2)
	assert.Equal(t, -1, stages[1].Limit)
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
		{"title", false},
		{"format", false},
		{"pattern", false},
		{"nullable", true},
		{"readOnly", true},
		{"writeOnly", true},
		{"deprecated", true},
		{"uniqueItems", true},
		{"discriminatorProperty", false},
		{"discriminatorMappingCount", true},
		{"requiredProperties", true},
		{"requiredCount", true},
		{"enum", true},
		{"enumCount", true},
		{"minimum", false},
		{"maximum", false},
		{"minLength", false},
		{"maxLength", false},
		{"minItems", false},
		{"maxItems", false},
		{"minProperties", false},
		{"maxProperties", false},
		{"extensionCount", true},
		{"contentEncoding", false},
		{"contentMediaType", false},
	}

	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet")`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	for _, f := range fields {
		v := oq.FieldValuePublic(result.Rows[0], f.field, g)
		if f.notNull {
			assert.NotEqual(t, expr.KindNull, v.Kind, "field %q should not be null", f.field)
		}
	}

	// Pet has required: [id, name]
	reqCount := oq.FieldValuePublic(result.Rows[0], "requiredCount", g)
	assert.Equal(t, 2, reqCount.Int)

	reqArr := oq.FieldValuePublic(result.Rows[0], "requiredProperties", g)
	assert.Equal(t, expr.KindArray, reqArr.Kind)
	assert.Contains(t, reqArr.Arr, "id")
	assert.Contains(t, reqArr.Arr, "name")
}

func TestExecute_OperationContentFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test operation content fields
	result, err := oq.Execute(`operations | where(operationId == "showPetById")`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	// showPetById has a default error response
	hasErr := oq.FieldValuePublic(row, "hasErrorResponse", g)
	assert.True(t, hasErr.Bool, "showPetById should have error response")

	respCount := oq.FieldValuePublic(row, "responseCount", g)
	assert.Positive(t, respCount.Int, "should have responses")

	hasBody := oq.FieldValuePublic(row, "hasRequestBody", g)
	assert.False(t, hasBody.Bool, "GET should not have request body")

	secCount := oq.FieldValuePublic(row, "securityCount", g)
	assert.Equal(t, expr.KindInt, secCount.Kind)

	tags := oq.FieldValuePublic(row, "tags", g)
	assert.Equal(t, expr.KindArray, tags.Kind)

	// createPet has a request body
	result2, err := oq.Execute(`operations | where(operationId == "createPet")`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result2.Rows)
	hasBody2 := oq.FieldValuePublic(result2.Rows[0], "hasRequestBody", g)
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
		{`components | where(kind == "parameter") | select name`, true, "name"},
		{`components | where(kind == "response") | select name`, true, "name"},
		{`components | where(kind == "security-scheme") | select name`, true, "name"},
		// No request-bodies or headers in components in our fixture
		{`components | where(kind == "request-body")`, false, ""},
		{`components | where(kind == "header")`, false, ""},
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

	result, err := oq.Execute(`components | where(kind == "parameter") | select name`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	// LimitParam: name=limit, in=query
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "LimitParam", name.Str)

	in := oq.FieldValuePublic(row, "in", g)
	assert.Equal(t, "query", in.Str)

	hasSch := oq.FieldValuePublic(row, "hasSchema", g)
	assert.True(t, hasSch.Bool)

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Empty(t, op.Str, "component param has no operation")
}

func TestExecute_ComponentResponseFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components | where(kind == "response")`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "NotFound", name.Str)

	desc := oq.FieldValuePublic(row, "description", g)
	assert.Equal(t, "Resource not found", desc.Str)

	ctCount := oq.FieldValuePublic(row, "contentTypeCount", g)
	assert.Equal(t, 1, ctCount.Int)

	hasCt := oq.FieldValuePublic(row, "hasContent", g)
	assert.True(t, hasCt.Bool)
}

func TestExecute_SecuritySchemeFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components | where(kind == "security-scheme")`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "bearerAuth", name.Str)

	typ := oq.FieldValuePublic(row, "schemeType", g)
	assert.Equal(t, "http", typ.Str)

	scheme := oq.FieldValuePublic(row, "scheme", g)
	assert.Equal(t, "bearer", scheme.Str)

	bf := oq.FieldValuePublic(row, "bearerFormat", g)
	assert.Equal(t, "JWT", bf.Str)

	hasFlows := oq.FieldValuePublic(row, "hasFlows", g)
	assert.False(t, hasFlows.Bool)
}

func TestExecute_NavigationPipeline_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test full navigation: operation -> responses -> content-types -> schema
	result, err := oq.Execute(`operations | where(operationId == "listPets") | responses | content-types | to-schema`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should resolve schema through navigation")

	// Test request-body -> content-types
	result2, err := oq.Execute(`operations | where(operationId == "createPet") | request-body | content-types`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows, "should have content types from request body")

	// Content type fields
	for _, row := range result2.Rows {
		mt := oq.FieldValuePublic(row, "mediaType", g)
		assert.NotEmpty(t, mt.Str, "mediaType should be set")
		hasSch := oq.FieldValuePublic(row, "hasSchema", g)
		assert.Equal(t, expr.KindBool, hasSch.Kind)
	}

	// Test headers from response
	result3, err := oq.Execute(`operations | where(operationId == "createPet") | responses | where(statusCode == "201") | headers`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result3.Rows, "201 response should have headers")

	for _, row := range result3.Rows {
		name := oq.FieldValuePublic(row, "name", g)
		assert.NotEmpty(t, name.Str)
	}

	// Test security navigation
	result4, err := oq.Execute(`operations | where(operationId == "createPet") | security`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result4.Rows, "createPet should have security requirements")

	for _, row := range result4.Rows {
		sn := oq.FieldValuePublic(row, "schemeName", g)
		assert.NotEmpty(t, sn.Str)
		st := oq.FieldValuePublic(row, "schemeType", g)
		assert.NotEmpty(t, st.Str)
	}
}

func TestExecute_OperationBackNav_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Navigate parameters -> operation
	result, err := oq.Execute(`operations | where(operationId == "listPets") | parameters | operation`, g)
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
		{"schemas | where(isComponent) | fields", "bfsDepth"},
		{"schemas | where(isComponent) | fields", "seed"},
		{"operations | where(operationId == \"listPets\") | parameters | fields", "in"},
		{"operations | where(operationId == \"listPets\") | responses | fields", "statusCode"},
		{"operations | where(operationId == \"createPet\") | request-body | fields", "required"},
		{"operations | where(operationId == \"listPets\") | responses | content-types | fields", "mediaType"},
		{"operations | where(operationId == \"createPet\") | responses | where(statusCode == \"201\") | headers | fields", "name"},
		{`components | where(kind == "security-scheme") | fields`, "bearerFormat"},
		{"operations | where(operationId == \"createPet\") | security | fields", "scopes"},
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
	result, err := oq.Execute(`schemas | where(isComponent) | group-by(type) | fields`, g)
	require.NoError(t, err)
	assert.Contains(t, result.Explain, "count")
}

func TestExecute_EdgeKinds_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Pet has property edges and Shape has oneOf edges — verify edge kind strings
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs(out) | select via`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	vias := make(map[string]bool)
	for _, row := range result.Rows {
		v := oq.FieldValuePublic(row, "via", g)
		vias[v.Str] = true
	}
	assert.True(t, vias["property"], "Pet should have property edges")

	// Shape has oneOf edges
	result2, err := oq.Execute(`schemas | where(isComponent) | where(name == "Shape") | refs(out) | select via`, g)
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
	result, err := oq.Execute(`schemas | where(isComponent) | unique | select name`, g)
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
	result3, err := oq.Execute(`operations | where(operationId == "listPets") | parameters | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result3.Rows)

	// Responses unique
	result4, err := oq.Execute(`operations | where(operationId == "showPetById") | responses | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result4.Rows)

	// Content-types unique
	result5, err := oq.Execute(`operations | where(operationId == "createPet") | request-body | content-types | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result5.Rows)

	// Headers unique
	result6, err := oq.Execute(`operations | where(operationId == "createPet") | responses | where(statusCode == "201") | headers | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result6.Rows)

	// Security schemes unique
	result7, err := oq.Execute(`components | where(kind == "security-scheme") | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result7.Rows)

	// Security requirements unique
	result8, err := oq.Execute(`operations | where(operationId == "createPet") | security | unique`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result8.Rows)
}

func TestExecute_RequestBodyFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(operationId == "createPet") | request-body`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	req := oq.FieldValuePublic(row, "required", g)
	assert.True(t, req.Bool, "createPet request body should be required")

	desc := oq.FieldValuePublic(row, "description", g)
	assert.Equal(t, expr.KindString, desc.Kind)

	ctCount := oq.FieldValuePublic(row, "contentTypeCount", g)
	assert.Equal(t, 2, ctCount.Int) // application/json + multipart/form-data

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Equal(t, "createPet", op.Str)
}

func TestExecute_ResponseFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(operationId == "showPetById") | responses`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	// Should have 200 and default responses
	codes := make(map[string]bool)
	for _, row := range result.Rows {
		sc := oq.FieldValuePublic(row, "statusCode", g)
		codes[sc.Str] = true
		desc := oq.FieldValuePublic(row, "description", g)
		assert.NotEmpty(t, desc.Str)
		lc := oq.FieldValuePublic(row, "linkCount", g)
		assert.Equal(t, expr.KindInt, lc.Kind)
		hc := oq.FieldValuePublic(row, "headerCount", g)
		assert.Equal(t, expr.KindInt, hc.Kind)
	}
	assert.True(t, codes["200"])
	assert.True(t, codes["default"])
}

func TestExecute_HeaderFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(operationId == "createPet") | responses | where(statusCode == "201") | headers`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "X-Request-Id", name.Str)

	req := oq.FieldValuePublic(row, "required", g)
	assert.True(t, req.Bool)

	hasSch := oq.FieldValuePublic(row, "hasSchema", g)
	assert.True(t, hasSch.Bool)

	sc := oq.FieldValuePublic(row, "statusCode", g)
	assert.Equal(t, "201", sc.Str)

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Equal(t, "createPet", op.Str)
}

func TestExecute_ContentTypeFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(operationId == "createPet") | responses | where(statusCode == "201") | content-types`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	mt := oq.FieldValuePublic(row, "mediaType", g)
	assert.Equal(t, "application/json", mt.Str)

	hasSch := oq.FieldValuePublic(row, "hasSchema", g)
	assert.True(t, hasSch.Bool)

	hasEnc := oq.FieldValuePublic(row, "hasEncoding", g)
	assert.Equal(t, expr.KindBool, hasEnc.Kind)

	hasEx := oq.FieldValuePublic(row, "hasExample", g)
	assert.Equal(t, expr.KindBool, hasEx.Kind)

	sc := oq.FieldValuePublic(row, "statusCode", g)
	assert.Equal(t, "201", sc.Str)

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Equal(t, "createPet", op.Str)
}

func TestExecute_SecurityRequirementFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(operationId == "createPet") | security`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	row := result.Rows[0]
	sn := oq.FieldValuePublic(row, "schemeName", g)
	assert.Equal(t, "bearerAuth", sn.Str)

	st := oq.FieldValuePublic(row, "schemeType", g)
	assert.Equal(t, "http", st.Str)

	scopes := oq.FieldValuePublic(row, "scopes", g)
	assert.Equal(t, expr.KindArray, scopes.Kind)

	scopeCount := oq.FieldValuePublic(row, "scopeCount", g)
	assert.Equal(t, expr.KindInt, scopeCount.Kind)

	op := oq.FieldValuePublic(row, "operation", g)
	assert.Equal(t, "createPet", op.Str)
}

func TestExecute_ParameterFields_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(operationId == "listPets") | parameters`, g)
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

		aev := oq.FieldValuePublic(row, "allowEmptyValue", g)
		assert.Equal(t, expr.KindBool, aev.Kind)

		ar := oq.FieldValuePublic(row, "allowReserved", g)
		assert.Equal(t, expr.KindBool, ar.Kind)

		op := oq.FieldValuePublic(row, "operation", g)
		assert.Equal(t, "listPets", op.Str)
	}
}

func TestExecute_SchemaFromNavRows_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Extract schema from parameter
	result, err := oq.Execute(`operations | where(operationId == "listPets") | parameters | where(name == "limit") | to-schema`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should extract schema from parameter")

	// Extract schema from header
	result2, err := oq.Execute(`operations | where(operationId == "createPet") | responses | where(statusCode == "201") | headers | to-schema`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows, "should extract schema from header")
}

func TestExecute_KindField_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test kind field for different row types
	result, err := oq.Execute(`schemas | where(isComponent) | take(1)`, g)
	require.NoError(t, err)
	kind := oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "schema", kind.Str)

	result2, err := oq.Execute(`operations | take(1)`, g)
	require.NoError(t, err)
	kind2 := oq.FieldValuePublic(result2.Rows[0], "kind", g)
	assert.Equal(t, "operation", kind2.Str)

	result3, err := oq.Execute(`operations | where(operationId == "listPets") | parameters | take(1)`, g)
	require.NoError(t, err)
	kind3 := oq.FieldValuePublic(result3.Rows[0], "kind", g)
	assert.Equal(t, "parameter", kind3.Str)

	result4, err := oq.Execute(`operations | where(operationId == "listPets") | responses | take(1)`, g)
	require.NoError(t, err)
	kind4 := oq.FieldValuePublic(result4.Rows[0], "kind", g)
	assert.Equal(t, "response", kind4.Str)

	result5, err := oq.Execute(`operations | where(operationId == "createPet") | request-body`, g)
	require.NoError(t, err)
	kind5 := oq.FieldValuePublic(result5.Rows[0], "kind", g)
	assert.Equal(t, "request-body", kind5.Str)

	result6, err := oq.Execute(`operations | where(operationId == "listPets") | responses | content-types | take(1)`, g)
	require.NoError(t, err)
	kind6 := oq.FieldValuePublic(result6.Rows[0], "kind", g)
	assert.Equal(t, "content-type", kind6.Str)

	result7, err := oq.Execute(`operations | where(operationId == "createPet") | responses | where(statusCode == "201") | headers | take(1)`, g)
	require.NoError(t, err)
	kind7 := oq.FieldValuePublic(result7.Rows[0], "kind", g)
	assert.Equal(t, "header", kind7.Str)

	result8, err := oq.Execute(`components | where(kind == "security-scheme") | take(1)`, g)
	require.NoError(t, err)
	kind8 := oq.FieldValuePublic(result8.Rows[0], "kind", g)
	assert.Equal(t, "security-scheme", kind8.Str)

	result9, err := oq.Execute(`operations | where(operationId == "createPet") | security | take(1)`, g)
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
	result, err := oq.Execute(`def top3: sort-by(depth, desc) | take(3); schemas | where(isComponent) | top3`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3)
}

func TestExecute_EmitYAML_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test emit on schemas
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | to-yaml`, g)
	require.NoError(t, err)
	assert.True(t, result.EmitYAML, "to-yaml should set EmitYAML flag")

	// FormatYAML should produce output
	output := oq.FormatYAML(result, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output, "YAML output should not be empty")
	assert.Contains(t, output, "type", "YAML should contain schema content")

	// Test emit on operations
	result2, err := oq.Execute(`operations | where(operationId == "listPets") | to-yaml`, g)
	require.NoError(t, err)
	output2 := oq.FormatYAML(result2, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output2)

	// Test emit on navigation rows (parameters, responses, etc.)
	result3, err := oq.Execute(`operations | where(operationId == "listPets") | parameters | to-yaml`, g)
	require.NoError(t, err)
	output3 := oq.FormatYAML(result3, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output3)

	result4, err := oq.Execute(`operations | where(operationId == "listPets") | responses | to-yaml`, g)
	require.NoError(t, err)
	output4 := oq.FormatYAML(result4, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output4)

	result5, err := oq.Execute(`operations | where(operationId == "createPet") | request-body | to-yaml`, g)
	require.NoError(t, err)
	output5 := oq.FormatYAML(result5, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output5)

	result6, err := oq.Execute(`operations | where(operationId == "listPets") | responses | content-types | to-yaml`, g)
	require.NoError(t, err)
	output6 := oq.FormatYAML(result6, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output6)

	result7, err := oq.Execute(`operations | where(operationId == "createPet") | responses | where(statusCode == "201") | headers | to-yaml`, g)
	require.NoError(t, err)
	output7 := oq.FormatYAML(result7, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output7)

	result8, err := oq.Execute(`components | where(kind == "security-scheme") | to-yaml`, g)
	require.NoError(t, err)
	output8 := oq.FormatYAML(result8, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output8)
}

func TestExecute_EmitKeys_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Test emit with select(emit keys)
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | select name | to-yaml`, g)
	require.NoError(t, err)
	output := oq.FormatYAML(result, g)
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestFormatTable_NavigationRows_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Parameters
	result, err := oq.Execute(`operations | where(operationId == "listPets") | parameters`, g)
	require.NoError(t, err)
	output := oq.FormatTable(result, g)
	require.NoError(t, err)
	assert.Contains(t, output, "limit")

	// Content types
	result2, err := oq.Execute(`operations | where(operationId == "createPet") | request-body | content-types`, g)
	require.NoError(t, err)
	output2 := oq.FormatTable(result2, g)
	require.NoError(t, err)
	assert.Contains(t, output2, "application/json")

	// Headers
	result3, err := oq.Execute(`operations | where(operationId == "createPet") | responses | where(statusCode == "201") | headers`, g)
	require.NoError(t, err)
	output3 := oq.FormatTable(result3, g)
	require.NoError(t, err)
	assert.Contains(t, output3, "X-Request-Id")

	// Security requirements
	result4, err := oq.Execute(`operations | where(operationId == "createPet") | security`, g)
	require.NoError(t, err)
	output4 := oq.FormatTable(result4, g)
	require.NoError(t, err)
	assert.Contains(t, output4, "bearerAuth")

	// Security schemes
	result5, err := oq.Execute(`components | where(kind == "security-scheme")`, g)
	require.NoError(t, err)
	output5 := oq.FormatTable(result5, g)
	require.NoError(t, err)
	assert.Contains(t, output5, "bearer")
}

func TestExecute_MembersDrillDown_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// clusters | members should expand cluster groups into individual schema rows
	result, err := oq.Execute(`schemas | where(isComponent) | clusters | members | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "members should expand cluster groups into schema rows")

	// All rows should be schema rows
	for _, row := range result.Rows {
		assert.Equal(t, oq.SchemaResult, row.Kind, "all member rows should be schemas")
	}

	// Total members should equal component schema count
	compResult, err := oq.Execute(`schemas | where(isComponent) | length`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, compResult.Count,
		"expanding all clusters should yield all component schemas")
}

func TestExecute_MembersSelect_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Select a single cluster and drill into its members
	result, err := oq.Execute(`schemas | where(isComponent) | clusters | take(1) | members | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "single cluster members should have rows")

	// Verify we can further filter members
	result2, err := oq.Execute(`schemas | where(isComponent) | clusters | take(1) | members | where(type == "object") | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows, "should be able to filter cluster members")
}

func TestExecute_GroupByMembers_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// group-by | where a group | members should expand to schemas
	result, err := oq.Execute(`schemas | where(isComponent) | group-by(type) | take(1) | members | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "group-by members should expand to schema rows")
}

func TestExecute_MembersNonGroup_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// members on non-group rows should return empty
	result, err := oq.Execute(`schemas | where(isComponent) | take(1) | members`, g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows, "members on non-group rows should be empty")
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

// --- Coverage improvement tests ---

func TestProbeSchemaField_CyclicSpec(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// The cyclic spec has schemas with additionalProperties, patternProperties,
	// if/then/else, not, contains, propertyNames, prefixItems, dependentSchemas.
	// Query all schemas and select these probe fields to exercise probeSchemaField.

	tests := []struct {
		name  string
		query string
	}{
		{"additionalProperties", `schemas | where(has(additionalProperties)) | select name`},
		{"patternProperties", `schemas | where(has(patternProperties)) | select name`},
		{"not", `schemas | where(has(not)) | select name`},
		{"if", `schemas | where(has(if)) | select name`},
		{"then", `schemas | where(has(then)) | select name`},
		{"else", `schemas | where(has(else)) | select name`},
		{"contains", `schemas | where(has(contains)) | select name`},
		{"propertyNames", `schemas | where(has(propertyNames)) | select name`},
		{"prefixItems", `schemas | where(has(prefixItems)) | select name`},
		{"dependentSchemas", `schemas | where(has(dependentSchemas)) | select name`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(tt.query, g)
			require.NoError(t, err)
			// These fields exist in the cyclic spec on inline schemas under NodeA
			assert.NotEmpty(t, result.Rows, "should find schemas with %s", tt.name)
		})
	}
}

func TestProbeSchemaField_NullFields(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Select probe fields that return null when not present on a component schema.
	// NodeB is a simple object with just one property, so most advanced fields are null.
	fields := []string{
		"xml", "externalDocs", "const", "multipleOf", "unevaluatedItems",
		"unevaluatedProperties", "anchor", "id", "schema", "defs", "examples",
	}
	for _, f := range fields {
		t.Run(f, func(t *testing.T) {
			t.Parallel()
			query := `schemas | where(isComponent) | where(name == "NodeB") | select name, ` + f
			result, err := oq.Execute(query, g)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Rows)
		})
	}
}

func TestProbeSchemaField_ExtensionLookup(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// x- extension lookup on a schema — should not error even if not present
	result, err := oq.Execute(`schemas | where(isComponent) | take(1) | select name, x-custom-field`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// Also test x_ variant (converted to x-)
	result2, err := oq.Execute(`schemas | where(isComponent) | take(1) | select name, x_custom_field`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows)
}

func TestParseDepthArg_Star(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// refs(out, *) uses parseDepthArg with "*"
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs(out, *) | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	// refs(in, *) uses parseDepthArg with "*"
	result2, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | refs(in, *) | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows)

	// properties(*) uses parseDepthArg with "*"
	result3, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | properties(*) | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result3.Rows)
}

func TestSchemaContentField_Constraints(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Select constraint fields that return null defaults — exercises each branch
	constraintFields := []string{
		"readOnly", "writeOnly", "uniqueItems", "discriminatorProperty",
		"minimum", "maximum", "minLength", "maxLength",
		"minItems", "maxItems", "minProperties", "maxProperties",
		"extensionCount", "contentEncoding", "contentMediaType",
	}
	for _, f := range constraintFields {
		t.Run(f, func(t *testing.T) {
			t.Parallel()
			query := `schemas | where(isComponent) | take(1) | select name, ` + f
			result, err := oq.Execute(query, g)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Rows, "should return rows with field %s", f)
		})
	}
}

func TestEdgeKindString_CyclicSpec(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Get refs(out) from ALL schemas (including inline) to capture allOf, anyOf, items edges
	result, err := oq.Execute(`schemas | refs(out) | select name, via, key`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)

	edgeKinds := make(map[string]bool)
	for _, row := range result.Rows {
		k := oq.FieldValuePublic(row, "via", g)
		edgeKinds[k.Str] = true
	}
	assert.True(t, edgeKinds["property"], "should have property edges")
}

func TestComponentRows_RequestBodiesAndHeaders(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// components | where(kind == "request-body") — petstore may or may not have them, but should not error
	result, err := oq.Execute(`components | where(kind == "request-body") | select name`, g)
	require.NoError(t, err)
	// Just verify it ran without error; may be empty

	// components | where(kind == "header") — same
	result2, err := oq.Execute(`components | where(kind == "header") | select name`, g)
	require.NoError(t, err)
	_ = result
	_ = result2
}

func TestDescribeStage_UncoveredPaths(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	tests := []struct {
		name    string
		query   string
		expects []string
	}{
		{
			"explain properties(*)",
			`schemas | where(isComponent) | where(name == "Pet") | properties(*) | explain`,
			[]string{"Traverse: properties(*) recursive"},
		},
		{
			"explain shared-refs with threshold",
			`operations | shared-refs(2) | explain`,
			[]string{"Analyze: schemas shared by at least 2"},
		},
		{
			"explain to-yaml",
			`schemas | where(isComponent) | take(1) | to-yaml | explain`,
			[]string{"Output: raw YAML"},
		},
		{
			"explain to-schema",
			`operations | take(1) | parameters | to-schema | explain`,
			[]string{"Navigate: extract schema"},
		},
		{
			"explain parameters",
			`operations | take(1) | parameters | explain`,
			[]string{"Navigate: operation parameters"},
		},
		{
			"explain responses",
			`operations | take(1) | responses | explain`,
			[]string{"Navigate: operation responses"},
		},
		{
			"explain request-body",
			`operations | take(1) | request-body | explain`,
			[]string{"Navigate: operation request body"},
		},
		{
			"explain content-types",
			`operations | take(1) | responses | content-types | explain`,
			[]string{"Navigate: content types"},
		},
		{
			"explain headers",
			`operations | take(1) | responses | headers | explain`,
			[]string{"Navigate: response headers"},
		},
		{
			"explain operation nav",
			`operations | take(1) | parameters | operation | explain`,
			[]string{"Navigate: back to source operation"},
		},
		{
			"explain security",
			`operations | take(1) | security | explain`,
			[]string{"Navigate: operation security"},
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

func TestExecLet_EmptyRows(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// let on empty result should bind null
	result, err := oq.Execute(`schemas | where(name == "NONEXISTENT") | let $x = name | select name`, g)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)
}

func TestExecLet_WithSubsequentWhere(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// let binding used in subsequent where
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "Pet") | let $t = type | where(type == $t) | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestLoadModule_Success(t *testing.T) {
	t.Parallel()

	// Create a temp module file
	dir := t.TempDir()
	modPath := dir + "/test.oq"
	err := os.WriteFile(modPath, []byte(`def big: where(depth > 5);`), 0644)
	require.NoError(t, err)

	defs, err := oq.LoadModule(modPath, nil)
	require.NoError(t, err)
	assert.Len(t, defs, 1)
	assert.Equal(t, "big", defs[0].Name)
}

func TestLoadModule_NotFound(t *testing.T) {
	t.Parallel()
	_, err := oq.LoadModule("nonexistent_module", nil)
	assert.Error(t, err)
}

func TestLoadModule_WithSearchPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := os.WriteFile(dir+"/mymod.oq", []byte(`def hello: where(type == "object");`), 0644)
	require.NoError(t, err)

	defs, err := oq.LoadModule("mymod", []string{dir})
	require.NoError(t, err)
	assert.Len(t, defs, 1)
	assert.Equal(t, "hello", defs[0].Name)
}

func TestLoadModule_AutoAppendOq(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := os.WriteFile(dir+"/auto.oq", []byte(`def auto_fn: take(1);`), 0644)
	require.NoError(t, err)

	// Should auto-append .oq extension
	defs, err := oq.LoadModule("auto", []string{dir})
	require.NoError(t, err)
	assert.Len(t, defs, 1)
}

func TestEdgeKindString_AllEdgeTypes_CyclicSpec(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Get ALL refs(out) from all schemas to hit more edge kind branches
	result, err := oq.Execute(`schemas | refs(out) | select name, via`, g)
	require.NoError(t, err)

	edgeKinds := make(map[string]bool)
	for _, row := range result.Rows {
		k := oq.FieldValuePublic(row, "via", g)
		edgeKinds[k.Str] = true
	}

	// Check for edge kinds that are present in cyclic.yaml inline schemas
	// NodeA has: additionalProperties, not, if, then, else, contains,
	// propertyNames, prefixItems, dependentSchemas, patternProperties
	for _, expected := range []string{"additionalProperties", "not", "if", "then", "else",
		"contains", "propertyNames", "prefixItems", "dependentSchema", "patternProperty"} {
		assert.True(t, edgeKinds[expected], "should have %s edge kind, got kinds: %v", expected, edgeKinds)
	}
}

func TestSchemaContentField_SelectAllConstraints(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Select many schema content fields on all schemas to exercise all branches
	query := `schemas | select name, readOnly, writeOnly, uniqueItems, discriminatorProperty, minimum, maximum, minLength, maxLength, minItems, maxItems, minProperties, maxProperties, extensionCount, contentEncoding, contentMediaType, nullable, deprecated, format, pattern, title, description, requiredCount, enumCount, discriminatorMappingCount`
	result, err := oq.Execute(query, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestProbeSchemaField_SnakeCaseConversion(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Test snake_case to camelCase conversion for probe fields
	result, err := oq.Execute(`schemas | where(has(additional_properties)) | select name`, g)
	require.NoError(t, err)
	// Should find schemas with additionalProperties via snake_case variant
	assert.NotEmpty(t, result.Rows)

	result2, err := oq.Execute(`schemas | where(has(pattern_properties)) | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows)
}

func TestComponentRows_AllSources(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Test all component sources — cyclic spec has minimal components but should not error
	sources := []string{
		`components | where(kind == "schema")`,
		`components | where(kind == "parameter")`,
		`components | where(kind == "response")`,
		`components | where(kind == "request-body")`,
		`components | where(kind == "header")`,
		`components | where(kind == "security-scheme")`,
	}
	for _, src := range sources {
		t.Run(src, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(src, g)
			require.NoError(t, err)
			_ = result
		})
	}
}

func TestComponentRows_PetstoreComponents(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Petstore has securitySchemes, parameters, and responses as components
	t.Run(`components | where(kind == "parameter")`, func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`components | where(kind == "parameter") | select name`, g)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Rows, "petstore should have component parameters")
	})

	t.Run(`components | where(kind == "response")`, func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`components | where(kind == "response") | select name`, g)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Rows, "petstore should have component responses")
	})

	t.Run(`components | where(kind == "security-scheme")`, func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`components | where(kind == "security-scheme") | select name`, g)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Rows, "petstore should have security schemes")
	})

	t.Run(`components | where(kind == "request-body") empty`, func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`components | where(kind == "request-body")`, g)
		require.NoError(t, err)
		// Petstore has no component request-bodies
		_ = result
	})

	t.Run(`components | where(kind == "header") empty`, func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`components | where(kind == "header")`, g)
		require.NoError(t, err)
		_ = result
	})
}

func TestFormatYAML_Schemas(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// to-yaml on schemas exercises FormatYAML and getRootNode for SchemaResult
	result, err := oq.Execute(`schemas | where(isComponent) | take(2) | to-yaml`, g)
	require.NoError(t, err)
	assert.True(t, result.EmitYAML)

	yaml := oq.FormatYAML(result, g)
	assert.NotEmpty(t, yaml)
}

func TestFormatYAML_Operations(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// to-yaml on operations exercises getRootNode for OperationResult
	result, err := oq.Execute(`operations | take(1) | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.NotEmpty(t, yaml)
}

func TestFormatYAML_Count(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | length`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.NotEmpty(t, yaml)
}

func TestFormatYAML_Empty(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(name == "NONEXISTENT") | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.Empty(t, yaml)
}

func TestFormatYAML_Groups(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(isComponent) | group-by(type) | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.NotEmpty(t, yaml)
}

func TestTraverseItems_CyclicSpec(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// NodeA has an "items" property which is an array with items
	result, err := oq.Execute(`schemas | where(type == "array") | items | select name, via`, g)
	require.NoError(t, err)
	// Should find items edges
	_ = result
}

func TestOperationNavigation_FullPipeline(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise parameters -> to-schema pipeline
	result, err := oq.Execute(`operations | take(1) | parameters | to-schema | select name, type`, g)
	require.NoError(t, err)
	_ = result

	// Exercise responses -> content-types pipeline
	result2, err := oq.Execute(`operations | take(1) | responses | content-types | select mediaType`, g)
	require.NoError(t, err)
	_ = result2

	// Exercise responses -> headers pipeline
	result3, err := oq.Execute(`operations | take(2) | responses | headers | select name`, g)
	require.NoError(t, err)
	_ = result3

	// Exercise request-body pipeline
	result4, err := oq.Execute(`operations | request-body | select name`, g)
	require.NoError(t, err)
	_ = result4

	// Exercise operation back-navigation
	result5, err := oq.Execute(`operations | take(1) | parameters | operation | select name`, g)
	require.NoError(t, err)
	_ = result5

	// Exercise security
	result6, err := oq.Execute(`operations | take(1) | security | select name`, g)
	require.NoError(t, err)
	_ = result6
}

func TestFieldValue_OperationFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise operation content fields
	result, err := oq.Execute(`operations | select name, responseCount, hasErrorResponse, hasRequestBody, securityCount, tags`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestFieldValue_ParameterFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise parameter fields
	result, err := oq.Execute(`operations | take(1) | parameters | select name, in, required, deprecated, description, style, explode, hasSchema, allowEmptyValue, allowReserved, operation`, g)
	require.NoError(t, err)
	_ = result
}

func TestFieldValue_ResponseFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise response fields
	result, err := oq.Execute(`operations | take(1) | responses | select statusCode, name, description, contentTypeCount, headerCount, linkCount, hasContent, operation`, g)
	require.NoError(t, err)
	_ = result
}

func TestFieldValue_ContentTypeFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise content type fields
	result, err := oq.Execute(`operations | take(1) | responses | content-types | select mediaType, name, hasSchema, hasEncoding, hasExample, statusCode, operation`, g)
	require.NoError(t, err)
	_ = result
}

func TestFieldValue_HeaderFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise header fields — petstore createPets has X-Request-Id header
	result, err := oq.Execute(`operations | responses | headers | select name, description, required, deprecated, hasSchema, statusCode, operation`, g)
	require.NoError(t, err)
	_ = result
}

func TestFieldValue_SecuritySchemeFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise security scheme fields
	result, err := oq.Execute(`components | where(kind == "security-scheme") | select name, schemeType, in, scheme, bearerFormat, description, hasFlows, deprecated`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestFieldValue_SecurityRequirementFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise security requirement fields
	result, err := oq.Execute(`operations | take(1) | security | select name, schemeName, schemeType, scopes, scopeCount, operation`, g)
	require.NoError(t, err)
	_ = result
}

func TestFieldValue_RequestBodyFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise request body fields — petstore createPets has a request body
	result, err := oq.Execute(`operations | request-body | select name, description, required, contentTypeCount, operation`, g)
	require.NoError(t, err)
	_ = result
}

func TestFieldValue_KindField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// "kind" is a universal field returning the row type
	result, err := oq.Execute(`schemas | where(isComponent) | take(1) | select kind, name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
	k := oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "schema", k.Str)

	result2, err := oq.Execute(`operations | take(1) | select kind, name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result2.Rows)
	k2 := oq.FieldValuePublic(result2.Rows[0], "kind", g)
	assert.Equal(t, "operation", k2.Str)
}

func TestRowKey_AllKinds(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Exercise unique on different row kinds to trigger rowKey for each kind
	t.Run("schema unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`schemas | where(isComponent) | unique | select name`, g)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Rows)
	})
	t.Run("operation unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`operations | unique | select name`, g)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Rows)
	})
	t.Run("parameter unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`operations | parameters | unique | select name`, g)
		require.NoError(t, err)
		_ = result
	})
	t.Run("response unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`operations | responses | unique | select name`, g)
		require.NoError(t, err)
		_ = result
	})
	t.Run("content-type unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`operations | responses | content-types | unique | select name`, g)
		require.NoError(t, err)
		_ = result
	})
	t.Run("header unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`operations | responses | headers | unique | select name`, g)
		require.NoError(t, err)
		_ = result
	})
	t.Run("security-scheme unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`components | where(kind == "security-scheme") | unique | select name`, g)
		require.NoError(t, err)
		_ = result
	})
	t.Run("security-requirement unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`operations | security | unique | select name`, g)
		require.NoError(t, err)
		_ = result
	})
	t.Run("group unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`schemas | where(isComponent) | group-by(type) | unique | select key`, g)
		require.NoError(t, err)
		_ = result
	})
	t.Run("request-body unique", func(t *testing.T) {
		t.Parallel()
		result, err := oq.Execute(`operations | request-body | unique | select name`, g)
		require.NoError(t, err)
		_ = result
	})
}

func TestPropertiesStar_Recursive(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// properties(*) recursively collects properties through allOf, oneOf, anyOf
	result, err := oq.Execute(`schemas | where(isComponent) | where(name == "NodeA") | properties(*) | select name, via, from`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestContentTypes_FromRequestBody(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// content-types can also work from request-body rows
	result, err := oq.Execute(`operations | request-body | content-types | select mediaType`, g)
	require.NoError(t, err)
	_ = result
}

func TestToSchema_FromContentType(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// to-schema from content-types
	result, err := oq.Execute(`operations | take(1) | responses | content-types | to-schema | select name, type`, g)
	require.NoError(t, err)
	_ = result
}

func TestToSchema_FromHeader(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// to-schema from headers
	result, err := oq.Execute(`operations | responses | headers | to-schema | select name, type`, g)
	require.NoError(t, err)
	_ = result
}

func TestFormatYAML_Parameters(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// to-yaml on parameters exercises getRootNode for ParameterResult
	result, err := oq.Execute(`operations | take(1) | parameters | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.NotEmpty(t, yaml)
}

func TestFormatYAML_Responses(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | take(1) | responses | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.NotEmpty(t, yaml)
}

func TestFormatYAML_RequestBody(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | request-body | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	_ = yaml
}

func TestFormatYAML_ContentTypes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | take(1) | responses | content-types | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.NotEmpty(t, yaml)
}

func TestFormatYAML_Headers(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | responses | headers | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	_ = yaml
}

func TestFormatYAML_SecuritySchemes(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`components | where(kind == "security-scheme") | to-yaml`, g)
	require.NoError(t, err)

	yaml := oq.FormatYAML(result, g)
	assert.NotEmpty(t, yaml)
}

func TestExecuteWithSearchPaths_IncludeModule(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	dir := t.TempDir()
	err := os.WriteFile(dir+"/helpers.oq", []byte(`def big: where(depth > 0);`), 0644)
	require.NoError(t, err)

	query := fmt.Sprintf(`include "%s/helpers.oq"; schemas | where(isComponent) | big | select name`, dir)
	result, err := oq.ExecuteWithSearchPaths(query, g, []string{dir})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestExecuteWithSearchPaths_DefsOnly(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Query with only defs and no pipeline should return empty result
	result, err := oq.ExecuteWithSearchPaths(`def noop: take(1);`, g, nil)
	require.NoError(t, err)
	assert.Empty(t, result.Rows)
}

func TestProbeSchemaField_AllNullBranches(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Use where(has(...)) which evaluates fieldValue -> probeSchemaField.
	// Pet doesn't have these fields so probeSchemaField returns null — hitting the null branches.
	// The where filter will produce empty results since has() returns false for null.
	probeFields := []string{
		"xml", "externalDocs", "const", "multipleOf",
		"unevaluatedItems", "unevaluatedProperties", "anchor", "id",
		"schema", "defs", "examples",
	}
	for _, f := range probeFields {
		t.Run(f, func(t *testing.T) {
			t.Parallel()
			query := fmt.Sprintf(`schemas | where(isComponent) | where(has(%s))`, f)
			result, err := oq.Execute(query, g)
			require.NoError(t, err)
			// These fields don't exist on petstore schemas, so result should be empty
			assert.Empty(t, result.Rows, "Pet should not have %s", f)
		})
	}
}

func TestProbeSchemaField_NonNullBranches(t *testing.T) {
	t.Parallel()
	g := loadCyclicGraph(t)

	// Select probe fields on NodeA's inline schemas where these fields exist
	// This exercises the non-null branches in probeSchemaField
	tests := []struct {
		name  string
		query string
	}{
		{"select additionalProperties", `schemas | where(has(additionalProperties)) | select name, additionalProperties`},
		{"select patternProperties", `schemas | where(has(patternProperties)) | select name, patternProperties`},
		{"select not", `schemas | where(has(not)) | select name, not`},
		{"select if", `schemas | where(has(if)) | select name, if`},
		{"select then", `schemas | where(has(then)) | select name, then`},
		{"select else", `schemas | where(has(else)) | select name, else`},
		{"select contains", `schemas | where(has(contains)) | select name, contains`},
		{"select propertyNames", `schemas | where(has(propertyNames)) | select name, propertyNames`},
		{"select prefixItems", `schemas | where(has(prefixItems)) | select name, prefixItems`},
		{"select dependentSchemas", `schemas | where(has(dependentSchemas)) | select name, dependentSchemas`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(tt.query, g)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Rows)
		})
	}
}

func TestSchemaContentField_MoreFields(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Select all schema content fields on all schemas to hit branches
	query := `schemas | select name, description, title, format, pattern, nullable, readOnly, writeOnly, deprecated, uniqueItems, discriminatorProperty, discriminatorMappingCount, required, requiredCount, enum, enumCount, minimum, maximum, minLength, maxLength, minItems, maxItems, minProperties, maxProperties, extensionCount, contentEncoding, contentMediaType`
	result, err := oq.Execute(query, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestTraverseItems_Petstore(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Items on array schemas in petstore
	result, err := oq.Execute(`schemas | where(type == "array") | items | select name, via`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "array schemas should have items")
}

func TestExecContentTypes_FromRequestBody(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// content-types from request body exercises the RequestBodyResult branch
	result, err := oq.Execute(`operations | where(hasRequestBody) | request-body | content-types | select mediaType`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "request body should have content types")
}

func TestExecParameters_FromOperations(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Parameters from operations — exercise the operation-level parameter collection
	result, err := oq.Execute(`operations | where(parameterCount > 0) | parameters | select name, in, required`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestLoadModule_ParseError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Write a module with invalid syntax (missing semicolon for def)
	err := os.WriteFile(dir+"/bad.oq", []byte(`def broken`), 0644)
	require.NoError(t, err)

	_, err = oq.LoadModule(dir+"/bad.oq", nil)
	assert.Error(t, err)
}

// --- New capability tests ---

func TestExecute_WebhooksSource(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`webhooks`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	// All rows should be webhook operations
	for _, row := range result.Rows {
		assert.Equal(t, oq.OperationResult, row.Kind)
		v := oq.FieldValuePublic(row, "isWebhook", g)
		assert.True(t, v.Bool, "webhook source should only return webhook operations")
	}
}

func TestExecute_WebhookIsWebhookField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Non-webhook operations should have isWebhook=false
	result, err := oq.Execute(`operations | where(not isWebhook) | select name, isWebhook`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have non-webhook operations")

	// Webhook operations should have isWebhook=true
	result, err = oq.Execute(`operations | where(isWebhook) | select name, isWebhook`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "should have webhook operations")
}

func TestExecute_ServersSource(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`servers`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 2, "petstore has 2 servers")

	// Check fields
	row := result.Rows[0]
	url := oq.FieldValuePublic(row, "url", g)
	assert.Equal(t, "https://api.petstore.io/v1", url.Str)

	desc := oq.FieldValuePublic(row, "description", g)
	assert.Equal(t, "Production server", desc.Str)

	varCount := oq.FieldValuePublic(row, "variableCount", g)
	assert.Equal(t, 1, varCount.Int)
}

func TestExecute_ServersSelect(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`servers | select url, description`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 2)
	assert.Equal(t, []string{"url", "description"}, result.Fields)
}

func TestExecute_TagsSource(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`tags`, g)
	require.NoError(t, err)
	require.Len(t, result.Rows, 2, "petstore has 2 tags")

	// Check tag fields
	row := result.Rows[0]
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "pets", name.Str)

	desc := oq.FieldValuePublic(row, "description", g)
	assert.Equal(t, "Everything about your pets", desc.Str)

	summary := oq.FieldValuePublic(row, "summary", g)
	assert.Equal(t, "Pet operations", summary.Str)

	opCount := oq.FieldValuePublic(row, "operationCount", g)
	assert.Equal(t, 3, opCount.Int, "pets tag should have 3 operations (listPets, createPet, showPetById)")
}

func TestExecute_TagsSelect(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`tags | select name, operationCount`, g)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 2)
	assert.Equal(t, []string{"name", "operationCount"}, result.Fields)
}

func TestExecute_Callbacks(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(name == "createPet") | callbacks`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows, "createPet has callbacks")

	row := result.Rows[0]
	assert.Equal(t, oq.OperationResult, row.Kind)
	assert.Equal(t, "onPetCreated", row.CallbackName)
}

func TestExecute_CallbackCount(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(callbackCount > 0) | select name, callbackCount`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)
	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "createPet", name.Str)
}

func TestExecute_Links(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | where(name == "listPets") | responses | links`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows, "listPets 200 response has links")

	row := result.Rows[0]
	assert.Equal(t, oq.LinkResult, row.Kind)
	name := oq.FieldValuePublic(row, "name", g)
	assert.Equal(t, "GetPetById", name.Str)

	opId := oq.FieldValuePublic(row, "operationId", g)
	assert.Equal(t, "showPetById", opId.Str)

	desc := oq.FieldValuePublic(row, "description", g)
	assert.Contains(t, desc.Str, "specific pet")
}

func TestExecute_LinksSelect(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | responses | links | select name, operationId, description`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
	assert.Equal(t, []string{"name", "operationId", "description"}, result.Fields)
}

func TestExecute_AdditionalProperties(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(has(additionalProperties)) | additional-properties`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "Metadata has additionalProperties")

	row := result.Rows[0]
	via := oq.FieldValuePublic(row, "via", g)
	assert.Equal(t, "additionalProperties", via.Str)
}

func TestExecute_PatternProperties(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`schemas | where(has(patternProperties)) | pattern-properties`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows, "Metadata has patternProperties")

	row := result.Rows[0]
	via := oq.FieldValuePublic(row, "via", g)
	assert.Equal(t, "patternProperty", via.Str)
}

func TestExecute_SchemaDefaultField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Pet.status has default: available
	result, err := oq.Execute(`schemas | where(name == "Pet") | properties | where(key == "status") | select key, default`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)
	def := oq.FieldValuePublic(result.Rows[0], "default", g)
	assert.Equal(t, "available", def.Str)
}

func TestExecute_SchemaEnumField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Pet.status has enum values
	result, err := oq.Execute(`schemas | where(name == "Pet") | properties | where(key == "status") | select key, enum, enumCount`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)
	enumCount := oq.FieldValuePublic(result.Rows[0], "enumCount", g)
	assert.Equal(t, 3, enumCount.Int)
}

func TestExecute_OperationExtensionField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// listPets has x-speakeasy-name-override; use x_ prefix in expr to avoid dash parsing issues
	result, err := oq.Execute(`operations | where(has(x_speakeasy_name_override)) | select name, x_speakeasy_name_override`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)
	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "listPets", name.Str)
	// Field access via direct fieldValue works with the canonical x- name
	ext := oq.FieldValuePublic(result.Rows[0], "x-speakeasy-name-override", g)
	assert.Equal(t, "ListAllPets", ext.Str)
}

func TestExecute_SchemaExtensionField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Pet has x-speakeasy-entity; use x_ prefix in has() for parser compatibility
	result, err := oq.Execute(`schemas | where(has(x_speakeasy_entity)) | select name, x_speakeasy_entity`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)
	name := oq.FieldValuePublic(result.Rows[0], "name", g)
	assert.Equal(t, "Pet", name.Str)
	ext := oq.FieldValuePublic(result.Rows[0], "x-speakeasy-entity", g)
	assert.Equal(t, "Pet", ext.Str)
}

func TestExecute_ParseNewStages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"webhooks source", "webhooks"},
		{"servers source", "servers"},
		{"tags source", "tags"},
		{"callbacks", "operations | callbacks"},
		{"links", "operations | responses | links"},
		{"additional-properties", "schemas | additional-properties"},
		{"pattern-properties", "schemas | pattern-properties"},
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

func TestExecute_ServerKindName(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`servers | select kind, url`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)
	kind := oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "server", kind.Str)
}

func TestExecute_TagKindName(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`tags | select kind, name`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)
	kind := oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "tag", kind.Str)
}

func TestExecute_LinkKindName(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | responses | links | select kind, name`, g)
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)
	kind := oq.FieldValuePublic(result.Rows[0], "kind", g)
	assert.Equal(t, "link", kind.Str)
}

func TestExecute_LinkOperationBackNav(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// links should support back-navigation to operation
	result, err := oq.Execute(`operations | responses | links | operation | unique | select name`, g)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}

func TestExecute_ServerEmitYAML(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`servers | take(1) | to-yaml`, g)
	require.NoError(t, err)
	assert.True(t, result.EmitYAML)
	output := oq.FormatYAML(result, g)
	assert.Contains(t, output, "api.petstore.io")
}

func TestExecute_TagEmitYAML(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`tags | take(1) | to-yaml`, g)
	require.NoError(t, err)
	assert.True(t, result.EmitYAML)
	output := oq.FormatYAML(result, g)
	assert.Contains(t, output, "pets")
}

func TestExecute_LinkEmitYAML(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute(`operations | responses | links | to-yaml`, g)
	require.NoError(t, err)
	output := oq.FormatYAML(result, g)
	assert.Contains(t, output, "GetPetById")
}

func TestExecute_ExplainNewStages(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	tests := []struct {
		query    string
		contains string
	}{
		{"operations | callbacks | explain", "callbacks"},
		{"operations | responses | links | explain", "links"},
		{"schemas | additional-properties | explain", "additional properties"},
		{"schemas | pattern-properties | explain", "pattern properties"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(tt.query, g)
			require.NoError(t, err)
			assert.Contains(t, strings.ToLower(result.Explain), tt.contains)
		})
	}
}

func TestExecute_FieldsNewKinds(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// Verify 'fields' stage works for new kinds
	tests := []struct {
		query    string
		contains string
	}{
		{"servers | fields", "url"},
		{"tags | fields", "operationCount"},
		{"operations | responses | links | fields", "operationId"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			t.Parallel()
			result, err := oq.Execute(tt.query, g)
			require.NoError(t, err)
			assert.Contains(t, result.Explain, tt.contains)
		})
	}
}

func TestExecute_DefaultFieldsNewKinds(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// servers default fields
	result, err := oq.Execute(`servers`, g)
	require.NoError(t, err)
	output := oq.FormatTable(result, g)
	assert.Contains(t, output, "url")
	assert.Contains(t, output, "description")
	assert.Contains(t, output, "variableCount")

	// tags default fields
	result, err = oq.Execute(`tags`, g)
	require.NoError(t, err)
	output = oq.FormatTable(result, g)
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "description")
	assert.Contains(t, output, "operationCount")
}

func TestExecute_ComponentDefaultField(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	// LimitParam has default: 20 on its schema
	result, err := oq.Execute(`components | where(kind == "parameter") | to-schema | select name, default`, g)
	require.NoError(t, err)
	if len(result.Rows) > 0 {
		def := oq.FieldValuePublic(result.Rows[0], "default", g)
		assert.Equal(t, "20", def.Str)
	}
}
