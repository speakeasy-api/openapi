package openapi_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundle_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/inline/inline_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure bundling options
	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/inline_input.yaml",
		},
		NamingStrategy: openapi.BundleNamingFilePath,
	}

	// Bundle all external references
	err = openapi.Bundle(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the bundled document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/inline/bundled_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Bundled document should match expected output")
}

func TestBundle_ExternalComponents_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	dir := t.TempDir()
	external := `components:
  callbacks:
    Event:
      '{$request.body#/callbackUrl}':
        $ref: '#/components/pathItems/EventPath'
  pathItems:
    EventPath:
      parameters:
        - $ref: './parameter.yaml#/Id'
      post:
        responses:
          '204':
            description: Accepted
  headers:
    Trace:
      schema:
        $ref: './schema.yaml#/Value'
      examples:
        sample:
          $ref: '#/components/examples/Sample'
  examples:
    Sample:
      value: trace-id
  links:
    Next:
      operationId: nextPage
  securitySchemes:
    Token:
      type: apiKey
      in: header
      name: X-Token
  responses:
    Result:
      description: Result
      headers:
        X-Trace:
          $ref: '#/components/headers/Trace'
      content:
        application/json:
          schema:
            $ref: './schema.yaml#/Value'
        application/jsonl:
          itemSchema:
            $ref: './schema.yaml#/Value'
  requestBodies:
    Input:
      content:
        application/json:
          schema:
            $ref: './schema.yaml#/Value'
        application/jsonl:
          itemSchema:
            $ref: './schema.yaml#/Value'
`
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "api"), 0o755), "create external document directory")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "api/components.yaml"), []byte(external), 0o600), "write external components")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "api/schema.yaml"), []byte("Value:\n  type: string\n  enum: [trace-id]\n"), 0o600), "write source-relative schema")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "api/parameter.yaml"), []byte(`Id:
  name: id
  in: query
  schema:
    $ref: './schema.yaml#/Value'
  examples:
    sample:
      $ref: './components.yaml#/components/examples/Sample'
`), 0o600), "write source-relative parameter")

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.2.0
info:
  title: External components
  version: 1.0.0
paths:
  /events:
    $ref: './api/components.yaml#/components/pathItems/EventPath'
components:
  callbacks:
    ImportedEvent:
      $ref: './api/components.yaml#/components/callbacks/Event'
  headers:
    ImportedTrace:
      $ref: './api/components.yaml#/components/headers/Trace'
  examples:
    ImportedSample:
      $ref: './api/components.yaml#/components/examples/Sample'
  links:
    ImportedNext:
      $ref: './api/components.yaml#/components/links/Next'
  securitySchemes:
    ImportedToken:
      $ref: './api/components.yaml#/components/securitySchemes/Token'
  responses:
    ImportedResult:
      $ref: './api/components.yaml#/components/responses/Result'
  requestBodies:
    ImportedInput:
      $ref: './api/components.yaml#/components/requestBodies/Input'
`))
	require.NoError(t, err, "unmarshal root document")
	require.Empty(t, validationErrs, "root document should be valid")
	err = openapi.Bundle(ctx, doc, openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{RootDocument: doc, TargetLocation: filepath.Join(dir, "openapi.yaml")},
		NamingStrategy: openapi.BundleNamingCounter,
	})
	require.NoError(t, err, "bundle external components and nested source-relative references")

	assert.Equal(t, "#/components/pathItems/EventPath", string(doc.Paths.GetOrZero("/events").GetReference()), "path should reference the bundled path item")
	assert.Equal(t, "#/components/callbacks/Event", string(doc.Components.Callbacks.GetOrZero("ImportedEvent").GetReference()), "callback should be localized")
	assert.Equal(t, "#/components/headers/Trace", string(doc.Components.Headers.GetOrZero("ImportedTrace").GetReference()), "header should be localized")
	assert.Equal(t, "#/components/examples/Sample", string(doc.Components.Examples.GetOrZero("ImportedSample").GetReference()), "example should be localized")
	assert.Equal(t, "#/components/links/Next", string(doc.Components.Links.GetOrZero("ImportedNext").GetReference()), "link should be localized")
	assert.Equal(t, "#/components/securitySchemes/Token", string(doc.Components.SecuritySchemes.GetOrZero("ImportedToken").GetReference()), "security scheme should be localized")
	assert.Equal(t, "#/components/responses/Result", string(doc.Components.Responses.GetOrZero("ImportedResult").GetReference()), "response should be localized")
	assert.Equal(t, "#/components/requestBodies/Input", string(doc.Components.RequestBodies.GetOrZero("ImportedInput").GetReference()), "request body should be localized")

	callback := doc.Components.Callbacks.GetOrZero("Event").GetObject()
	require.NotNil(t, callback, "callback object should be bundled")
	assert.Equal(t, "#/components/pathItems/EventPath", string(callback.GetOrZero("{$request.body#/callbackUrl}").GetReference()), "callback path should use a local reference")
	pathItem := doc.Components.PathItems.GetOrZero("EventPath").GetObject()
	require.NotNil(t, pathItem, "path item should be bundled")
	assert.Equal(t, "#/components/parameters/Id", string(pathItem.Parameters[0].GetReference()), "path parameter should resolve relative to its source document")
	parameter := doc.Components.Parameters.GetOrZero("Id").GetObject()
	require.NotNil(t, parameter, "parameter should be bundled")
	assert.Equal(t, "#/components/schemas/Value", string(parameter.Schema.GetRef()), "parameter schema should be localized")
	assert.Equal(t, "#/components/examples/Sample", string(parameter.Examples.GetOrZero("sample").GetReference()), "parameter example should be localized")
	header := doc.Components.Headers.GetOrZero("Trace").GetObject()
	require.NotNil(t, header, "header object should be bundled")
	assert.Equal(t, "#/components/schemas/Value", string(header.Schema.GetRef()), "header schema should be localized")
	assert.Equal(t, "#/components/examples/Sample", string(header.Examples.GetOrZero("sample").GetReference()), "header example should be localized")
	response := doc.Components.Responses.GetOrZero("Result").GetObject()
	require.NotNil(t, response, "response object should be bundled")
	assert.Equal(t, "#/components/headers/Trace", string(response.Headers.GetOrZero("X-Trace").GetReference()), "response header should be localized")
	assert.Equal(t, "#/components/schemas/Value", string(response.Content.GetOrZero("application/json").Schema.GetRef()), "response schema should be localized")
	assert.Equal(t, "#/components/schemas/Value", string(response.Content.GetOrZero("application/jsonl").ItemSchema.GetRef()), "response item schema should be localized")
	body := doc.Components.RequestBodies.GetOrZero("Input").GetObject()
	require.NotNil(t, body, "request body should be bundled")
	assert.Equal(t, "#/components/schemas/Value", string(body.Content.GetOrZero("application/json").Schema.GetRef()), "request body schema should be localized")
	assert.Equal(t, "#/components/schemas/Value", string(body.Content.GetOrZero("application/jsonl").ItemSchema.GetRef()), "request body item schema should be localized")
	assert.Equal(t, 1, doc.Components.Schemas.Len(), "shared source schema should be bundled only once")
	example := doc.Components.Examples.GetOrZero("Sample").GetObject()
	require.NotNil(t, example, "example object should be bundled")
	assert.Equal(t, "trace-id", example.Value.Value, "example content should be preserved")
	link := doc.Components.Links.GetOrZero("Next").GetObject()
	require.NotNil(t, link, "link object should be bundled")
	assert.Equal(t, "nextPage", link.GetOperationID(), "link content should be preserved")
	scheme := doc.Components.SecuritySchemes.GetOrZero("Token").GetObject()
	require.NotNil(t, scheme, "security scheme object should be bundled")
	assert.Equal(t, "X-Token", scheme.GetName(), "security scheme content should be preserved")

	var output bytes.Buffer
	require.NoError(t, openapi.Marshal(ctx, doc, &output), "marshal bundled document")
	assert.NotContains(t, output.String(), "./api/", "root external references should be removed")
	assert.NotContains(t, output.String(), "./schema.yaml", "nested source-relative references should be removed")
}

func TestBundle_MissingExternalComponent_Error(t *testing.T) {
	t.Parallel()

	for _, section := range []string{"schemas", "callbacks", "pathItems", "headers", "links", "examples", "securitySchemes", "responses", "parameters", "requestBodies"} {
		t.Run(section, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader("openapi: 3.1.0\ninfo:\n  title: Missing component\n  version: 1.0.0\ncomponents:\n  "+section+":\n    Missing:\n      $ref: './missing.yaml#/Missing'\n"))
			require.NoError(t, err, "unmarshal document with unresolved reference")
			require.Empty(t, validationErrs, "document should be valid before resolving")
			err = openapi.Bundle(ctx, doc, openapi.BundleOptions{
				ResolveOptions: openapi.ResolveOptions{RootDocument: doc, TargetLocation: filepath.Join(t.TempDir(), "openapi.yaml")},
			})
			require.Error(t, err, "missing external component should fail bundling")
			assert.Contains(t, err.Error(), "/components/"+section+"/Missing", "error should identify the source reference")
			assert.Contains(t, err.Error(), "missing.yaml", "error should identify the missing file")
		})
	}
}

func TestBundle_CounterNaming_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/inline/inline_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure bundling options with counter naming
	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/inline_input.yaml",
		},
		NamingStrategy: openapi.BundleNamingCounter,
	}

	// Bundle all external references
	err = openapi.Bundle(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the bundled document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/inline/bundled_counter_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Bundled document with counter naming should match expected output")
}

func TestBundle_EmptyDocument(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test with nil document
	err := openapi.Bundle(ctx, nil, openapi.BundleOptions{})
	require.NoError(t, err)

	// Test with minimal document
	doc := &openapi.OpenAPI{
		OpenAPI: openapi.Version,
		Info: openapi.Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
	}

	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   doc,
			TargetLocation: "test.yaml",
		},
		NamingStrategy: openapi.BundleNamingFilePath,
	}

	err = openapi.Bundle(ctx, doc, opts)
	require.NoError(t, err)

	// Document should remain unchanged
	assert.Equal(t, openapi.Version, doc.OpenAPI)
	assert.Equal(t, "Empty API", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)

	// No components should be added
	if doc.Components != nil && doc.Components.Schemas != nil {
		assert.Equal(t, 0, doc.Components.Schemas.Len(), "No schemas should be added for document without external references")
	}
}

func TestBundle_SiblingDirectories_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document with sibling directory references
	inputFile, err := os.Open("testdata/inline/test/openapi.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure bundling options
	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/inline/test/openapi.yaml",
		},
		NamingStrategy: openapi.BundleNamingFilePath,
	}

	// Bundle all external references
	err = openapi.Bundle(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the bundled document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/inline/bundled_sibling_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Bundled document should match expected output")
}

func TestBundle_Issue50_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/bundle/issue50/test/testapi.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Configure bundling options
	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   inputDoc,
			TargetLocation: "testdata/bundle/issue50/test/testapi.yaml",
		},
		NamingStrategy: openapi.BundleNamingFilePath,
	}

	// Bundle all external references
	err = openapi.Bundle(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the bundled document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/bundle/issue50/expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Bundled document should match expected output")
}
