package oq_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/oq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_InvalidContracts_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		query   string
		message string
	}{
		{`include "helpers"`, "include missing terminating ;"},
		{`include ""; schemas`, "include requires a path"},
		{`def : properties; schemas`, "def requires a name"},
		{`def ($x): properties; schemas`, "def requires a name"},
		{`def f($x: properties; schemas`, "def params missing closing )"},
		{`def f(x): properties; schemas`, `def param "x" must start with $`},
		{`schemas | where name`, "where requires parentheses: where(expr)"},
		{`schemas | where (name)`, "where requires parentheses: where(expr)"},
		{`schemas | where(name`, `unknown stage: "where(name"`},
		{`schemas | where(name) extra`, `unknown stage: "where(name)"`},
		{`schemas | select(name)`, "select is for projection, not filtering — use where(expr) to filter"},
		{`schemas | select`, "select requires field names"},
		{`schemas | sort-by()`, "sort-by requires a field name"},
		{`schemas | group-by()`, "group-by requires a field name"},
		{`schemas | group-by(, name)`, "group-by requires a field name"},
		{`schemas | last(nope)`, `last requires a number: strconv.Atoi: parsing "nope": invalid syntax`},
		{`schemas | path(Pet,)`, "path requires two schema names"},
		{`schemas | highest(3)`, "highest requires a number and a field name"},
		{`schemas | highest(nope, depth)`, `highest requires a number: strconv.Atoi: parsing "nope": invalid syntax`},
		{`schemas | highest 3`, "highest requires a number and a field name"},
		{`schemas | highest nope depth`, `highest requires a number: strconv.Atoi: parsing "nope": invalid syntax`},
		{`schemas | lowest(3)`, "lowest requires a number and a field name"},
		{`schemas | lowest(nope, depth)`, `lowest requires a number: strconv.Atoi: parsing "nope": invalid syntax`},
		{`schemas | lowest 3`, "lowest requires a number and a field name"},
		{`schemas | lowest nope depth`, `lowest requires a number: strconv.Atoi: parsing "nope": invalid syntax`},
		{`schemas | format xml`, `format must be table, json, markdown, toon, or gcf, got "xml"`},
		{`schemas | shared-refs(nope)`, `shared-refs requires a minimum count: strconv.Atoi: parsing "nope": invalid syntax`},
		{`schemas | properties(nope)`, `properties requires a depth number or *: strconv.Atoi: parsing "nope": invalid syntax`},
		{`schemas | refs(sideways, 2)`, `refs first argument must be out or in, got "sideways"`},
		{`schemas | refs(out, nope)`, `refs requires a depth number or *: strconv.Atoi: parsing "nope": invalid syntax`},
		{`schemas | let $ = name`, "let variable must start with $"},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			t.Parallel()
			_, err := oq.ParseQuery(tt.query)
			require.EqualError(t, err, tt.message, "invalid query should report the specific contract")
		})
	}
}

func TestParse_ExactStages_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		query  string
		stages []oq.Stage
	}{
		{`path(Pet, Owner) | last(2)`, []oq.Stage{{Kind: oq.StagePath, PathFrom: "Pet", PathTo: "Owner"}, {Kind: oq.StageLast, Limit: 2}}},
		{`schemas || refs() | refs(2) | refs(in, 3)`, []oq.Stage{{Kind: oq.StageSource, Source: "schemas"}, {Kind: oq.StageRefs, Limit: 1}, {Kind: oq.StageRefs, Limit: 2}, {Kind: oq.StageRefs, RefsDir: "in", Limit: 3}}},
		{`schemas | path "Pet Store" "Owner"`, []oq.Stage{{Kind: oq.StageSource, Source: "schemas"}, {Kind: oq.StagePath, PathFrom: "Pet Store", PathTo: "Owner"}}},
		{`schemas | properties(2) | select name, , depth`, []oq.Stage{{Kind: oq.StageSource, Source: "schemas"}, {Kind: oq.StageProperties, Limit: 2}, {Kind: oq.StageSelect, Fields: []string{"name", "depth"}}}},
		{`schemas | where(contains(name, 'a\'|b')) | select name`, []oq.Stage{{Kind: oq.StageSource, Source: "schemas"}, {Kind: oq.StageWhere, Expr: `contains(name, 'a\'|b')`}, {Kind: oq.StageSelect, Fields: []string{"name"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			t.Parallel()
			stages, err := oq.Parse(tt.query)
			require.NoError(t, err, "valid query should parse")
			assert.Equal(t, tt.stages, stages, "stage arguments and quoted delimiters should survive parsing")
		})
	}
}

func TestParseQuery_DeclarationsOnly_Success(t *testing.T) {
	t.Parallel()
	q, err := oq.ParseQuery(`include "semi;colon"; def pick($x): where(name == "escaped\";pipe|value");`)
	require.NoError(t, err, "quoted delimiters should not terminate declarations")
	assert.Equal(t, &oq.Query{Includes: []string{"semi;colon"}, Defs: []oq.FuncDef{{Name: "pick", Params: []string{"$x"}, Body: `where(name == "escaped\";pipe|value")`}}}, q, "declarations-only queries should preserve their definitions")
}
