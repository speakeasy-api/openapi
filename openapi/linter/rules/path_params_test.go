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

// Helper function to create DocumentInfo with Index
func createDocInfoWithIndex(t *testing.T, ctx context.Context, doc *openapi.OpenAPI, location string) *linter.DocumentInfo[*openapi.OpenAPI] {
	t.Helper()
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: location,
	})
	return linter.NewDocumentInfoWithIndex(doc, location, idx)
}

func TestPathParamsRule_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "single path param in operation",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "path param defined at path item level",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        required: true
        schema:
          type: string
    get:
      responses:
        '200':
          description: ok
    post:
      responses:
        '201':
          description: created
`,
		},
		{
			name: "multiple path params",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}/posts/{postId}:
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
        - name: postId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "deeply nested path params",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /orgs/{orgId}/teams/{teamId}/members/{memberId}:
    get:
      parameters:
        - name: orgId
          in: path
          required: true
          schema:
            type: string
        - name: teamId
          in: path
          required: true
          schema:
            type: string
        - name: memberId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "path param override at operation level",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        required: true
        schema:
          type: string
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "mixed path item and operation params",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}/posts/{postId}:
    parameters:
      - name: userId
        in: path
        required: true
        schema:
          type: string
    get:
      parameters:
        - name: postId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "multiple operations sharing path item params",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        required: true
        schema:
          type: string
    get:
      responses:
        '200':
          description: ok
    put:
      responses:
        '200':
          description: ok
    delete:
      responses:
        '204':
          description: deleted
`,
		},
		{
			name: "path without template params",
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
			name: "query param ignored for path validation",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: filter
          in: query
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "header param ignored for path validation",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: X-Request-Id
          in: header
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
		},
		{
			name: "referenced parameter from components",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      parameters:
        - $ref: '#/components/parameters/UserId'
      responses:
        '200':
          description: ok
components:
  parameters:
    UserId:
      name: userId
      in: path
      required: true
      schema:
        type: string
`,
		},
		{
			name: "referenced path item from components",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    $ref: '#/components/pathItems/UserPath'
components:
  pathItems:
    UserPath:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      get:
        responses:
          '200':
            description: ok
`,
		},
		{
			name: "path item ref with params defined in operations",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    $ref: '#/components/pathItems/UserPath'
components:
  pathItems:
    UserPath:
      get:
        parameters:
          - name: userId
            in: path
            required: true
            schema:
              type: string
        responses:
          '200':
            description: ok
`,
		},
		{
			name: "mixed inline and referenced parameters",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}/posts/{postId}:
    get:
      parameters:
        - $ref: '#/components/parameters/UserId'
        - name: postId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
components:
  parameters:
    UserId:
      name: userId
      in: path
      required: true
      schema:
        type: string
`,
		},
		{
			name: "path item level ref param inherited by multiple ops",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    parameters:
      - $ref: '#/components/parameters/UserId'
    get:
      responses:
        '200':
          description: ok
    delete:
      responses:
        '204':
          description: deleted
components:
  parameters:
    UserId:
      name: userId
      in: path
      required: true
      schema:
        type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.PathParamsRule{}
			config := &linter.RuleConfig{}

			docInfo := createDocInfoWithIndex(t, ctx, doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			assert.Empty(t, errs, "should have no lint errors")
		})
	}
}

func TestPathParamsRule_MissingPathParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yaml           string
		expectedErrors []string
	}{
		{
			name: "missing single path param",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
`,
			expectedErrors: []string{
				"[9:7] error semantic-path-params path parameter `{userId}` is not defined in operation parameters",
			},
		},
		{
			name: "missing one of multiple path params",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}/posts/{postId}:
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
			expectedErrors: []string{
				"[9:7] error semantic-path-params path parameter `{postId}` is not defined in operation parameters",
			},
		},
		{
			name: "missing all path params",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}/posts/{postId}:
    get:
      responses:
        '200':
          description: ok
`,
			expectedErrors: []string{
				"[9:7] error semantic-path-params path parameter `{userId}` is not defined in operation parameters",
				"[9:7] error semantic-path-params path parameter `{postId}` is not defined in operation parameters",
			},
		},
		{
			name: "missing path param in one operation but not another",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
    post:
      responses:
        '201':
          description: created
`,
			expectedErrors: []string{
				"[19:7] error semantic-path-params path parameter `{userId}` is not defined in operation parameters",
			},
		},
		{
			name: "case sensitive param names",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      parameters:
        - name: userid
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
			expectedErrors: []string{
				"[9:7] error semantic-path-params path parameter `{userId}` is not defined in operation parameters",
				"[9:7] error semantic-path-params parameter `userid` is declared as path parameter but not used in path template `/users/{userId}`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.PathParamsRule{}
			config := &linter.RuleConfig{}

			docInfo := createDocInfoWithIndex(t, ctx, doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			var errMsgs []string
			for _, err := range errs {
				errMsgs = append(errMsgs, err.Error())
			}

			assert.ElementsMatch(t, tt.expectedErrors, errMsgs)
		})
	}
}

func TestPathParamsRule_UnusedPathParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yaml           string
		expectedErrors []string
	}{
		{
			name: "unused single path param",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
			expectedErrors: []string{
				"[9:7] error semantic-path-params parameter `userId` is declared as path parameter but not used in path template `/users`",
			},
		},
		{
			name: "unused path param at path item level",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    parameters:
      - name: userId
        in: path
        required: true
        schema:
          type: string
    get:
      responses:
        '200':
          description: ok
`,
			expectedErrors: []string{
				"[15:7] error semantic-path-params parameter `userId` is declared as path parameter but not used in path template `/users`",
			},
		},
		{
			name: "one used one unused path param",
			yaml: `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
        - name: postId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`,
			expectedErrors: []string{
				"[9:7] error semantic-path-params parameter `postId` is declared as path parameter but not used in path template `/users/{userId}`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(tt.yaml))
			require.NoError(t, err, "unmarshal should succeed")

			rule := &rules.PathParamsRule{}
			config := &linter.RuleConfig{}

			docInfo := createDocInfoWithIndex(t, ctx, doc, "test.yaml")

			errs := rule.Run(ctx, docInfo, config)

			var errMsgs []string
			for _, err := range errs {
				errMsgs = append(errMsgs, err.Error())
			}

			assert.ElementsMatch(t, tt.expectedErrors, errMsgs)
		})
	}
}

func TestPathParamsRule_RuleMetadata(t *testing.T) {
	t.Parallel()

	rule := &rules.PathParamsRule{}

	assert.Equal(t, "semantic-path-params", rule.ID(), "rule ID should match")
	assert.Equal(t, rules.CategorySemantic, rule.Category(), "rule category should match")
	assert.NotEmpty(t, rule.Description(), "rule should have description")
	assert.Equal(t, validation.SeverityError, rule.DefaultSeverity(), "default severity should be error")
	assert.Nil(t, rule.Versions(), "versions should be nil (all versions)")
}

func TestPathParamsRule_SeverityOverride(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
`
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err, "unmarshal should succeed")

	rule := &rules.PathParamsRule{}
	warningSeverity := validation.SeverityWarning
	config := &linter.RuleConfig{
		Severity: &warningSeverity,
	}

	docInfo := createDocInfoWithIndex(t, ctx, doc, "test.yaml")

	errs := rule.Run(ctx, docInfo, config)
	require.Len(t, errs, 1, "should have one error")

	// Check full error string includes warning severity
	assert.Equal(t, "[9:7] warning semantic-path-params path parameter `{userId}` is not defined in operation parameters", errs[0].Error())
}

func TestPathParamsRule_ExternalReferenceResolution(t *testing.T) {
	t.Parallel()

	t.Run("external reference to parameter resolved successfully", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		// Create a mock HTTP server for this test
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/params/user-id.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`
name: userId
in: path
required: true
schema:
  type: string
`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		yamlInput := fmt.Sprintf(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      parameters:
        - $ref: "%s/params/user-id.yaml"
      responses:
        "200":
          description: ok
`, server.URL)
		doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
		require.NoError(t, err, "unmarshal should succeed")

		rule := &rules.PathParamsRule{}
		config := &linter.RuleConfig{}

		docInfo := createDocInfoWithIndex(t, ctx, doc, server.URL+"/openapi.yaml")

		errs := rule.Run(ctx, docInfo, config)

		// Should have no errors because the external reference resolves to a valid path param
		assert.Empty(t, errs, "should have no lint errors when external ref is resolved")
	})

	t.Run("multiple external references resolved", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		// Create a mock HTTP server for this test
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/params/user-id.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`
name: userId
in: path
required: true
schema:
  type: string
`))
			case "/params/post-id.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`
name: postId
in: path
required: true
schema:
  type: string
`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		yamlInput := fmt.Sprintf(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}/posts/{postId}:
    get:
      parameters:
        - $ref: "%s/params/user-id.yaml"
        - $ref: "%s/params/post-id.yaml"
      responses:
        "200":
          description: ok
`, server.URL, server.URL)
		doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
		require.NoError(t, err, "unmarshal should succeed")

		rule := &rules.PathParamsRule{}
		config := &linter.RuleConfig{}

		docInfo := createDocInfoWithIndex(t, ctx, doc, server.URL+"/openapi.yaml")

		errs := rule.Run(ctx, docInfo, config)

		// Should have no errors - both path params are defined via external refs
		assert.Empty(t, errs, "should have no lint errors when all external refs resolve")
	})

	t.Run("missing path param detected even with external references", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		// Create a mock HTTP server for this test
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/params/user-id.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`
name: userId
in: path
required: true
schema:
  type: string
`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		yamlInput := fmt.Sprintf(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}/posts/{postId}:
    get:
      parameters:
        - $ref: "%s/params/user-id.yaml"
      responses:
        "200":
          description: ok
`, server.URL)
		doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
		require.NoError(t, err, "unmarshal should succeed")

		rule := &rules.PathParamsRule{}
		config := &linter.RuleConfig{}

		docInfo := createDocInfoWithIndex(t, ctx, doc, server.URL+"/openapi.yaml")

		errs := rule.Run(ctx, docInfo, config)

		// Should have one error - postId is not defined
		require.Len(t, errs, 1, "should have one lint error for missing postId")
		assert.Contains(t, errs[0].Error(), "postId", "error should mention postId")
	})

	t.Run("resolution error reported when external reference fails", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		// Create a mock HTTP server that returns 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		yamlInput := fmt.Sprintf(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      parameters:
        - $ref: "%s/params/missing.yaml"
      responses:
        "200":
          description: ok
`, server.URL)
		doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
		require.NoError(t, err, "unmarshal should succeed")

		rule := &rules.PathParamsRule{}
		config := &linter.RuleConfig{}

		docInfo := createDocInfoWithIndex(t, ctx, doc, server.URL+"/openapi.yaml")

		errs := rule.Run(ctx, docInfo, config)

		// Should have errors for both resolution failure and missing param
		require.NotEmpty(t, errs, "should have errors when resolution fails")

		// Check that we have a resolution error
		var foundResolutionError bool
		for _, err := range errs {
			if strings.Contains(err.Error(), "failed to resolve parameter reference") {
				foundResolutionError = true
				break
			}
		}
		assert.True(t, foundResolutionError, "should report resolution error")
	})

	t.Run("resolution error for invalid yaml content", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		// Create a mock HTTP server that returns invalid YAML
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/params/invalid.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`this is not valid: yaml: content: [unclosed`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		yamlInput := fmt.Sprintf(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      parameters:
        - $ref: "%s/params/invalid.yaml"
      responses:
        "200":
          description: ok
`, server.URL)
		doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
		require.NoError(t, err, "unmarshal should succeed")

		rule := &rules.PathParamsRule{}
		config := &linter.RuleConfig{}

		docInfo := createDocInfoWithIndex(t, ctx, doc, server.URL+"/openapi.yaml")

		errs := rule.Run(ctx, docInfo, config)

		// Should have a resolution error for invalid YAML
		require.NotEmpty(t, errs, "should have errors when YAML is invalid")

		var foundResolutionError bool
		for _, err := range errs {
			if strings.Contains(err.Error(), "failed to resolve parameter reference") {
				foundResolutionError = true
				break
			}
		}
		assert.True(t, foundResolutionError, "should report resolution error for invalid YAML")
	})
}
