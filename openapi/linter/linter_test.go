package linter_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockVirtualFS struct {
	files map[string]string
}

func newMockVirtualFS() *mockVirtualFS {
	return &mockVirtualFS{files: make(map[string]string)}
}

func (m *mockVirtualFS) addFile(path, content string) {
	m.files[path] = content
}

func (m *mockVirtualFS) Open(name string) (fs.File, error) {
	content, exists := m.files[name]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", name)
	}
	return &mockFile{content: content}, nil
}

type mockFile struct {
	content string
	pos     int
}

func (m *mockFile) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.content) {
		return 0, io.EOF
	}
	n = copy(p, m.content[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockFile) Close() error {
	return nil
}

func (m *mockFile) Stat() (fs.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func TestNewLinter(t *testing.T) {
	t.Parallel()

	config := linter.NewConfig()
	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	assert.NotNil(t, lntr)
	assert.NotNil(t, lntr.Registry())
}

func TestOpenAPILinter_PathParamsRule(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
`

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	// Use empty Extends to disable all rules - we only want to test the path params validation
	config := &linter.Config{
		Extends: []string{},
		Rules: []linter.RuleEntry{
			{
				ID:       "semantic-path-params",
				Disabled: pointer.From(false),
			},
		},
	}
	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "/spec/openapi.yaml")
	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	// Should have lint error for missing path parameter
	assert.NotEmpty(t, output.Results)
	assert.True(t, output.HasErrors())
	assert.Contains(t, output.Results[0].Error(), "userId")
}

func TestOpenAPILinter_OutputFormats(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
`

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	config := &linter.Config{
		Extends: []string{"all"},
	}
	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "/spec/openapi.yaml")
	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	t.Run("text format", func(t *testing.T) {
		t.Parallel()

		text := output.FormatText()
		assert.NotEmpty(t, text)
		assert.Contains(t, text, "semantic-path-params")
	})

	t.Run("json format", func(t *testing.T) {
		t.Parallel()

		json := output.FormatJSON()
		assert.NotEmpty(t, json)
		assert.Contains(t, json, "semantic-path-params")
	})
}

func TestOpenAPILinter_ValidDocument(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
`

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	// Use empty Extends to disable all rules - we only want to test the path params validation
	config := &linter.Config{
		Extends: []string{},
		Rules: []linter.RuleEntry{
			{
				ID:       "semantic-path-params",
				Disabled: pointer.From(false),
			},
		},
	}
	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "/spec/openapi.yaml")
	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	// Should have no lint errors for valid document
	assert.Empty(t, output.Results)
	assert.False(t, output.HasErrors())
	assert.Equal(t, 0, output.ErrorCount())
}

func TestOpenAPILinter_IndexValidationErrorsExposed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: 'schema.yaml'
`

	fs := newMockVirtualFS()
	fs.addFile("/spec/schema.yaml", "type: invalid_type")

	resolveOpts := &references.ResolveOptions{
		VirtualFS: fs,
	}

	// Use empty Extends to disable all lint rules - we only want index validation errors
	config := &linter.Config{
		Extends: []string{},
	}

	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/openapi.yaml"), nil, &linter.LintOptions{
		ResolveOptions: resolveOpts,
	})
	require.NoError(t, err)

	var errorStrings []string
	for _, result := range output.Results {
		errorStrings = append(errorStrings, result.Error())
	}

	assert.ElementsMatch(t, []string{
		"[1:7] error validation-invalid-schema schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string' (document: /spec/schema.yaml)",
		"[1:7] error validation-type-mismatch schema.type expected array, got string (document: /spec/schema.yaml)",
	}, errorStrings)
}

func TestOpenAPILinter_IndexResolutionErrorsExposed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: 'missing.yaml'
`

	fs := newMockVirtualFS()
	resolveOpts := &references.ResolveOptions{
		VirtualFS: fs,
	}

	// Use empty Extends to disable all rules - we only want to test the path params validation
	config := &linter.Config{
		Extends: []string{},
		Rules: []linter.RuleEntry{
			{
				ID:       "semantic-path-params",
				Disabled: pointer.From(false),
			},
		},
	}

	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/openapi.yaml"), nil, &linter.LintOptions{
		ResolveOptions: resolveOpts,
	})
	require.NoError(t, err)

	var resolutionErrors []string
	for _, result := range output.Results {
		resolutionErrors = append(resolutionErrors, result.Error())
	}

	assert.Equal(t, []string{
		"[16:17] error resolution-json-schema file not found: /spec/missing.yaml",
	}, resolutionErrors)
}

func TestOpenAPILinter_IndexCircularReferenceErrorsExposed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Node'
components:
  schemas:
    Node:
      type: object
      required: [child]
      properties:
        child:
          $ref: '/spec/external.yaml#/ExternalNode'
`

	fs := newMockVirtualFS()
	fs.addFile("/spec/openapi.yaml", yamlInput)
	fs.addFile("/spec/external.yaml", `
ExternalNode:
  type: object
  required: [child]
  properties:
    child:
      $ref: '/spec/openapi.yaml#/components/schemas/Node'
`)

	resolveOpts := &references.ResolveOptions{
		VirtualFS: fs,
	}

	// Use empty Extends to disable all lint rules - we only want index circular reference errors
	config := &linter.Config{
		Extends: []string{},
	}

	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/openapi.yaml"), nil, &linter.LintOptions{
		ResolveOptions: resolveOpts,
	})
	require.NoError(t, err)

	var circularErrors []string
	for _, result := range output.Results {
		circularErrors = append(circularErrors, result.Error())
	}

	assert.ElementsMatch(t, []string{
		"[7:7] error circular-reference-invalid non-terminating circular reference detected: /spec/openapi.yaml#/components/schemas/Node -> /spec/external.yaml#/ExternalNode -> /spec/openapi.yaml#/components/schemas/Node (document: /spec/external.yaml)",
		"[24:11] error circular-reference-invalid non-terminating circular reference detected: /spec/external.yaml#/ExternalNode -> /spec/openapi.yaml#/components/schemas/Node -> /spec/external.yaml#/ExternalNode",
	}, circularErrors)
}

func TestOpenAPILinter_ExternalDocumentDetailsExposed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: 'schema.yaml'
`

	fs := newMockVirtualFS()
	fs.addFile("/spec/schema.yaml", "type: invalid_type")

	resolveOpts := &references.ResolveOptions{
		VirtualFS: fs,
	}

	// Use empty Extends to disable all lint rules - we only want index validation errors from external docs
	config := &linter.Config{
		Extends: []string{},
	}

	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	output, err := lntr.Lint(ctx, linter.NewDocumentInfo(doc, "/spec/openapi.yaml"), nil, &linter.LintOptions{
		ResolveOptions: resolveOpts,
	})
	require.NoError(t, err)

	var documentErrors []string
	for _, result := range output.Results {
		documentErrors = append(documentErrors, result.Error())
	}

	assert.ElementsMatch(t, []string{
		"[1:7] error validation-invalid-schema schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string' (document: /spec/schema.yaml)",
		"[1:7] error validation-type-mismatch schema.type expected array, got string (document: /spec/schema.yaml)",
	}, documentErrors)
}

func TestNewLinter_WithoutDefaultRules(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create linter with no rules
	config := linter.NewConfig()
	lntr, err := openapiLinter.NewLinter(config, openapiLinter.WithoutDefaultRules())
	require.NoError(t, err)

	require.NotNil(t, lntr)
	require.NotNil(t, lntr.Registry())

	// Verify no rules registered
	assert.Empty(t, lntr.Registry().AllRules(), "should have no rules registered")

	// Parse test document with known linting issues
	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
`

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "/spec/openapi.yaml")
	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	// Should have no lint errors since no rules are registered
	assert.Empty(t, output.Results, "should have no lint errors with no rules")
	assert.False(t, output.HasErrors())
}

func TestNewLinter_WithoutDefaultRules_ManualRegistration(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create linter with no rules
	config := &linter.Config{
		Extends: []string{"all"}, // Enable all registered rules
	}
	lntr, err := openapiLinter.NewLinter(config, openapiLinter.WithoutDefaultRules())
	require.NoError(t, err)

	// Manually register specific rules
	lntr.Registry().Register(&rules.PathParamsRule{})

	// Verify only one rule registered
	assert.Len(t, lntr.Registry().AllRules(), 1, "should have exactly one rule registered")

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
`

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "/spec/openapi.yaml")
	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	// Should have error from PathParamsRule only
	assert.NotEmpty(t, output.Results, "should have lint errors from registered rule")
	assert.Contains(t, output.Results[0].Error(), "userId", "error should be about missing userId parameter")
}

func TestNewLinter_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	// Verify NewLinter() without options still registers all default rules
	config := linter.NewConfig()
	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	// Should have many rules registered (60+)
	assert.Greater(t, len(lntr.Registry().AllRules()), 60, "should have all default rules registered")
}

func TestNewLinter_Rulesets(t *testing.T) {
	t.Parallel()

	config := linter.NewConfig()
	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	t.Run("recommended ruleset exists", func(t *testing.T) {
		t.Parallel()

		ruleIDs, ok := lntr.Registry().GetRuleset("recommended")
		assert.True(t, ok, "recommended ruleset should exist")
		assert.NotEmpty(t, ruleIDs, "recommended ruleset should have rules")

		// Verify it has a reasonable number of rules (not too many, not too few)
		assert.GreaterOrEqual(t, len(ruleIDs), 20, "recommended should have at least 20 rules")
		assert.LessOrEqual(t, len(ruleIDs), 40, "recommended should have at most 40 rules")

		// Verify key rules are included
		assert.Contains(t, ruleIDs, rules.RuleSemanticPathParams, "should include path-params rule")
		assert.Contains(t, ruleIDs, rules.RuleSemanticOperationOperationId, "should include operation-id rule")
		assert.Contains(t, ruleIDs, rules.RuleOwaspNoCredentialsInURL, "should include basic security rules")
	})

	t.Run("security ruleset exists", func(t *testing.T) {
		t.Parallel()

		ruleIDs, ok := lntr.Registry().GetRuleset("security")
		assert.True(t, ok, "security ruleset should exist")
		assert.NotEmpty(t, ruleIDs, "security ruleset should have rules")

		// Verify all OWASP rules are included
		assert.Contains(t, ruleIDs, rules.RuleOwaspNoHttpBasic)
		assert.Contains(t, ruleIDs, rules.RuleOwaspNoAPIKeysInURL)
		assert.Contains(t, ruleIDs, rules.RuleOwaspDefineErrorResponses401)
		assert.Contains(t, ruleIDs, rules.RuleOwaspArrayLimit)
	})

	t.Run("all ruleset always exists", func(t *testing.T) {
		t.Parallel()

		ruleIDs, ok := lntr.Registry().GetRuleset("all")
		assert.True(t, ok, "all ruleset should exist")
		assert.NotEmpty(t, ruleIDs, "all ruleset should have rules")

		// "all" should include every registered rule
		allRules := lntr.Registry().AllRules()
		assert.Equal(t, len(allRules), len(ruleIDs), "all ruleset should include every rule")
	})

	t.Run("AllRulesets returns available rulesets", func(t *testing.T) {
		t.Parallel()

		rulesets := lntr.Registry().AllRulesets()
		assert.Contains(t, rulesets, "all")
		assert.Contains(t, rulesets, "recommended")
		assert.Contains(t, rulesets, "security")
	})
}

func TestLinter_RecommendedRulesetUsage(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yamlInput := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
`

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(yamlInput))
	require.NoError(t, err)

	// Use recommended ruleset
	config := &linter.Config{
		Extends: []string{"recommended"},
	}
	lntr, err := openapiLinter.NewLinter(config)
	require.NoError(t, err)

	docInfo := linter.NewDocumentInfo(doc, "/spec/openapi.yaml")
	output, err := lntr.Lint(ctx, docInfo, nil, nil)
	require.NoError(t, err)

	// Should have lint errors from recommended rules
	assert.NotEmpty(t, output.Results, "recommended ruleset should catch issues")

	// Verify at least the path params error is caught
	var hasPathParamsError bool
	for _, result := range output.Results {
		if strings.Contains(result.Error(), "semantic-path-params") {
			hasPathParamsError = true
			break
		}
	}
	assert.True(t, hasPathParamsError, "recommended ruleset should catch path params error")
}
