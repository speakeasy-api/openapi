package rules

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOASSchemaCheck_StringConstraints_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "valid minLength and maxLength",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      minLength: 5
      maxLength: 10
paths: {}
`,
		},
		{
			name: "valid pattern",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      pattern: ^[a-z]+$
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &OASSchemaCheckRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestOASSchemaCheck_StringConstraints_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "negative minLength",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      minLength: -1
paths: {}
`,
			expected: 1,
		},
		{
			name: "maxLength less than minLength",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      minLength: 10
      maxLength: 5
paths: {}
`,
			expected: 1,
		},
		{
			name: "invalid regex pattern",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      pattern: "[invalid("
paths: {}
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &OASSchemaCheckRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Len(t, errs, tt.expected)
		})
	}
}

func TestOASSchemaCheck_NumberConstraints_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "valid minimum and maximum",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: number
      minimum: 0
      maximum: 100
paths: {}
`,
		},
		{
			name: "valid multipleOf",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: integer
      multipleOf: 5
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &OASSchemaCheckRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestOASSchemaCheck_NumberConstraints_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "multipleOf zero",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: integer
      multipleOf: 0
paths: {}
`,
			expected: 1,
		},
		{
			name: "maximum less than minimum",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: number
      minimum: 100
      maximum: 0
paths: {}
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &OASSchemaCheckRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Len(t, errs, tt.expected)
		})
	}
}

func TestOASSchemaCheck_TypeMismatch_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "string type with number constraints",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: string
      minimum: 0
      maximum: 100
paths: {}
`,
			expected: 2, // minimum and maximum
		},
		{
			name: "number type with string constraints",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: number
      minLength: 5
      pattern: ^[a-z]+$
paths: {}
`,
			expected: 2, // minLength and pattern
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &OASSchemaCheckRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Len(t, errs, tt.expected)
		})
	}
}

func TestOASSchemaCheck_ObjectRequired_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "required without properties",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      required:
        - name
paths: {}
`,
			expected: 1,
		},
		{
			name: "required field not in properties",
			yaml: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      properties:
        age:
          type: integer
      required:
        - name
paths: {}
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &OASSchemaCheckRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Len(t, errs, tt.expected)
		})
	}
}
