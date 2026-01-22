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

func TestOperationSuccessResponseRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "operation with 2xx response",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "operation with 3xx response",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '302':
          description: redirect
`,
		},
		{
			name: "operation with mixed responses",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '204':
          description: no content
        '404':
          description: missing
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OperationSuccessResponseRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs, "should have no lint errors")
		})
	}
}

func TestOperationSuccessResponseRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yaml           string
		expectedErrors []string
	}{
		{
			name: "missing success response",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '404':
          description: missing
`,
			expectedErrors: []string{
				"[10:7] warning style-operation-success-response operation `listUsers` must define at least a single `2xx` or `3xx` response",
			},
		},
		{
			name: "missing responses",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
`,
			expectedErrors: []string{
				"[9:7] warning style-operation-success-response operation `listUsers` must define at least a single `2xx` or `3xx` response",
			},
		},
		{
			name: "missing success response without operationId",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '500':
          description: error
`,
			expectedErrors: []string{
				"[9:7] warning style-operation-success-response operation `undefined operation (no operationId)` must define at least a single `2xx` or `3xx` response",
			},
		},
		{
			name: "integer response code in OAS3",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        200:
          description: ok
`,
			expectedErrors: []string{
				"[10:7] warning style-operation-success-response operation `listUsers` uses an `integer` instead of a `string` for response code `200`",
			},
		},
		{
			name: "missing success response and integer response codes",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        404:
          description: missing
`,
			expectedErrors: []string{
				"[10:7] warning style-operation-success-response operation `listUsers` uses an `integer` instead of a `string` for response code `404`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OperationSuccessResponseRule{}
			config := &linter.RuleConfig{}

			// Build index for the rule
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			require.NotEmpty(t, errs, "should have lint errors")

			var errMsgs []string
			for _, err := range errs {
				errMsgs = append(errMsgs, err.Error())
			}

			assert.ElementsMatch(t, tt.expectedErrors, errMsgs)
		})
	}
}

func TestOperationSuccessResponseRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OperationSuccessResponseRule{}

	assert.Equal(t, "style-operation-success-response", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
