package linter_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPILinter_ExternalDiagnostic_NestedReferences(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          $ref: './response.yaml'
  /teams:
    get:
      responses:
        '200':
          $ref: './response.yaml'
`))
	require.NoError(t, err, "main document should parse")
	fs := newMockVirtualFS()
	fs.addFile("/spec/response.yaml", `description: ok
content:
  application/json:
    schema:
      $ref: './schemas/alias.yaml'
`)
	fs.addFile("/spec/schemas/alias.yaml", "$ref: './types.yaml#/Entry'\n")
	fs.addFile("/spec/schemas/types.yaml", "Entry:\n  type: integer\n  enum:\n    - bad\n")
	lntr, err := openapiLinter.NewLinter(&linter.Config{Rules: []linter.RuleEntry{{ID: rules.RuleSemanticTypedEnum}}})
	require.NoError(t, err, "linter should initialise")
	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/main.yaml"), nil, &linter.LintOptions{
		ResolveOptions: &references.ResolveOptions{VirtualFS: fs},
	})
	require.NoError(t, err, "linting should succeed")
	require.Len(t, output.Results, 1, "a shared external enum defect should be reported once")
	var finding *validation.Error
	require.ErrorAs(t, output.Results[0], &finding, "finding should retain its metadata")
	assert.Equal(t, "/spec/schemas/types.yaml", finding.DocumentLocation, "the final target, not the alias or referring response, owns the enum")
	assert.Equal(t, 4, finding.GetLineNumber(), "the original external line should be retained")
	assert.Equal(t, 7, finding.GetColumnNumber(), "the original external column should be retained")
	assert.Contains(t, output.FormatText(), "/spec/schemas/types.yaml", "text output should identify the source file")
	assert.Contains(t, output.FormatJSON(), `"document": "/spec/schemas/types.yaml"`, "JSON output should expose the source file")
}

func TestOpenAPILinter_ExternalDiagnostic_DistinctDocuments(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Left:
      $ref: './left.yaml#/Entry'
    Right:
      $ref: './right.yaml#/Entry'
`))
	require.NoError(t, err, "main document should parse")
	fs := newMockVirtualFS()
	fs.addFile("/spec/left.yaml", "Entry:\n  type: integer\n  enum:\n    - bad\n")
	fs.addFile("/spec/right.yaml", "Entry:\n  type: integer\n  enum:\n    - bad\n")
	lntr, err := openapiLinter.NewLinter(&linter.Config{Rules: []linter.RuleEntry{{ID: rules.RuleSemanticTypedEnum}}})
	require.NoError(t, err, "linter should initialise")
	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/main.yaml"), nil, &linter.LintOptions{
		ResolveOptions: &references.ResolveOptions{VirtualFS: fs},
	})
	require.NoError(t, err, "linting should succeed")
	require.Len(t, output.Results, 2, "identical defects in different files must not collapse")
	locations := make([]string, 0, len(output.Results))
	for _, result := range output.Results {
		var finding *validation.Error
		require.ErrorAs(t, result, &finding, "finding should retain its metadata")
		locations = append(locations, finding.DocumentLocation)
		assert.Equal(t, 4, finding.GetLineNumber(), "same external coordinates should be preserved independently")
	}
	assert.Equal(t, []string{"/spec/left.yaml", "/spec/right.yaml"}, locations, "findings should be sorted by actual source document")
}

func TestOpenAPILinter_ExternalDiagnostic_CopiesMetadataAndDisablesFix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		explicit string
		expected string
	}{
		{name: "infer cached source", expected: "/spec/schema.yaml"},
		{name: "preserve explicit source", explicit: "https://custom.example.com/schema.yaml", expected: "https://custom.example.com/schema.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    External:
      $ref: './schema.yaml'
`))
			require.NoError(t, err, "main document should parse")
			fs := newMockVirtualFS()
			fs.addFile("/spec/schema.yaml", "type: string\nnullable: true\n")
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument: doc, TargetDocument: doc, TargetLocation: "/spec/main.yaml", VirtualFS: fs,
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "/spec/main.yaml", idx)
			originalErrors := (&rules.OAS3NoNullableRule{}).Run(ctx, docInfo, nil)
			require.Len(t, originalErrors, 1, "the external nullable keyword should produce one original finding")
			var original *validation.Error
			require.ErrorAs(t, originalErrors[0], &original, "finding should retain its metadata")
			original.DocumentLocation = tt.explicit
			require.NotNil(t, original.Fix, "the original rule should supply its normal node fix")

			lntr, err := openapiLinter.NewLinter(&linter.Config{})
			require.NoError(t, err, "linter without enabled rules should initialise")
			output, err := lntr.Lint(ctx, docInfo, append(originalErrors, originalErrors...), nil)
			require.NoError(t, err, "linting should succeed with a supplied index")
			require.Len(t, output.Results, 1, "replayed errors for the same external defect should collapse without mutating the originals")
			var attributed *validation.Error
			require.ErrorAs(t, output.Results[0], &attributed, "finding should retain its metadata")
			assert.Equal(t, tt.expected, attributed.DocumentLocation, "explicit locations must take precedence over inferred locations")
			assert.Nil(t, attributed.Fix, "the single-document fix engine must not offer an external edit")
			assert.NotSame(t, original, attributed, "attribution must not mutate a caller-owned or cached error")
			assert.Equal(t, tt.explicit, original.DocumentLocation, "the original source metadata should be unchanged")
			assert.NotNil(t, original.Fix, "the original fix must remain unchanged")
			assert.Same(t, original.Node, attributed.Node, "coordinates must still refer to the original external node")
		})
	}
}

func TestOpenAPILinter_ReferenceLayout_MissingOperationID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		paths    string
		location string
	}{
		{
			name: "inline operation",
			paths: `  /users:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "external operation",
			paths: `  /users:
    $ref: './path.yaml'
`,
			location: "/spec/path.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, validationErrors, err := openapi.Unmarshal(ctx, strings.NewReader("openapi: 3.1.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths:\n"+tt.paths))
			require.NoError(t, err, "main document should parse")
			fs := newMockVirtualFS()
			fs.addFile("/spec/path.yaml", "get:\n  responses:\n    '200':\n      description: ok\n")
			lntr, err := openapiLinter.NewLinter(&linter.Config{Rules: []linter.RuleEntry{{ID: rules.RuleSemanticOperationOperationId}}})
			require.NoError(t, err, "linter should initialise")
			output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/main.yaml"), validationErrors, &linter.LintOptions{
				ResolveOptions: &references.ResolveOptions{VirtualFS: fs},
			})
			require.NoError(t, err, "linting should succeed")
			require.Len(t, output.Results, 1, "both layouts must report the missing operationId")
			var finding *validation.Error
			require.ErrorAs(t, output.Results[0], &finding, "finding should retain its metadata")
			assert.Equal(t, rules.RuleSemanticOperationOperationId, finding.Rule, "the same rule should run in each layout")
			assert.Equal(t, "the `GET` operation does not contain an `operationId`", finding.UnderlyingError.Error(), "logical diagnostic should be layout-independent")
			assert.Equal(t, tt.location, finding.DocumentLocation, "coordinates should belong to the operation's actual source")
		})
	}
}

func TestOpenAPILinter_ReferenceLayout_DuplicateOperationIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		paths      string
		locations  []string
		duplicates []string
	}{
		{
			name: "inline then external",
			paths: `  /first:
    get:
      operationId: shared
      responses:
        '200':
          description: ok
  /second:
    $ref: './path.yaml'
`,
			locations:  []string{"/spec/path.yaml"},
			duplicates: []string{"/second"},
		},
		{
			name: "external then inline and inline",
			paths: `  /first:
    $ref: './path.yaml'
  /second:
    get:
      operationId: shared
      responses:
        '200':
          description: ok
  /third:
    get:
      operationId: shared
      responses:
        '200':
          description: ok
`,
			locations:  []string{"", ""},
			duplicates: []string{"/second", "/third"},
		},
		{
			name: "three mounts of the same source operation",
			paths: `  /first:
    $ref: './path.yaml'
  /second:
    $ref: './path.yaml'
  /third:
    $ref: './path.yaml'
`,
			locations:  []string{"/spec/path.yaml", "/spec/path.yaml"},
			duplicates: []string{"/second", "/third"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, validationErrors, err := openapi.Unmarshal(ctx, strings.NewReader("openapi: 3.1.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths:\n"+tt.paths))
			require.NoError(t, err, "main document should parse")
			fs := newMockVirtualFS()
			fs.addFile("/spec/path.yaml", "get:\n  operationId: shared\n  responses:\n    '200':\n      description: ok\n")
			lntr, err := openapiLinter.NewLinter(&linter.Config{})
			require.NoError(t, err, "linter without style rules should initialise")
			output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/main.yaml"), validationErrors, &linter.LintOptions{
				ResolveOptions: &references.ResolveOptions{VirtualFS: fs},
			})
			require.NoError(t, err, "linting should succeed")
			require.Len(t, output.Results, len(tt.duplicates), "each additional operation usage should fail uniqueness exactly once")
			for i, result := range output.Results {
				var finding *validation.Error
				require.ErrorAs(t, result, &finding, "finding should retain its metadata")
				assert.Equal(t, validation.RuleValidationOperationIdUnique, finding.Rule, "duplicate IDs are structural validation findings")
				assert.Equal(t, tt.locations[i], finding.DocumentLocation, "duplicate should point to its source declaration")
				assert.Contains(t, finding.UnderlyingError.Error(), "path `"+tt.duplicates[i]+"`", "distinct mounted usages must not collapse by source pointer")
			}
		})
	}
}

func TestOpenAPILinter_ReferenceLayout_SharedDescriptionDefect(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /first:
    $ref: './path.yaml'
  /second:
    $ref: './path.yaml'
`))
	require.NoError(t, err, "main document should parse")
	fs := newMockVirtualFS()
	fs.addFile("/spec/path.yaml", `summary: same
description: same
get:
  summary: read
  description: Read an item
  responses:
    '200':
      description: ok
`)
	lntr, err := openapiLinter.NewLinter(&linter.Config{Rules: []linter.RuleEntry{{ID: rules.RuleStyleDescriptionDuplication}}})
	require.NoError(t, err, "linter should initialise")
	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/main.yaml"), nil, &linter.LintOptions{
		ResolveOptions: &references.ResolveOptions{VirtualFS: fs},
	})
	require.NoError(t, err, "linting should succeed")
	require.Len(t, output.Results, 1, "one source-only description defect must not multiply with target traversal or mounts")
	var finding *validation.Error
	require.ErrorAs(t, output.Results[0], &finding, "finding should retain its metadata")
	assert.Equal(t, "/spec/path.yaml", finding.DocumentLocation, "the defect belongs to the shared path item source")
}

func TestOpenAPILinter_ExternalDiagnostic_ReferenceAnnotationsBelongToSite(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          $ref: './response.yaml'
          summary: same
          description: same
`))
	require.NoError(t, err, "main document should parse")
	fs := newMockVirtualFS()
	fs.addFile("/spec/response.yaml", "description: target response\n")
	lntr, err := openapiLinter.NewLinter(&linter.Config{Rules: []linter.RuleEntry{{ID: rules.RuleStyleDescriptionDuplication}}})
	require.NoError(t, err, "linter should initialise")
	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/main.yaml"), nil, &linter.LintOptions{
		ResolveOptions: &references.ResolveOptions{VirtualFS: fs},
	})
	require.NoError(t, err, "linting should succeed")
	require.Len(t, output.Results, 1, "reference-site summary/description annotations should be checked independently of the target")
	var finding *validation.Error
	require.ErrorAs(t, output.Results[0], &finding, "finding should retain its metadata")
	assert.Empty(t, finding.DocumentLocation, "annotations belong to the main reference site, not the external target")
	assert.Equal(t, 11, finding.GetLineNumber(), "the summary value in the main document owns this defect")
}

func TestOpenAPILinter_ReferenceLayout_SuppliedIndexDiagnostics(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /first:
    $ref: './path.yaml'
  /second:
    $ref: './path.yaml'
`))
	require.NoError(t, err, "main document should parse")
	fs := newMockVirtualFS()
	fs.addFile("/spec/path.yaml", "get:\n  operationId: shared\n  responses:\n    '200':\n      description: ok\n")
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument: doc, TargetDocument: doc, TargetLocation: "/spec/main.yaml", VirtualFS: fs,
	})
	require.Len(t, idx.GetValidationErrors(), 1, "index should detect the mounted ID collision")
	lntr, err := openapiLinter.NewLinter(&linter.Config{})
	require.NoError(t, err, "linter should initialise")
	docInfo := linter.NewDocumentInfoWithIndex(doc, "/spec/main.yaml", idx)
	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err, "linting with a supplied index should succeed")
	assert.Len(t, output.Results, 1, "supplied-index validation errors must not be omitted")
	repeated, err := lntr.Lint(ctx, docInfo, idx.GetValidationErrors(), nil)
	require.NoError(t, err, "linting with already-forwarded errors should succeed")
	assert.Len(t, repeated.Results, 1, "already-forwarded index errors should not be appended again")
}

func TestOpenAPILinter_ExternalDiagnostic_MainDocumentFixPreserved(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Inline:
      type: string
      nullable: true
`))
	require.NoError(t, err, "main document should parse")
	lntr, err := openapiLinter.NewLinter(&linter.Config{Rules: []linter.RuleEntry{{ID: rules.RuleOAS3NoNullable}}})
	require.NoError(t, err, "linter should initialise")
	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/main.yaml"), nil, nil)
	require.NoError(t, err, "linting should succeed")
	require.Len(t, output.Results, 1, "the inline nullable keyword should produce a finding")
	var finding *validation.Error
	require.ErrorAs(t, output.Results[0], &finding, "finding should retain its metadata")
	assert.Empty(t, finding.DocumentLocation, "main-document attribution should stay unchanged")
	assert.NotNil(t, finding.Fix, "safe main-document node fixes should remain available")
}
