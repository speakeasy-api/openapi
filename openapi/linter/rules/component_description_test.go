package rules_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentDescriptionRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "schema with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      description: A user object
      type: object
`,
		},
		{
			name: "parameter with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  parameters:
    userId:
      name: userId
      in: path
      description: The user identifier
      required: true
      schema:
        type: string
`,
		},
		{
			name: "requestBody with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  requestBodies:
    UserCreate:
      description: Request body for creating a user
      content:
        application/json:
          schema:
            type: object
`,
		},
		{
			name: "response with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  responses:
    NotFound:
      description: Resource not found
      content:
        application/json:
          schema:
            type: object
`,
		},
		{
			name: "example with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  examples:
    UserExample:
      description: Example user object
      value:
        name: John Doe
`,
		},
		{
			name: "header with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  headers:
    X-Rate-Limit:
      description: Rate limit information
      schema:
        type: integer
`,
		},
		{
			name: "link with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  links:
    UserByUserId:
      description: Link to user by ID
      operationId: getUser
`,
		},
		{
			name: "securityScheme with description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      description: Bearer token authentication
`,
		},
		{
			name: "no components",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
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

			rule := &rules.ComponentDescriptionRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs)
		})
	}
}

func TestComponentDescriptionRule_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "schema missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
`,
			expectedError: "[9:5] warning style-component-description `schemas` component `User` is missing a description",
		},
		{
			name: "parameter missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  parameters:
    userId:
      name: userId
      in: path
      required: true
      schema:
        type: string
`,
			expectedError: "[9:5] warning style-component-description `parameters` component `userId` is missing a description",
		},
		{
			name: "requestBody missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  requestBodies:
    UserCreate:
      content:
        application/json:
          schema:
            type: object
`,
			expectedError: "[9:5] warning style-component-description `requestBodies` component `UserCreate` is missing a description",
		},
		{
			name: "response missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  responses:
    NotFound:
      content:
        application/json:
          schema:
            type: object
`,
			expectedError: "[8:3] warning style-component-description `responses` component `NotFound` is missing a description",
		},
		{
			name: "example missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  examples:
    UserExample:
      value:
        name: John Doe
`,
			expectedError: "[9:5] warning style-component-description `examples` component `UserExample` is missing a description",
		},
		{
			name: "header missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  headers:
    X-Rate-Limit:
      schema:
        type: integer
`,
			expectedError: "[9:5] warning style-component-description `headers` component `X-Rate-Limit` is missing a description",
		},
		{
			name: "link missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  links:
    UserByUserId:
      operationId: getUser
`,
			expectedError: "[9:5] warning style-component-description `links` component `UserByUserId` is missing a description",
		},
		{
			name: "securityScheme missing description",
			yaml: `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
`,
			expectedError: "[9:5] warning style-component-description `securitySchemes` component `BearerAuth` is missing a description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err)

			rule := &rules.ComponentDescriptionRule{}
			config := &linter.RuleConfig{}
			docInfo := linter.NewDocumentInfo(doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			require.NotEmpty(t, errs)
			assert.Equal(t, tt.expectedError, errs[0].Error())
		})
	}
}

func TestComponentDescriptionRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.ComponentDescriptionRule{}

	assert.Equal(t, "style-component-description", rule.ID())
	assert.Equal(t, rules.CategoryStyle, rule.Category())
	assert.NotEmpty(t, rule.Description())
	assert.NotEmpty(t, rule.Link())
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity())
	assert.Nil(t, rule.Versions())
}
