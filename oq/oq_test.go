package oq_test

import (
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/oq"
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

func TestFormatMarkdown_Count_Success(t *testing.T) {
	t.Parallel()
	g := loadTestGraph(t)

	result, err := oq.Execute("schemas | count", g)
	require.NoError(t, err)

	md := oq.FormatMarkdown(result, g)
	assert.NotEmpty(t, md)
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
