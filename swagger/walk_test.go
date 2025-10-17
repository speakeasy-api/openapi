package swagger_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/speakeasy-api/openapi/walk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadSwaggerDocument loads a fresh Swagger document for each test to ensure thread safety
func loadSwaggerDocument(ctx context.Context) (*swagger.Swagger, error) {
	f, err := os.Open("testdata/walk.swagger.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s, validationErrs, err := swagger.Unmarshal(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(validationErrs) > 0 {
		return nil, errors.Join(validationErrs...)
	}

	return s, nil
}

func TestWalkSwagger_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Swagger: func(s *swagger.Swagger) error {
				swaggerLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, swaggerLoc)

				if swaggerLoc == expectedLoc {
					assert.Equal(t, "2.0", s.Swagger)
					assert.Equal(t, "api.example.com", s.GetHost())
					assert.Equal(t, "/v1", s.GetBasePath())

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

func TestWalkSwagger_Extensions_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Extensions: func(e *extensions.Extensions) error {
				extensionsLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, extensionsLoc)

				if extensionsLoc == expectedLoc {
					assert.Equal(t, "root-extension", e.GetOrZero("x-root-custom").Value)

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

func TestWalkInfo_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/info"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Info: func(i *swagger.Info) error {
				infoLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, infoLoc)

				if infoLoc == expectedLoc {
					assert.Equal(t, "Comprehensive Swagger API", i.GetTitle())
					assert.Equal(t, "1.0.0", i.GetVersion())
					assert.Equal(t, "A comprehensive Swagger API for testing walk functionality", i.GetDescription())

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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/info/contact"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Contact: func(c *swagger.Contact) error {
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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/info/license"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			License: func(l *swagger.License) error {
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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*swagger.ExternalDocumentation){
		"/externalDocs": func(e *swagger.ExternalDocumentation) {
			assert.Equal(t, "https://example.com/docs", e.GetURL())
			assert.Equal(t, "Additional documentation", e.GetDescription())
		},
		"/tags/0/externalDocs": func(e *swagger.ExternalDocumentation) {
			assert.Equal(t, "https://example.com/users", e.GetURL())
			assert.Equal(t, "User documentation", e.GetDescription())
		},
	}

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			ExternalDocs: func(e *swagger.ExternalDocumentation) error {
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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*swagger.Tag){
		"/tags/0": func(tag *swagger.Tag) {
			assert.Equal(t, "users", tag.GetName())
			assert.Equal(t, "User operations", tag.GetDescription())
		},
		"/tags/1": func(tag *swagger.Tag) {
			assert.Equal(t, "pets", tag.GetName())
			assert.Equal(t, "Pet operations", tag.GetDescription())
		},
	}

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Tag: func(tag *swagger.Tag) error {
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

func TestWalkSecurity_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/security/0"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			SecurityRequirement: func(sr *swagger.SecurityRequirement) error {
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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Paths: func(p *swagger.Paths) error {
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

func TestWalkPathItem_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths/~1users~1{id}"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			PathItem: func(pi *swagger.PathItem) error {
				pathItemLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, pathItemLoc)

				if pathItemLoc == expectedLoc {
					assert.NotNil(t, pi)
					assert.NotNil(t, pi.Get())
					assert.Equal(t, "getUser", pi.Get().GetOperationID())

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

func TestWalkOperation_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths/~1users~1{id}/get"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Operation: func(op *swagger.Operation) error {
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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*swagger.ReferencedParameter){
		"/paths/~1users~1{id}/parameters/0": func(rp *swagger.ReferencedParameter) {
			assert.False(t, rp.IsReference())
			assert.NotNil(t, rp.Object)
			assert.Equal(t, "id", rp.Object.GetName())
			assert.Equal(t, swagger.ParameterInPath, rp.Object.GetIn())
		},
		"/paths/~1users~1{id}/get/parameters/0": func(rp *swagger.ReferencedParameter) {
			assert.False(t, rp.IsReference())
			assert.NotNil(t, rp.Object)
			assert.Equal(t, "expand", rp.Object.GetName())
			assert.Equal(t, swagger.ParameterInQuery, rp.Object.GetIn())
		},
	}

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			ReferencedParameter: func(rp *swagger.ReferencedParameter) error {
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

func TestWalkParameter_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*swagger.Parameter){
		"/paths/~1users~1{id}/parameters/0": func(p *swagger.Parameter) {
			assert.Equal(t, "id", p.GetName())
			assert.Equal(t, swagger.ParameterInPath, p.GetIn())
			assert.Equal(t, "integer", p.GetType())
		},
		"/parameters/PageParam": func(p *swagger.Parameter) {
			assert.Equal(t, "page", p.GetName())
			assert.Equal(t, swagger.ParameterInQuery, p.GetIn())
			assert.Equal(t, "integer", p.GetType())
		},
	}

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Parameter: func(p *swagger.Parameter) error {
				paramLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, paramLoc)

				if assertFunc, exists := expectedAssertions[paramLoc]; exists {
					assertFunc(p)
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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(any){
		"/definitions/User": func(a any) {
			schema, ok := a.(*oas3.JSONSchema[oas3.Concrete])
			assert.True(t, ok)
			assert.NotNil(t, schema)
			// For concrete schemas, the schema object is in Left
			assert.True(t, schema.IsLeft())
			if schema.Left != nil {
				schemaType := schema.Left.GetType()
				assert.Len(t, schemaType, 1)
				assert.Equal(t, oas3.SchemaTypeObject, schemaType[0])
				assert.Equal(t, "User object", schema.Left.GetDescription())
			}
		},
		"/paths/~1users~1{id}/get/responses/200/schema": func(a any) {
			// Schema reference, just verify it exists
			assert.NotNil(t, a)
		},
	}

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Any: func(model any) error {
				loc := string(item.Location.ToJSONPointer())

				if assertFunc, exists := expectedAssertions[loc]; exists {
					matchedLocations = append(matchedLocations, loc)
					assertFunc(model)
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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths/~1users~1{id}/get/responses"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Responses: func(r *swagger.Responses) error {
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

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*swagger.ReferencedResponse){
		"/paths/~1users~1{id}/get/responses/200": func(rr *swagger.ReferencedResponse) {
			assert.False(t, rr.IsReference())
			assert.NotNil(t, rr.Object)
			assert.Equal(t, "Successful response", rr.Object.GetDescription())
		},
		"/paths/~1users~1{id}/get/responses/default": func(rr *swagger.ReferencedResponse) {
			assert.False(t, rr.IsReference())
			assert.NotNil(t, rr.Object)
			assert.Equal(t, "Error response", rr.Object.GetDescription())
		},
	}

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			ReferencedResponse: func(rr *swagger.ReferencedResponse) error {
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

func TestWalkGlobalResponse_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/responses/ErrorResponse"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Response: func(r *swagger.Response) error {
				responseLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, responseLoc)

				if responseLoc == expectedLoc {
					assert.Equal(t, "Error response", r.GetDescription())
					assert.NotNil(t, r.Schema)

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

func TestWalkResponse_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths/~1users~1{id}/get/responses/200"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Response: func(r *swagger.Response) error {
				responseLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, responseLoc)

				if responseLoc == expectedLoc {
					assert.Equal(t, "Successful response", r.GetDescription())
					assert.NotNil(t, r.Schema)

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

func TestWalkHeader_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths/~1users~1{id}/get/responses/200/headers/X-Rate-Limit"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Header: func(h *swagger.Header) error {
				headerLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, headerLoc)

				if headerLoc == expectedLoc {
					assert.Equal(t, "integer", h.GetType())
					assert.Equal(t, "Rate limit remaining", h.GetDescription())

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

func TestWalkItems_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedLoc := "/paths/~1pets/get/parameters/0/items"

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Items: func(i *swagger.Items) error {
				itemsLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, itemsLoc)

				if itemsLoc == expectedLoc {
					assert.Equal(t, "string", i.GetType())

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

func TestWalkSecurityScheme_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	matchedLocations := []string{}
	expectedAssertions := map[string]func(*swagger.SecurityScheme){
		"/securityDefinitions/apiKey": func(ss *swagger.SecurityScheme) {
			assert.Equal(t, swagger.SecuritySchemeTypeAPIKey, ss.GetType())
			assert.Equal(t, "X-API-Key", ss.GetName())
			assert.Equal(t, swagger.SecuritySchemeInHeader, ss.GetIn())
		},
		"/securityDefinitions/oauth2": func(ss *swagger.SecurityScheme) {
			assert.Equal(t, swagger.SecuritySchemeTypeOAuth2, ss.GetType())
			assert.Equal(t, swagger.OAuth2FlowAccessCode, ss.GetFlow())
			assert.Equal(t, "https://example.com/oauth/authorize", ss.GetAuthorizationURL())
			assert.Equal(t, "https://example.com/oauth/token", ss.GetTokenURL())
		},
	}

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			SecurityScheme: func(ss *swagger.SecurityScheme) error {
				schemeLoc := string(item.Location.ToJSONPointer())
				matchedLocations = append(matchedLocations, schemeLoc)

				if assertFunc, exists := expectedAssertions[schemeLoc]; exists {
					assertFunc(ss)
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

func TestWalkAny_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	visitCounts := make(map[string]int)

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
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
	assert.Positive(t, visitCounts["/paths"], "Should visit paths")
	assert.Positive(t, visitCounts["/definitions/User"], "Should visit User definition")

	// Should have visited many locations
	assert.Greater(t, len(visitCounts), 30, "Should visit many locations in comprehensive document")
}

func TestWalk_Terminate_Success(t *testing.T) {
	t.Parallel()

	swaggerDoc, err := loadSwaggerDocument(t.Context())
	require.NoError(t, err)

	visits := 0

	for item := range swagger.Walk(t.Context(), swaggerDoc) {
		err := item.Match(swagger.Matcher{
			Swagger: func(s *swagger.Swagger) error {
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
