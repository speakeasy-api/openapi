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

func TestOperationIDValidInURLRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "URL-friendly operationId",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: get-users
`,
		},
		{
			name: "operationId with underscores",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: get_users_list
`,
		},
		{
			name: "operationId with dots",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: users.list
`,
		},
		{
			name: "operationId with reserved characters",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: get@users
`,
		},
		{
			name: "no operationId",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get users
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OperationIDValidInURLRule{}
			config := &linter.RuleConfig{}

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

func TestOperationIDValidInURLRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "operationId with spaces",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: get users
`,
			expectedError: "[9:20] error semantic-operation-id-valid-in-url operationId `get users` contains characters that are not URL-friendly",
		},
		{
			name: "operationId with percent sign",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: get%users
`,
			expectedError: "[9:20] error semantic-operation-id-valid-in-url operationId `get%users` contains characters that are not URL-friendly",
		},
		{
			name: "operationId with angle brackets",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: <getUsers>
`,
			expectedError: "[9:20] error semantic-operation-id-valid-in-url operationId `<getUsers>` contains characters that are not URL-friendly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.OperationIDValidInURLRule{}
			config := &linter.RuleConfig{}

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})
			docInfo := linter.NewDocumentInfoWithIndex(doc, "test.yaml", idx)

			errs := rule.Run(ctx, docInfo, config)
			require.Len(t, errs, 1, "should have one lint error")
			assert.Equal(t, tt.expectedError, errs[0].Error(), "error message should match exactly")
		})
	}
}

func TestOperationIDValidInURLRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.OperationIDValidInURLRule{}

	assert.Equal(t, "semantic-operation-id-valid-in-url", rule.ID())
	assert.Equal(t, rules.CategorySemantic, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
