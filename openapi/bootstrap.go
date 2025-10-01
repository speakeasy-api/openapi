package openapi

import (
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	"github.com/speakeasy-api/openapi/yml"
)

// Bootstrap creates a new OpenAPI document with best practice examples.
// This serves as a template demonstrating proper structure for operations,
// components, security schemes, servers, tags, and references.
func Bootstrap() *OpenAPI {
	return &OpenAPI{
		OpenAPI: Version,
		Info:    createBootstrapInfo(),
		Servers: createBootstrapServers(),
		Tags:    createBootstrapTags(),
		Security: []*SecurityRequirement{
			NewSecurityRequirement(
				sequencedmap.NewElem("ApiKeyAuth", []string{}),
			),
		},
		Paths:      createBootstrapPaths(),
		Components: createBootstrapComponents(),
	}
}

// createBootstrapInfo creates a complete info section with best practices
func createBootstrapInfo() Info {
	return Info{
		Title:       "My API",
		Description: pointer.From("A new OpenAPI document template ready to populate"),
		Version:     "1.0.0",
		Contact: &Contact{
			Name:  pointer.From("API Support"),
			Email: pointer.From("support@example.com"),
			URL:   pointer.From("https://example.com/support"),
		},
		License: &License{
			Name: "MIT",
			URL:  pointer.From("https://opensource.org/licenses/MIT"),
		},
		TermsOfService: pointer.From("https://example.com/terms"),
	}
}

// createBootstrapServers creates example servers for different environments
func createBootstrapServers() []*Server {
	return []*Server{
		{
			URL:         "https://api.example.com/v1",
			Description: pointer.From("Production server"),
		},
		{
			URL:         "https://staging-api.example.com/v1",
			Description: pointer.From("Staging server"),
		},
	}
}

// createBootstrapTags creates example tags for organizing operations
func createBootstrapTags() []*Tag {
	return []*Tag{
		{
			Name:        "users",
			Description: pointer.From("User management operations"),
			ExternalDocs: &oas3.ExternalDocumentation{
				Description: pointer.From("User API documentation"),
				URL:         "https://docs.example.com/users",
			},
		},
	}
}

// createBootstrapPaths creates a single POST operation demonstrating best practices
func createBootstrapPaths() *Paths {
	paths := NewPaths()

	// Create a single POST operation as an example
	usersPath := NewPathItem()
	usersPath.Set(HTTPMethodPost, &Operation{
		OperationID: pointer.From("createUser"),
		Summary:     pointer.From("Create a new user"),
		Description: pointer.From("Creates a new user account with the provided information"),
		Tags:        []string{"users"},
		RequestBody: NewReferencedRequestBodyFromRequestBody(&RequestBody{
			Description: pointer.From("User creation request"),
			Required:    pointer.From(true),
			Content: sequencedmap.New(
				sequencedmap.NewElem("application/json", &MediaType{
					Schema: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
						Type: oas3.NewTypeFromString(oas3.SchemaTypeObject),
						Properties: sequencedmap.New(
							sequencedmap.NewElem("name", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
								Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
								Description: pointer.From("Full name of the user"),
								MinLength:   pointer.From(int64(1)),
								MaxLength:   pointer.From(int64(100)),
							})),
							sequencedmap.NewElem("email", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
								Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
								Format:      pointer.From("email"),
								Description: pointer.From("Email address of the user"),
							})),
						),
						Required: []string{"name", "email"},
					}),
				}),
			),
		}),
		Responses: createUserResponses(),
	})

	paths.Set("/users", &ReferencedPathItem{Object: usersPath})
	return paths
}

// createUserResponses creates example responses with references
func createUserResponses() Responses {
	return NewResponses(
		sequencedmap.NewElem("201", NewReferencedResponseFromRef("#/components/responses/UserResponse")),
		sequencedmap.NewElem("400", NewReferencedResponseFromRef("#/components/responses/BadRequestResponse")),
		sequencedmap.NewElem("401", NewReferencedResponseFromRef("#/components/responses/UnauthorizedResponse")),
	)
}

// createBootstrapComponents creates reusable components demonstrating best practices
func createBootstrapComponents() *Components {
	return &Components{
		Schemas:         createBootstrapSchemas(),
		Responses:       createBootstrapResponses(),
		SecuritySchemes: createBootstrapSecuritySchemes(),
	}
}

// createBootstrapSchemas creates example schemas with proper structure
func createBootstrapSchemas() *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]] {
	return sequencedmap.New(
		sequencedmap.NewElem("User", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type:        oas3.NewTypeFromString(oas3.SchemaTypeObject),
			Title:       pointer.From("User"),
			Description: pointer.From("A user account"),
			Properties: sequencedmap.New(
				sequencedmap.NewElem("id", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type:        oas3.NewTypeFromString(oas3.SchemaTypeInteger),
					Format:      pointer.From("int64"),
					Description: pointer.From("Unique identifier for the user"),
					Examples:    []values.Value{yml.CreateIntNode(123)},
				})),
				sequencedmap.NewElem("name", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
					Description: pointer.From("Full name of the user"),
					MinLength:   pointer.From(int64(1)),
					MaxLength:   pointer.From(int64(100)),
					Examples:    []values.Value{yml.CreateStringNode("John Doe")},
				})),
				sequencedmap.NewElem("email", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
					Format:      pointer.From("email"),
					Description: pointer.From("Email address of the user"),
					Examples:    []values.Value{yml.CreateStringNode("john.doe@example.com")},
				})),
				sequencedmap.NewElem("status", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
					Description: pointer.From("Current status of the user account"),
					Enum:        []values.Value{yml.CreateStringNode("active"), yml.CreateStringNode("inactive"), yml.CreateStringNode("pending")},
					Examples:    []values.Value{yml.CreateStringNode("active")},
				})),
			),
			Required: []string{"name", "email"},
		})),
		sequencedmap.NewElem("Error", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type:        oas3.NewTypeFromString(oas3.SchemaTypeObject),
			Title:       pointer.From("Error"),
			Description: pointer.From("Error response"),
			Properties: sequencedmap.New(
				sequencedmap.NewElem("code", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
					Description: pointer.From("Error code"),
					Examples:    []values.Value{yml.CreateStringNode("VALIDATION_ERROR")},
				})),
				sequencedmap.NewElem("message", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
					Description: pointer.From("Human-readable error message"),
					Examples:    []values.Value{yml.CreateStringNode("The request is invalid")},
				})),
			),
			Required: []string{"code", "message"},
		})),
	)
}

// createBootstrapResponses creates reusable response components
func createBootstrapResponses() *sequencedmap.Map[string, *ReferencedResponse] {
	return sequencedmap.New(
		sequencedmap.NewElem("UserResponse", &ReferencedResponse{
			Object: &Response{
				Description: "User details",
				Content: sequencedmap.New(
					sequencedmap.NewElem("application/json", &MediaType{
						Schema: oas3.NewJSONSchemaFromReference("#/components/schemas/User"),
					}),
				),
			},
		}),
		sequencedmap.NewElem("BadRequestResponse", &ReferencedResponse{
			Object: &Response{
				Description: "Bad request - validation error",
				Content: sequencedmap.New(
					sequencedmap.NewElem("application/json", &MediaType{
						Schema: oas3.NewJSONSchemaFromReference("#/components/schemas/Error"),
					}),
				),
			},
		}),
		sequencedmap.NewElem("UnauthorizedResponse", &ReferencedResponse{
			Object: &Response{
				Description: "Unauthorized - authentication required",
				Content: sequencedmap.New(
					sequencedmap.NewElem("application/json", &MediaType{
						Schema: oas3.NewJSONSchemaFromReference("#/components/schemas/Error"),
					}),
				),
			},
		}),
	)
}

// createBootstrapSecuritySchemes creates example security schemes
func createBootstrapSecuritySchemes() *sequencedmap.Map[string, *ReferencedSecurityScheme] {
	return sequencedmap.New(
		sequencedmap.NewElem("ApiKeyAuth", &ReferencedSecurityScheme{
			Object: &SecurityScheme{
				Type:        "apiKey",
				In:          pointer.From[SecuritySchemeIn]("header"),
				Name:        pointer.From("X-API-Key"),
				Description: pointer.From("API key for authentication"),
			},
		}),
	)
}
