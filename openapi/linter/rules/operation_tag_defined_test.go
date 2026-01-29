package rules_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationTagDefinedRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "all operation tags are defined globally",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
  - name: products
paths:
  /users:
    get:
      tags:
        - users
      responses:
        '200':
          description: ok
  /products:
    get:
      tags:
        - products
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "operations without tags",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "no global tags but no operation tags",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "multiple tags all defined",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
  - name: admin
paths:
  /users:
    get:
      tags:
        - users
        - admin
      responses:
        '200':
          description: ok
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OperationTagDefinedRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
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

func TestOperationTagDefinedRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "operation tag not defined globally",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
paths:
  /products:
    get:
      tags:
        - products
      responses:
        '200':
          description: ok
`,
			expectedError: "[12:11] warning style-operation-tag-defined tag `products` for `GET` /products operation is not defined as a global tag",
		},
		{
			name: "one of multiple tags not defined",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: users
paths:
  /users:
    get:
      tags:
        - users
        - admin
      responses:
        '200':
          description: ok
`,
			expectedError: "[13:11] warning style-operation-tag-defined tag `admin` for `GET` /users operation is not defined as a global tag",
		},
		{
			name: "no global tags but operation has tag",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      tags:
        - users
      responses:
        '200':
          description: ok
`,
			expectedError: "[10:11] warning style-operation-tag-defined tag `users` for `GET` /users operation is not defined as a global tag",
		},
		{
			name: "operation with operationId uses id in error message",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      tags:
        - admin
      responses:
        '200':
          description: ok
`,
			expectedError: "[11:11] warning style-operation-tag-defined tag `admin` for listUsers operation is not defined as a global tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.OperationTagDefinedRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)

			require.Len(t, errs, 1)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestOperationTagDefinedRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OperationTagDefinedRule{}

	assert.Equal(t, "style-operation-tag-defined", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
