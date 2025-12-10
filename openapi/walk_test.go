package openapi_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/walk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadOpenAPIDocument loads a fresh OpenAPI document for each test to ensure thread safety
func loadOpenAPIDocument(ctx context.Context) (*openapi.OpenAPI, error) {
	f, err := os.Open("testdata/walk.openapi.yaml")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	o, validationErrs, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(validationErrs) > 0 {
		return nil, errors.Join(validationErrs...)
	}

	return o, nil
}

func TestWalkOpenAPI_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			OpenAPI: func(o *openapi.OpenAPI) error {
				openAPILoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, openAPILoc)

				if openAPILoc == expectedLoc {
					assert.Equal(t, "3.1.0", o.OpenAPI)
					assert.Equal(t, o.JSONSchemaDialect, pointer.From("https://json-schema.org/draft/2020-12/schema"))

					return walk.ErrTerminate // Found our target now terminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break // Break out of the iterator loop
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkOpenAPI_Extensions_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Extensions: func(e *extensions.Extensions) error {
				extensionsLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, extensionsLoc)

				if extensionsLoc == expectedLoc {
					assert.Equal(t, "root-extension", e.GetOrZero("x-custom").Value)

					return walk.ErrTerminate // Found our target now terminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break // Break out of the iterator loop
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkInfo_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/info"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Info: func(i *openapi.Info) error {
				infoLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, infoLoc)

				if infoLoc == expectedLoc {
					assert.Equal(t, "Comprehensive API", i.GetTitle())
					assert.Equal(t, "1.0.0", i.GetVersion())
					assert.Equal(t, "A comprehensive API for testing walk functionality", i.GetDescription())

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkContact_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/info/contact"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Contact: func(c *openapi.Contact) error {
				contactLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, contactLoc)

				if contactLoc == expectedLoc {
					assert.Equal(t, "API Team", c.GetName())
					assert.Equal(t, "api@example.com", c.GetEmail())
					assert.Equal(t, "https://example.com/contact", c.GetURL())

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkLicense_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/info/license"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			License: func(l *openapi.License) error {
				licenseLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, licenseLoc)

				if licenseLoc == expectedLoc {
					assert.Equal(t, "MIT", l.GetName())
					assert.Equal(t, "https://opensource.org/licenses/MIT", l.GetURL())

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkExternalDocs_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*oas3.ExternalDocumentation){
		"/externalDocs": func(e *oas3.ExternalDocumentation) {
			assert.Equal(t, "https://example.com/docs", e.GetURL())
			assert.Equal(t, "Additional documentation", e.GetDescription())
		},
		"/tags/0/externalDocs": func(e *oas3.ExternalDocumentation) {
			assert.Equal(t, "https://example.com/users", e.GetURL())
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			ExternalDocs: func(e *oas3.ExternalDocumentation) error {
				externalDocsLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, externalDocsLoc)

				if assertFunc, exists := expectedAssertions[externalDocsLoc]; exists {
					assertFunc(e)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkTag_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*openapi.Tag){
		"/tags/0": func(tag *openapi.Tag) {
			assert.Equal(t, "users", tag.GetName())
			assert.Equal(t, "User operations", tag.GetDescription())
		},
		"/tags/1": func(tag *openapi.Tag) {
			assert.Equal(t, "pets", tag.GetName())
			assert.Equal(t, "Pet operations", tag.GetDescription())
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Tag: func(tag *openapi.Tag) error {
				tagLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, tagLoc)

				if assertFunc, exists := expectedAssertions[tagLoc]; exists {
					assertFunc(tag)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkServer_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*openapi.Server){
		"/servers/0": func(s *openapi.Server) {
			assert.Equal(t, "https://api.example.com/{version}", s.GetURL())
			assert.Equal(t, "Production server", s.GetDescription())
		},
		"/servers/1": func(s *openapi.Server) {
			assert.Equal(t, "https://staging.example.com", s.GetURL())
			assert.Equal(t, "Staging server", s.GetDescription())
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Server: func(s *openapi.Server) error {
				serverLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, serverLoc)

				if assertFunc, exists := expectedAssertions[serverLoc]; exists {
					assertFunc(s)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkServerVariable_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/servers/0/variables/version"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			ServerVariable: func(sv *openapi.ServerVariable) error {
				serverVarLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, serverVarLoc)

				if serverVarLoc == expectedLoc {
					assert.Equal(t, "v1", sv.GetDefault())
					assert.Equal(t, "API version", sv.GetDescription())
					assert.Contains(t, sv.GetEnum(), "v1")
					assert.Contains(t, sv.GetEnum(), "v2")

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkSecurity_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/security/0"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Security: func(sr *openapi.SecurityRequirement) error {
				securityLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, securityLoc)

				if securityLoc == expectedLoc {
					assert.NotNil(t, sr)
					// Security requirement should have apiKey
					apiKeyScopes, exists := sr.Get("apiKey")
					assert.True(t, exists)
					assert.Empty(t, apiKeyScopes) // Empty array for API key

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkPaths_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Paths: func(p *openapi.Paths) error {
				pathsLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, pathsLoc)

				if pathsLoc == expectedLoc {
					assert.NotNil(t, p)
					// Should contain the /users/{id} path
					pathItem, exists := p.Get("/users/{id}")
					assert.True(t, exists)
					assert.NotNil(t, pathItem)

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkReferencedPathItem_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*openapi.ReferencedPathItem){
		"/paths/~1users~1{id}": func(rpi *openapi.ReferencedPathItem) {
			assert.False(t, rpi.IsReference())
			assert.NotNil(t, rpi.Object)
			assert.Equal(t, "User operations", rpi.Object.GetSummary())
		},
		"/webhooks/newUser": func(rpi *openapi.ReferencedPathItem) {
			assert.False(t, rpi.IsReference())
			assert.NotNil(t, rpi.Object)
			assert.Equal(t, "New user webhook", rpi.Object.GetSummary())
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			ReferencedPathItem: func(rpi *openapi.ReferencedPathItem) error {
				pathItemLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, pathItemLoc)

				if assertFunc, exists := expectedAssertions[pathItemLoc]; exists {
					assertFunc(rpi)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkOperation_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths/~1users~1{id}/get"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Operation: func(op *openapi.Operation) error {
				operationLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, operationLoc)

				if operationLoc == expectedLoc {
					assert.Equal(t, "getUser", op.GetOperationID())
					assert.Equal(t, "Get user by ID", op.GetSummary())
					assert.Equal(t, "Retrieve a user by their ID", op.GetDescription())
					assert.Contains(t, op.GetTags(), "users")

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkReferencedParameter_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*openapi.ReferencedParameter){
		"/paths/~1users~1{id}/parameters/0": func(rp *openapi.ReferencedParameter) {
			assert.False(t, rp.IsReference())
			assert.NotNil(t, rp.Object)
			assert.Equal(t, "id", rp.Object.GetName())
			assert.Equal(t, openapi.ParameterInPath, rp.Object.GetIn())
		},
		"/paths/~1users~1{id}/get/parameters/0": func(rp *openapi.ReferencedParameter) {
			// Basic validation for the operation parameter
			assert.NotNil(t, rp)
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			ReferencedParameter: func(rp *openapi.ReferencedParameter) error {
				paramLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, paramLoc)

				if assertFunc, exists := expectedAssertions[paramLoc]; exists {
					assertFunc(rp)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkSchema_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*oas3.JSONSchema[oas3.Referenceable]){
		"/components/schemas/User": func(schema *oas3.JSONSchema[oas3.Referenceable]) {
			assert.True(t, schema.IsLeft())
			s := schema.Left
			schemaType := s.GetType()
			assert.Len(t, schemaType, 1, "User schema should have exactly one type")
			assert.Equal(t, oas3.SchemaTypeObject, schemaType[0])
			assert.Equal(t, "User object", s.GetDescription())
		},
		"/paths/~1users~1{id}/parameters/0/schema": func(schema *oas3.JSONSchema[oas3.Referenceable]) {
			assert.True(t, schema.IsLeft())
			s := schema.Left
			schemaType := s.GetType()
			assert.Len(t, schemaType, 1, "Parameter schema should have exactly one type")
			assert.Equal(t, oas3.SchemaTypeInteger, schemaType[0])
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				schemaLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, schemaLoc)

				if assertFunc, exists := expectedAssertions[schemaLoc]; exists {
					assertFunc(schema)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkMediaType_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		location        string
		assertMediaType func(t *testing.T, mt *openapi.MediaType)
	}{
		{
			name:     "media type with schema",
			location: "/paths/~1users~1{id}/get/requestBody/content/application~1json",
			assertMediaType: func(t *testing.T, mt *openapi.MediaType) {
				t.Helper()
				assert.NotNil(t, mt.Schema, "Schema should not be nil")
				assert.True(t, mt.Schema.IsLeft() || mt.Schema.IsRight(), "Schema should be Left or Right")
				assert.Nil(t, mt.ItemSchema, "ItemSchema should be nil when Schema is present")
			},
		},
		{
			name:     "media type with itemSchema",
			location: "/paths/~1users/post/responses/202/content/application~1json",
			assertMediaType: func(t *testing.T, mt *openapi.MediaType) {
				t.Helper()
				assert.Nil(t, mt.Schema, "Schema should be nil when ItemSchema is present")
				assert.NotNil(t, mt.ItemSchema, "ItemSchema should not be nil")
				assert.True(t, mt.ItemSchema.IsLeft() || mt.ItemSchema.IsRight(), "ItemSchema should be Left or Right")
			},
		},
		{
			name:     "media type with prefixEncoding",
			location: "/paths/~1users/post/responses/203/content/multipart~1mixed",
			assertMediaType: func(t *testing.T, mt *openapi.MediaType) {
				t.Helper()
				assert.NotNil(t, mt.PrefixEncoding, "PrefixEncoding should not be nil")
				assert.Len(t, mt.PrefixEncoding, 2, "PrefixEncoding should have 2 items")
				assert.Equal(t, "application/json", mt.PrefixEncoding[0].GetContentTypeValue(), "First prefix encoding should be application/json")
				assert.Equal(t, "text/plain", mt.PrefixEncoding[1].GetContentTypeValue(), "Second prefix encoding should be text/plain")
				assert.Nil(t, mt.Encoding, "Encoding should be nil when PrefixEncoding is present")
			},
		},
		{
			name:     "media type with itemEncoding",
			location: "/paths/~1users/post/responses/204/content/multipart~1mixed",
			assertMediaType: func(t *testing.T, mt *openapi.MediaType) {
				t.Helper()
				assert.NotNil(t, mt.ItemEncoding, "ItemEncoding should not be nil")
				assert.Equal(t, "application/json", mt.ItemEncoding.GetContentTypeValue(), "ItemEncoding should be application/json")
				assert.Nil(t, mt.Encoding, "Encoding should be nil when ItemEncoding is present")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			openAPIDoc, err := loadOpenAPIDocument(ctx)
			require.NoError(t, err)

			matchedLocations := []string{}
			found := false

			for item := range openapi.Walk(ctx, openAPIDoc) {
				err := item.Match(openapi.Matcher{
					MediaType: func(mt *openapi.MediaType) error {
						mediaTypeLoc := string(item.Location.ToJSONPointer())
						matchedLocations = append(matchedLocations, mediaTypeLoc)

						if mediaTypeLoc == tt.location {
							tt.assertMediaType(t, mt)
							found = true
							return walk.ErrTerminate
						}

						return nil
					},
				})

				if errors.Is(err, walk.ErrTerminate) {
					break
				}
				require.NoError(t, err)
			}

			assert.True(t, found, "Should find MediaType at location: %s", tt.location)
			assert.Contains(t, matchedLocations, tt.location, "Should visit MediaType at location: %s", tt.location)
		})
	}
}

func TestWalkComponents_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/components"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Components: func(c *openapi.Components) error {
				componentsLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, componentsLoc)

				if componentsLoc == expectedLoc {
					assert.NotNil(t, c)
					// Should have schemas
					assert.NotNil(t, c.Schemas)
					userSchema, exists := c.Schemas.Get("User")
					assert.True(t, exists)
					assert.NotNil(t, userSchema)

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkReferencedExample_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*openapi.ReferencedExample){
		"/components/examples/UserExample": func(re *openapi.ReferencedExample) {
			assert.False(t, re.IsReference())
			assert.NotNil(t, re.Object)
			assert.Equal(t, "User example", re.Object.GetSummary())
		},
		"/paths/~1users~1{id}/parameters/0/examples/user-id-example": func(re *openapi.ReferencedExample) {
			assert.False(t, re.IsReference())
			assert.NotNil(t, re.Object)
			assert.Equal(t, "User ID example", re.Object.GetSummary())
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			ReferencedExample: func(re *openapi.ReferencedExample) error {
				exampleLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, exampleLoc)

				if assertFunc, exists := expectedAssertions[exampleLoc]; exists {
					assertFunc(re)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkResponses_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths/~1users~1{id}/get/responses"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Responses: func(r *openapi.Responses) error {
				responsesLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, responsesLoc)

				if responsesLoc == expectedLoc {
					assert.NotNil(t, r)
					// Should have 200 response
					response200, exists := r.Get("200")
					assert.True(t, exists)
					assert.NotNil(t, response200)
					// Should have default response
					assert.NotNil(t, r.Default)

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkReferencedResponse_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*openapi.ReferencedResponse){
		"/paths/~1users~1{id}/get/responses/200": func(rr *openapi.ReferencedResponse) {
			assert.False(t, rr.IsReference())
			assert.NotNil(t, rr.Object)
			assert.Equal(t, "Successful response", rr.Object.GetDescription())
		},
		"/paths/~1users~1{id}/get/responses/default": func(rr *openapi.ReferencedResponse) {
			assert.False(t, rr.IsReference())
			assert.NotNil(t, rr.Object)
			assert.Equal(t, "Error response", rr.Object.GetDescription())
		},
		"/components/responses/ErrorResponse": func(rr *openapi.ReferencedResponse) {
			assert.False(t, rr.IsReference())
			assert.NotNil(t, rr.Object)
			assert.Equal(t, "Error response", rr.Object.GetDescription())
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			ReferencedResponse: func(rr *openapi.ReferencedResponse) error {
				responseLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, responseLoc)

				if assertFunc, exists := expectedAssertions[responseLoc]; exists {
					assertFunc(rr)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkOAuthFlows_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/components/securitySchemes/oauth2/flows"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			OAuthFlows: func(flows *openapi.OAuthFlows) error {
				flowsLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, flowsLoc)

				if flowsLoc == expectedLoc {
					assert.NotNil(t, flows)
					assert.NotNil(t, flows.Implicit)
					assert.NotNil(t, flows.Password)
					assert.NotNil(t, flows.ClientCredentials)
					assert.NotNil(t, flows.AuthorizationCode)

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkOAuthFlow_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*openapi.OAuthFlow){
		"/components/securitySchemes/oauth2/flows/implicit": func(flow *openapi.OAuthFlow) {
			assert.Equal(t, "https://example.com/oauth/authorize", flow.GetAuthorizationURL())
			scopes := flow.GetScopes()
			assert.NotNil(t, scopes)
			assert.True(t, scopes.Has("read"))
			assert.True(t, scopes.Has("write"))
		},
		"/components/securitySchemes/oauth2/flows/password": func(flow *openapi.OAuthFlow) {
			assert.Equal(t, "https://example.com/oauth/token", flow.GetTokenURL())
			scopes := flow.GetScopes()
			assert.NotNil(t, scopes)
			assert.True(t, scopes.Has("read"))
			assert.True(t, scopes.Has("write"))
		},
		"/components/securitySchemes/oauth2/flows/clientCredentials": func(flow *openapi.OAuthFlow) {
			assert.Equal(t, "https://example.com/oauth/token", flow.GetTokenURL())
			scopes := flow.GetScopes()
			assert.NotNil(t, scopes)
			assert.True(t, scopes.Has("read"))
			assert.True(t, scopes.Has("write"))
		},
		"/components/securitySchemes/oauth2/flows/authorizationCode": func(flow *openapi.OAuthFlow) {
			assert.Equal(t, "https://example.com/oauth/authorize", flow.GetAuthorizationURL())
			assert.Equal(t, "https://example.com/oauth/token", flow.GetTokenURL())
			scopes := flow.GetScopes()
			assert.NotNil(t, scopes)
			assert.True(t, scopes.Has("read"))
			assert.True(t, scopes.Has("write"))
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			OAuthFlow: func(flow *openapi.OAuthFlow) error {
				flowLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, flowLoc)

				if assertFunc, exists := expectedAssertions[flowLoc]; exists {
					assertFunc(flow)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc)
	}
}

func TestWalkDiscriminator_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/components/schemas/User/discriminator"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Discriminator: func(d *oas3.Discriminator) error {
				discriminatorLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, discriminatorLoc)

				if discriminatorLoc == expectedLoc {
					assert.Equal(t, "type", d.GetPropertyName())
					mapping := d.GetMapping()
					adminMapping, adminExists := mapping.Get("admin")
					userMapping, userExists := mapping.Get("user")
					assert.True(t, adminExists)
					assert.True(t, userExists)
					assert.Equal(t, "#/components/schemas/AdminUser", adminMapping)
					assert.Equal(t, "#/components/schemas/RegularUser", userMapping)

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkXML_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/components/schemas/User/xml"

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			XML: func(x *oas3.XML) error {
				xmlLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, xmlLoc)

				if xmlLoc == expectedLoc {
					assert.Equal(t, "user", x.GetName())
					assert.Equal(t, "https://example.com/user", x.GetNamespace())
					assert.Equal(t, "usr", x.GetPrefix())
					assert.False(t, x.GetAttribute())
					assert.False(t, x.GetWrapped())

					return walk.ErrTerminate
				}

				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Contains(t, matchedLocations, expectedLoc)
}

func TestWalkComplexSchema_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	complexSchemaLocations := []string{
		"/components/schemas/ComplexSchema",
		"/components/schemas/ComplexSchema/oneOf/0",
		"/components/schemas/ComplexSchema/oneOf/1",
		"/components/schemas/ComplexSchema/anyOf/0",
		"/components/schemas/ComplexSchema/anyOf/1",
		"/components/schemas/ComplexSchema/if",
		"/components/schemas/ComplexSchema/then",
		"/components/schemas/ComplexSchema/else",
		"/components/schemas/ComplexSchema/not",
		"/components/schemas/ComplexSchema/patternProperties/^x-",
		"/components/schemas/ComplexSchema/additionalProperties",
		"/components/schemas/ComplexSchema/contains",
		"/components/schemas/ComplexSchema/prefixItems/0",
		"/components/schemas/ComplexSchema/prefixItems/1",
		"/components/schemas/ComplexSchema/items",
		"/components/schemas/ComplexSchema/propertyNames",
		"/components/schemas/ComplexSchema/dependentSchemas/name",
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				schemaLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, schemaLoc)
				return nil
			},
		})
		require.NoError(t, err)
	}

	// Verify we visited the complex schema locations
	for _, expectedLoc := range complexSchemaLocations {
		assert.Contains(t, matchedLocations, expectedLoc, "Should visit complex schema location: %s", expectedLoc)
	}
}

func TestWalkAny_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	visitCounts := make(map[string]int)

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Any: func(model any) error {
				location := string(item.Location.ToJSONPointer())
				visitCounts[location]++
				return nil
			},
		})
		require.NoError(t, err)
	}

	// Verify we visited key locations
	assert.Positive(t, visitCounts["/"], "Should visit root")
	assert.Positive(t, visitCounts["/info"], "Should visit info")
	assert.Positive(t, visitCounts["/components"], "Should visit components")
	assert.Positive(t, visitCounts["/paths"], "Should visit paths")

	// Should have visited many locations
	assert.Greater(t, len(visitCounts), 50, "Should visit many locations in comprehensive document")
}

func TestWalk_Terminate_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	visits := 0

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			OpenAPI: func(o *openapi.OpenAPI) error {
				return walk.ErrTerminate
			},
			Any: func(a any) error {
				visits++
				return nil
			},
		})

		if errors.Is(err, walk.ErrTerminate) {
			break
		}
		require.NoError(t, err)
	}

	assert.Equal(t, 1, visits, "expected only one visit before terminating")
}

func TestWalkAdditionalOperations_Success(t *testing.T) {
	t.Parallel()

	// Load OpenAPI document with additionalOperations
	f, err := os.Open("testdata/walk.additionaloperations.openapi.yaml")
	require.NoError(t, err)
	defer f.Close()

	openAPIDoc, validationErrs, err := openapi.Unmarshal(t.Context(), f)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Document should be valid")

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*openapi.Operation){
		"/paths/~1custom~1{id}/get": func(op *openapi.Operation) {
			assert.Equal(t, "getCustomResource", op.GetOperationID())
			assert.Equal(t, "Get custom resource", op.GetSummary())
		},
		"/paths/~1custom~1{id}/additionalOperations/COPY": func(op *openapi.Operation) {
			assert.Equal(t, "copyCustomResource", op.GetOperationID())
			assert.Equal(t, "Copy custom resource", op.GetSummary())
			assert.Equal(t, "Custom COPY operation to duplicate a resource", op.GetDescription())
			assert.Contains(t, op.GetTags(), "custom")
		},
		"/paths/~1custom~1{id}/additionalOperations/PURGE": func(op *openapi.Operation) {
			assert.Equal(t, "purgeCustomResource", op.GetOperationID())
			assert.Equal(t, "Purge custom resource", op.GetSummary())
			assert.Equal(t, "Custom PURGE operation to completely remove a resource", op.GetDescription())
			assert.Contains(t, op.GetTags(), "custom")
		},
		"/paths/~1standard/get": func(op *openapi.Operation) {
			assert.Equal(t, "getStandardResource", op.GetOperationID())
			assert.Equal(t, "Get standard resource", op.GetSummary())
		},
	}

	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Operation: func(op *openapi.Operation) error {
				operationLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, operationLoc)

				if assertFunc, exists := expectedAssertions[operationLoc]; exists {
					assertFunc(op)
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	// Verify all expected operations were visited
	for expectedLoc := range expectedAssertions {
		assert.Contains(t, matchedLocations, expectedLoc, "Should visit operation at location: %s", expectedLoc)
	}

	// Verify we found both standard and additional operations
	assert.Contains(t, matchedLocations, "/paths/~1custom~1{id}/get", "Should visit standard GET operation")
	assert.Contains(t, matchedLocations, "/paths/~1custom~1{id}/additionalOperations/COPY", "Should visit additional COPY operation")
	assert.Contains(t, matchedLocations, "/paths/~1custom~1{id}/additionalOperations/PURGE", "Should visit additional PURGE operation")
	assert.Contains(t, matchedLocations, "/paths/~1standard/get", "Should visit standard operation on path without additionalOperations")
}
