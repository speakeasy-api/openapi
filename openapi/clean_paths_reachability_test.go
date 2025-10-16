package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensures Clean only preserves components reachable from /paths and security,
// and removes components that are only referenced from within components.
//
// Covers scenarios missed previously:
// - Keep schemas transitively reachable from operations (A -> B)
// - Remove self-referential and component-only cycles (Self, Cycle1, Cycle2)
// - Keep security schemes referenced by name via top-level security
// - Remove unused security schemes
func TestClean_ReachabilityFromPathsAndSecurity_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	const yml = `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
security:
  - ApiKeyAuth: []
paths:
  /keep:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/A"
components:
  schemas:
    A:
      type: object
      properties:
        b:
          $ref: "#/components/schemas/B"
    B:
      type: string
    Self:
      $ref: "#/components/schemas/Self"
    Cycle1:
      $ref: "#/components/schemas/Cycle2"
    Cycle2:
      $ref: "#/components/schemas/Cycle1"
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
    UnusedScheme:
      type: http
      scheme: bearer
`

	// Unmarshal
	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "input should be valid")

	// Clean
	err = openapi.Clean(ctx, doc)
	require.NoError(t, err, "clean should succeed")

	// Marshal and assert against expected YAML output
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, doc, &buf)
	require.NoError(t, err, "marshal should succeed")
	actual := buf.String()

	const expected = `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
security:
  - ApiKeyAuth: []
paths:
  /keep:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/A"
components:
  schemas:
    A:
      type: object
      properties:
        b:
          $ref: "#/components/schemas/B"
    B:
      type: string
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
`

	assert.Equal(t, expected, actual, "Clean should retain only reachable components (A, B) and ApiKeyAuth")
}

// Ensures that when no paths (or top-level/operation security) reference components,
// purely self-referential or component-only cycles are all removed and the entire
// components section is dropped.
func TestClean_RemoveOnlySelfReferencedComponents_NoPaths_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	const yml = `
openapi: 3.1.0
info:
  title: Only Self-Referenced Components
  version: 1.0.0
paths: {}
components:
  schemas:
    Self:
      $ref: "#/components/schemas/Self"
    LoopA:
      $ref: "#/components/schemas/LoopB"
    LoopB:
      $ref: "#/components/schemas/LoopA"
  responses:
    OnlyComponentResponse:
      description: Component-only, not referenced from any path
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Self"
  parameters:
    OnlyComponentParameter:
      name: "p"
      in: query
      schema:
        type: string
  requestBodies:
    OnlyComponentRequestBody:
      required: false
      content:
        application/json:
          schema:
            type: object
  headers:
    OnlyComponentHeader:
      schema:
        type: string
  examples:
    OnlyComponentExample:
      value:
        ok: true
  links:
    OnlyComponentLink:
      description: Not referenced from paths
      parameters:
        id: "$response.body#/id"
  callbacks:
    OnlyComponentCallback:
      "{$request.body#/cb}":
        post:
          requestBody:
            content:
              application/json:
                schema:
                  type: object
          responses:
            "200":
              description: ok
  pathItems:
    OnlyComponentPathItem:
      get:
        responses:
          "200":
            description: ok
  securitySchemes:
    UnusedApiKey:
      type: apiKey
      in: header
      name: X-API-Key
    UnusedBearer:
      type: http
      scheme: bearer
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "input should be valid")

	err = openapi.Clean(ctx, doc)
	require.NoError(t, err, "clean should succeed")

	// Marshal and assert against expected YAML output
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, doc, &buf)
	require.NoError(t, err, "marshal should succeed")
	actual := buf.String()

	const expected = `openapi: 3.1.0
info:
  title: Only Self-Referenced Components
  version: 1.0.0
paths: {}
`

	assert.Equal(t, expected, actual, "All components should be removed when only self/component-only references exist")
}
