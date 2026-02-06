package rules_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func createDocInfoWithIndexUnusedComponents(t *testing.T, ctx context.Context, doc *openapi.OpenAPI, location string) *linter.DocumentInfo[*openapi.OpenAPI] {
	t.Helper()
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: location,
	})
	return linter.NewDocumentInfoWithIndex(doc, location, idx)
}

func TestUnusedComponentRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "all components referenced",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      security:
        - ApiKey: []
      parameters:
        - $ref: '#/components/parameters/PetId'
      responses:
        '200':
          $ref: '#/components/responses/PetResponse'
components:
  schemas:
    Pet:
      type: string
  parameters:
    PetId:
      name: petId
      in: query
      schema:
        type: string
  responses:
    PetResponse:
      description: ok
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Pet'
  securitySchemes:
    ApiKey:
      type: apiKey
      in: header
      name: X-API-Key
`,
		},
		{
			name: "no components",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
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
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.UnusedComponentRule{}
			config := &linter.RuleConfig{}

			docInfo := createDocInfoWithIndexUnusedComponents(t, ctx, doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)
			assert.Empty(t, errs, "should have no lint errors")
		})
	}
}

func TestUnusedComponentRule_Violations(t *testing.T) {
	t.Parallel()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: string
    Orphan:
      type: string
  responses:
    UnusedResponse:
      description: not used
  securitySchemes:
    ApiKey:
      type: apiKey
      in: header
      name: X-API-Key
security:
  - ApiKey: []
`

	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err, "unmarshal should succeed")

	rule := &rules.UnusedComponentRule{}
	config := &linter.RuleConfig{}

	docInfo := createDocInfoWithIndexUnusedComponents(t, ctx, doc, "test.yaml")

	errs := rule.Run(ctx, docInfo, config)

	expectedErrors := []string{
		"[20:5] warning semantic-unused-component `#/components/schemas/Orphan` is potentially unused or has been orphaned",
		"[23:5] warning semantic-unused-component `#/components/responses/UnusedResponse` is potentially unused or has been orphaned",
	}

	var errMsgs []string
	for _, lintErr := range errs {
		errMsgs = append(errMsgs, lintErr.Error())
	}

	assert.ElementsMatch(t, expectedErrors, errMsgs)
}

func TestUnusedComponentRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.UnusedComponentRule{}

	assert.Equal(t, "semantic-unused-component", rule.ID(), "rule ID should match")
	assert.Equal(t, rules.CategorySemantic, rule.Category(), "rule category should match")
	assert.NotEmpty(t, rule.Description(), "rule should have description")
	assert.NotEmpty(t, rule.Link(), "rule should have documentation link")
	assert.Equal(t, validation.SeverityWarning, rule.DefaultSeverity(), "default severity should be warning")
	assert.Nil(t, rule.Versions(), "versions should be nil (all versions)")
}

func TestUnusedComponentRule_UsageMarkingExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yaml           string
		expectedErrors []string
	}{
		{
			name: "x-speakeasy-include marks schema as used",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
components:
  schemas:
    MarkedUsed:
      type: string
      x-speakeasy-include: true
    ActuallyUnused:
      type: string
`,
			expectedErrors: []string{
				"[17:5] warning semantic-unused-component `#/components/schemas/ActuallyUnused` is potentially unused or has been orphaned",
			},
		},
		{
			name: "x-include marks schema as used",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
components:
  schemas:
    MarkedUsed:
      type: string
      x-include: true
    ActuallyUnused:
      type: string
`,
			expectedErrors: []string{
				"[17:5] warning semantic-unused-component `#/components/schemas/ActuallyUnused` is potentially unused or has been orphaned",
			},
		},
		{
			name: "x-used marks schema as used",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
components:
  schemas:
    MarkedUsed:
      type: string
      x-used: true
    ActuallyUnused:
      type: string
`,
			expectedErrors: []string{
				"[17:5] warning semantic-unused-component `#/components/schemas/ActuallyUnused` is potentially unused or has been orphaned",
			},
		},
		{
			name: "extension set to false does not mark as used",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
components:
  schemas:
    NotMarked:
      type: string
      x-speakeasy-include: false
`,
			expectedErrors: []string{
				"[14:5] warning semantic-unused-component `#/components/schemas/NotMarked` is potentially unused or has been orphaned",
			},
		},
		{
			name: "x-speakeasy-include marks parameter as used",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
components:
  parameters:
    MarkedUsed:
      name: test
      in: query
      x-speakeasy-include: true
    ActuallyUnused:
      name: unused
      in: query
`,
			expectedErrors: []string{
				"[18:5] warning semantic-unused-component `#/components/parameters/ActuallyUnused` is potentially unused or has been orphaned",
			},
		},
		{
			name: "x-include marks response as used",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
components:
  responses:
    MarkedUsed:
      description: marked
      x-include: true
    ActuallyUnused:
      description: unused
`,
			expectedErrors: []string{
				"[17:5] warning semantic-unused-component `#/components/responses/ActuallyUnused` is potentially unused or has been orphaned",
			},
		},
		{
			name: "x-used marks security scheme as used",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
components:
  securitySchemes:
    MarkedUsed:
      type: apiKey
      in: header
      name: X-API-Key
      x-used: true
    ActuallyUnused:
      type: apiKey
      in: header
      name: X-Unused
`,
			expectedErrors: []string{
				"[19:5] warning semantic-unused-component `#/components/securitySchemes/ActuallyUnused` is potentially unused or has been orphaned",
			},
		},
		{
			name: "all components with extensions are not flagged",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
components:
  schemas:
    Schema1:
      type: string
      x-speakeasy-include: true
    Schema2:
      type: string
      x-include: true
    Schema3:
      type: string
      x-used: true
`,
			expectedErrors: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.UnusedComponentRule{}
			config := &linter.RuleConfig{}

			docInfo := createDocInfoWithIndexUnusedComponents(t, ctx, doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			var errMsgs []string
			for _, lintErr := range errs {
				errMsgs = append(errMsgs, lintErr.Error())
			}

			assert.ElementsMatch(t, tt.expectedErrors, errMsgs, "should match expected errors")
		})
	}
}

func TestUnusedComponentRule_ExternalReferenceChainMarksUsed(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	mainYaml := `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '%s/external.yaml#/components/schemas/ExternalSchema'
components:
  schemas:
    SharedUsed:
      type: string
    SharedUnused:
      type: string`

	externalYaml := `
openapi: 3.1.0
info:
  title: External
  version: 1.0.0
paths:
  /external:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ExternalUnused'
components:
  schemas:
    ExternalSchema:
      type: object
      properties:
        shared:
          $ref: '%s/main.yaml#/components/schemas/SharedUsed'
    ExternalUnused:
      type: object
      properties:
        unused:
          $ref: '%s/main.yaml#/components/schemas/SharedUnused'
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/external.yaml":
			w.Header().Set("Content-Type", "application/yaml")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, externalYaml, server.URL, server.URL)
		case "/main.yaml":
			w.Header().Set("Content-Type", "application/yaml")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, mainYaml, server.URL)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(fmt.Sprintf(mainYaml, server.URL)))
	require.NoError(t, err, "unmarshal should succeed")

	rule := &rules.UnusedComponentRule{}
	config := &linter.RuleConfig{}

	docInfo := createDocInfoWithIndexUnusedComponents(t, ctx, doc, server.URL+"/main.yaml")

	errs := rule.Run(ctx, docInfo, config)

	require.Len(t, errs, 1, "should only flag unreferenced components in main doc")
	assert.Contains(t, errs[0].Error(), "`#/components/schemas/SharedUnused`", "should flag SharedUnused as unused")
}
