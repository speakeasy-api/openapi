package oq

import (
	"testing"

	"github.com/speakeasy-api/openapi/oq/expr"
	"github.com/stretchr/testify/assert"
)

func TestToonValue_ArrayEscapesSemicolonElements(t *testing.T) {
	t.Parallel()

	value := expr.ArrayVal([]string{"v1;deprecated", "v2;current"})

	encoded := toonValue(value)

	assert.Equal(t, `"v1;deprecated";"v2;current"`, encoded, "array elements containing the delimiter should be quoted individually")
}

// TestFormatGCF_ArrayValuedProjection pins the GCF output for a projection that
// includes an array-valued field. FormatGCF passes array fields to the encoder as
// []string, which are emitted as a GCF array attachment (a "^" marker on the row
// plus a ".scopes [N]: ..." continuation line) rather than a single scalar cell.
// This golden covers the array path that the default petstore smoke query does not
// exercise (its operation fields contain no arrays).
func TestFormatGCF_ArrayValuedProjection(t *testing.T) {
	t.Parallel()

	// SecurityRequirement rows read schemeName/scopes directly off the Row, so this
	// drives FormatGCF end to end without needing a populated SchemaGraph.
	result := &Result{
		Rows: []Row{{
			Kind:       SecurityRequirementResult,
			SchemeName: "petstore_auth",
			Scopes:     []string{"read:pets", "write:pets"},
		}},
		Fields: []string{"schemeName", "scopes"},
	}

	got := FormatGCF(result, nil)

	want := "GCF profile=generic\n" +
		"## [1]{schemeName,scopes}\n" +
		"@0 petstore_auth|^\n" +
		".scopes [2]: read:pets,write:pets\n"
	assert.Equal(t, want, got, "array-valued fields should emit as a GCF array attachment")
}

// TestFormatGCF_MarkerShapedStringQuoted pins the GCF output for a string value
// shaped like a GCF marker ("^{...}"). Such values are quoted rather than emitted
// bare so they round-trip as literal strings instead of being read as markup.
func TestFormatGCF_MarkerShapedStringQuoted(t *testing.T) {
	t.Parallel()

	result := &Result{
		Rows: []Row{{
			Kind:       SecurityRequirementResult,
			SchemeName: "^{oauth}",
			Scopes:     []string{"read"},
		}},
		Fields: []string{"schemeName", "scopes"},
	}

	got := FormatGCF(result, nil)

	want := "GCF profile=generic\n" +
		"## [1]{schemeName,scopes}\n" +
		"@0 \"^{oauth}\"|^\n" +
		".scopes [1]: read\n"
	assert.Equal(t, want, got, "marker-shaped string values should be quoted")
}
