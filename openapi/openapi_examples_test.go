package openapi_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/walk"
	"github.com/speakeasy-api/openapi/yml"
)

// The below examples should be copied into the README.md file if ever changed

// Example_reading demonstrates how to read and parse an OpenAPI document from a file.
// This includes validation by default and shows how to access document properties.
func Example_reading() {
	ctx := context.Background()

	r, err := os.Open("testdata/test.openapi.yaml")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Unmarshal the OpenAPI document which will also validate it against the OpenAPI Specification
	doc, validationErrs, err := openapi.Unmarshal(ctx, r /*, openapi.WithSkipValidation()*/) // Optionally skip validation
	if err != nil {
		panic(err)
	}

	// Validation errors are returned separately from any errors that block the document from being unmarshalled
	// allowing an invalid document to be mutated and fixed before being marshalled again
	for _, err := range validationErrs {
		fmt.Println(err.Error())
	}

	fmt.Printf("OpenAPI Version: %s\n", doc.OpenAPI)
	fmt.Printf("API Title: %s\n", doc.Info.Title)
	fmt.Printf("API Version: %s\n", doc.Info.Version)
	// Output: OpenAPI Version: 3.2.0
	// API Title: Test OpenAPI Document
	// API Version: 1.0.0
}

// Example_workingWithJSONSchema demonstrates how to work with JSON Schema directly.
// Shows how to unmarshal a JSONSchema from YAML or JSON and validate it manually.
func Example_workingWithJSONSchema() {
	ctx := context.Background()

	// Example JSON Schema as YAML
	schemaYAML := `
type: object
properties:
  id:
    type: integer
    format: int64
  name:
    type: string
    maxLength: 100
  email:
    type: string
    format: email
required:
  - id
  - name
  - email
`

	// Unmarshal directly to a JSONSchema using marshaller.Unmarshal
	var schema oas3.JSONSchema[oas3.Concrete]
	validationErrs, err := marshaller.Unmarshal(ctx, bytes.NewReader([]byte(schemaYAML)), &schema)
	if err != nil {
		panic(err)
	}

	// Validate manually
	additionalErrs := schema.Validate(ctx)
	validationErrs = append(validationErrs, additionalErrs...)

	if len(validationErrs) > 0 {
		for _, err := range validationErrs {
			fmt.Println("Validation error:", err.Error())
		}
	}

	// Access schema properties
	if schema.IsSchema() {
		schemaObj := schema.GetSchema()
		fmt.Println("Schema Types:")
		for _, t := range schemaObj.GetType() {
			fmt.Printf("  %s\n", t)
		}
		fmt.Printf("Required Fields: %v\n", schemaObj.GetRequired())
		fmt.Printf("Number of Properties: %d\n", schemaObj.GetProperties().Len())
	}
	// Output: Schema Types:
	//   object
	// Required Fields: [id name email]
	// Number of Properties: 3
}

// Example_marshaling demonstrates how to marshal an OpenAPI document to a writer.
// Shows creating a simple document and outputting it as YAML.
func Example_marshaling() {
	ctx := context.Background()

	// Create a simple OpenAPI document
	doc := &openapi.OpenAPI{
		OpenAPI: openapi.Version,
		Info: openapi.Info{
			Title:   "Example API",
			Version: "1.0.0",
		},
		Paths: openapi.NewPaths(),
	}

	buf := bytes.NewBuffer([]byte{})

	// Marshal the document to a writer
	if err := openapi.Marshal(ctx, doc, buf); err != nil {
		panic(err)
	}

	fmt.Printf("%s", buf.String())
	// Output: openapi: 3.2.0
	// info:
	//   title: Example API
	//   version: 1.0.0
	// paths: {}
}

// Example_marshalingJSONSchema demonstrates how to marshal a JSONSchema directly.
// Shows creating a schema programmatically and outputting it as YAML.
func Example_marshalingJSONSchema() {
	ctx := context.Background()

	// Create a JSONSchema programmatically
	properties := sequencedmap.New(
		sequencedmap.NewElem("id", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type:   oas3.NewTypeFromString(oas3.SchemaTypeInteger),
			Format: pointer.From("int64"),
		})),
		sequencedmap.NewElem("name", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type:      oas3.NewTypeFromString(oas3.SchemaTypeString),
			MaxLength: pointer.From(int64(100)),
		})),
	)

	schema := oas3.NewJSONSchemaFromSchema[oas3.Concrete](&oas3.Schema{
		Type:       oas3.NewTypeFromString(oas3.SchemaTypeObject),
		Properties: properties,
		Required:   []string{"id", "name"},
	})

	buf := bytes.NewBuffer([]byte{})

	// Marshal the schema using marshaller.Marshal
	if err := marshaller.Marshal(ctx, schema, buf); err != nil {
		panic(err)
	}

	fmt.Printf("%s", buf.String())
	// Output: type: object
	// properties:
	//   id:
	//     type: integer
	//     format: int64
	//   name:
	//     type: string
	//     maxLength: 100
	// required:
	//   - id
	//   - name
}

// Example_validating demonstrates how to validate an OpenAPI document and fix validation errors.
// Shows automatic validation during unmarshaling, fixing errors programmatically, and re-validating.
func Example_validating() {
	ctx := context.Background()

	path := "testdata/invalid.openapi.yaml"

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Unmarshal with validation (default behavior)
	doc, validationErrs, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Initial validation errors: %d\n", len(validationErrs))
	for _, err := range validationErrs {
		fmt.Printf("  %s\n", err.Error())
	}

	// Fix some of the validation errors programmatically
	fmt.Println("\nFixing validation errors...")

	// Fix 1: Add missing info.version
	doc.Info.Version = "1.0.0"
	fmt.Println("  ✓ Added missing info.version")

	// Fix 2: Add missing responses to the POST /users operation
	if doc.Paths != nil {
		if pathItem, ok := doc.Paths.Get("/users/{id}"); ok && pathItem.GetObject() != nil {
			if post := pathItem.GetObject().Post(); post != nil {
				post.Responses.Set("200", &openapi.ReferencedResponse{
					Object: &openapi.Response{
						Description: "Success",
					},
				})
				fmt.Println("  ✓ Added missing response to POST /users/{id}")
			}
		}
	}

	// Fix 3: Add missing responses to the POST /invalid operation
	if doc.Paths != nil {
		if pathItem, ok := doc.Paths.Get("/invalid"); ok && pathItem.GetObject() != nil {
			if post := pathItem.GetObject().Post(); post != nil {
				post.Responses = openapi.NewResponses()
				post.Responses.Set("200", &openapi.ReferencedResponse{
					Object: &openapi.Response{
						Description: "Success",
					},
				})
				fmt.Println("  ✓ Added missing responses to POST /invalid")
			}
		}
	}

	// Re-validate after fixes
	newValidationErrs := doc.Validate(ctx)
	validation.SortValidationErrors(newValidationErrs)

	fmt.Printf("\nValidation errors after fixes: %d\n", len(newValidationErrs))
	for _, err := range newValidationErrs {
		fmt.Printf("  %s\n", err.Error())
	}

	fmt.Printf("\nReduced validation errors from %d to %d\n", len(validationErrs), len(newValidationErrs))
	// Output: Initial validation errors: 16
	//   [3:3] info.version is missing
	//   [22:17] schema.type.0 expected string, got null
	//   [28:30] response.content.application/json expected object, got ``
	//   [31:18] responses must have at least one response code
	//   [34:7] operation.responses is missing
	//   [43:17] schema.properties.required failed to validate either Schema [schema.properties.required expected object, got sequence] or bool [schema.properties.required expected bool, got sequence]
	//   [51:25] schema.properties.name.type expected array, got string
	//   [51:25] schema.properties.name.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'
	//   [56:7] schema.examples expected array, got object
	//   [59:15] schema.properties.name expected one of [boolean, object], got string
	//   [59:15] schema.properties.name expected one of [boolean, object], got string
	//   [59:15] schema.properties.name failed to validate either Schema [schema.properties.name expected object, got `string`] or bool [schema.properties.name line 59: cannot unmarshal !!str `string` into bool]
	//   [60:18] schema.properties.example expected one of [boolean, object], got string
	//   [60:18] schema.properties.example expected one of [boolean, object], got string
	//   [60:18] schema.properties.example failed to validate either Schema [schema.properties.example expected object, got `John Doe`] or bool [schema.properties.example line 60: cannot unmarshal !!str `John Doe` into bool]
	//   [63:9] schema.examples expected sequence, got object
	//
	// Fixing validation errors...
	//   ✓ Added missing info.version
	//   ✓ Added missing response to POST /users/{id}
	//   ✓ Added missing responses to POST /invalid
	//
	// Validation errors after fixes: 7
	//   [51:25] schema.properties.name.type expected array, got string
	//   [51:25] schema.properties.name.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'
	//   [56:7] schema.examples expected array, got object
	//   [59:15] schema.properties.name expected one of [boolean, object], got string
	//   [59:15] schema.properties.name expected one of [boolean, object], got string
	//   [60:18] schema.properties.example expected one of [boolean, object], got string
	//   [60:18] schema.properties.example expected one of [boolean, object], got string
	//
	// Reduced validation errors from 16 to 7
}

// Example_mutating demonstrates how to read and modify an OpenAPI document.
// Shows loading a document, making changes, and marshaling it back to YAML.
func Example_mutating() {
	ctx := context.Background()

	r, err := os.Open("testdata/simple.openapi.yaml")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Unmarshal the OpenAPI document
	doc, validationErrs, err := openapi.Unmarshal(ctx, r)
	if err != nil {
		panic(err)
	}

	// Print any validation errors
	for _, err := range validationErrs {
		fmt.Println(err.Error())
	}

	// Mutate the document by modifying the returned OpenAPI object
	doc.Info.Title = "Updated Simple API"
	doc.Info.Description = pointer.From("This API has been updated with new description")

	// Add a new server
	doc.Servers = append(doc.Servers, &openapi.Server{
		URL:         "https://api.updated.com/v1",
		Description: pointer.From("Updated server"),
	})

	buf := bytes.NewBuffer([]byte{})

	// Marshal the updated document
	if err := openapi.Marshal(ctx, doc, buf); err != nil {
		panic(err)
	}

	fmt.Println("Updated document:")
	fmt.Println(buf.String())
	// Output: Updated document:
	// openapi: 3.1.1
	// info:
	//   title: Updated Simple API
	//   description: This API has been updated with new description
	//   version: 1.0.0
	// servers:
	//   - url: https://api.example.com/v1
	//     description: Main server
	//   - url: https://api.updated.com/v1
	//     description: Updated server
	// paths:
	//   /users:
	//     get:
	//       operationId: getUsers
	//       summary: Get all users
	//       responses:
	//         "200":
	//           description: List of users
}

// Example_walking demonstrates how to traverse an OpenAPI document using the iterator API.
// Shows how to match different types of objects and terminate the walk early.
func Example_walking() {
	ctx := context.Background()

	f, err := os.Open("testdata/test.openapi.yaml")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	doc, _, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	operationCount := 0

	// Walk through the document using the iterator API
	for item := range openapi.Walk(ctx, doc) {
		// Use the matcher to handle different types of objects
		err := item.Match(openapi.Matcher{
			OpenAPI: func(o *openapi.OpenAPI) error {
				fmt.Printf("Found OpenAPI document: %s\n", o.Info.Title)
				return nil
			},
			Info: func(info *openapi.Info) error {
				fmt.Printf("Found Info: %s (version %s)\n", info.Title, info.Version)
				return nil
			},
			Operation: func(op *openapi.Operation) error {
				if op.OperationID != nil {
					fmt.Printf("Found Operation: %s\n", *op.OperationID)
				}
				operationCount++
				// Terminate after finding 3 operations
				if operationCount >= 3 {
					return walk.ErrTerminate
				}
				return nil
			},
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				if schema.IsSchema() && schema.GetSchema().Type != nil {
					types := schema.GetSchema().GetType()
					if len(types) > 0 {
						fmt.Printf("Found Schema of type: %s\n", types[0])
					}
				}
				return nil
			},
		})
		if err != nil {
			if errors.Is(err, walk.ErrTerminate) {
				fmt.Println("Walk terminated early")
				break
			}
			fmt.Printf("Error during walk: %s\n", err.Error())
			break
		}
	}
	// Output: Found OpenAPI document: Test OpenAPI Document
	// Found Info: Test OpenAPI Document (version 1.0.0)
	// Found Schema of type: string
	// Found Operation: test
	// Found Operation: copyTest
	// Found Schema of type: integer
	// Found Operation: updateUser
	// Walk terminated early
}

// Example_resolvingAllReferences demonstrates how to resolve all references in an OpenAPI document
// in a single operation, which is convenient as you can then use MustGetObject() and expect them to be resolved already.
func Example_resolvingAllReferences() {
	ctx := context.Background()

	absPath, err := filepath.Abs("testdata/resolve_test/main.yaml")
	if err != nil {
		panic(err)
	}

	f, err := os.Open(absPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Unmarshal the document
	doc, validationErrs, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	if len(validationErrs) > 0 {
		for _, err := range validationErrs {
			fmt.Printf("Validation error: %s\n", err.Error())
		}
	}

	// Resolve all references in the document
	resolveValidationErrs, resolveErrs := doc.ResolveAllReferences(ctx, openapi.ResolveAllOptions{
		OpenAPILocation: absPath,
	})

	if resolveErrs != nil {
		fmt.Printf("Resolution error: %s\n", resolveErrs.Error())
		return
	}

	if len(resolveValidationErrs) > 0 {
		for _, err := range resolveValidationErrs {
			fmt.Printf("Resolution validation error: %s\n", err.Error())
		}
	}

	// Now all references are resolved and can be accessed directly
	if doc.Paths != nil {
		for path, pathItem := range doc.Paths.All() {
			if pathItem.IsReference() && pathItem.IsResolved() {
				fmt.Printf("Path %s is a resolved reference\n", path)
			}
		}
	}

	fmt.Println("All references resolved successfully!")
	// Output: All references resolved successfully!
}

// Example_resolvingReferencesAsYouGo demonstrates how to resolve references individually
// as you encounter them during document traversal using the model API instead of the walk API.
func Example_resolvingReferencesAsYouGo() {
	ctx := context.Background()

	absPath, err := filepath.Abs("testdata/resolve_test/main.yaml")
	if err != nil {
		panic(err)
	}

	f, err := os.Open(absPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Unmarshal the document
	doc, _, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	resolveOpts := openapi.ResolveOptions{
		TargetLocation: absPath,
		RootDocument:   doc,
	}

	// Walk through the document using the model API and resolve references as we encounter them
	if doc.Paths != nil {
		for path, pathItem := range doc.Paths.All() {
			fmt.Printf("Processing path: %s\n", path)

			if pathItem.IsReference() && !pathItem.IsResolved() {
				fmt.Printf("  Resolving path item reference: %s\n", pathItem.GetReference())
				_, err := pathItem.Resolve(ctx, resolveOpts)
				if err != nil {
					fmt.Printf("  Failed to resolve path item: %v\n", err)
					continue
				}
			}

			// Get the resolved path item
			pathItemObj := pathItem.GetObject()
			if pathItemObj == nil {
				continue
			}

			// Check parameters
			for i, param := range pathItemObj.Parameters {
				if param.IsReference() && !param.IsResolved() {
					fmt.Printf("  Resolving parameter reference [%d]: %s\n", i, param.GetReference())
					_, err := param.Resolve(ctx, resolveOpts)
					if err != nil {
						fmt.Printf("  Failed to resolve parameter: %v\n", err)
						continue
					}
					if paramObj := param.GetObject(); paramObj != nil {
						fmt.Printf("  Parameter resolved: %s\n", paramObj.Name)
					}
				}
			}

			// Check operations
			for method, operation := range pathItemObj.All() {
				fmt.Printf("  Processing operation: %s\n", method)

				// Check operation parameters
				for i, param := range operation.Parameters {
					if param.IsReference() && !param.IsResolved() {
						fmt.Printf("    Resolving operation parameter reference [%d]: %s\n", i, param.GetReference())
						_, err := param.Resolve(ctx, resolveOpts)
						if err != nil {
							fmt.Printf("    Failed to resolve parameter: %v\n", err)
							continue
						}
						if paramObj := param.GetObject(); paramObj != nil {
							fmt.Printf("    Parameter resolved: %s\n", paramObj.Name)
						}
					}
				}

				// Check responses
				for statusCode, response := range operation.Responses.All() {
					if response.IsReference() && !response.IsResolved() {
						fmt.Printf("    Resolving response reference [%s]: %s\n", statusCode, response.GetReference())
						_, err := response.Resolve(ctx, resolveOpts)
						if err != nil {
							fmt.Printf("    Failed to resolve response: %v\n", err)
							continue
						}
						if respObj := response.GetObject(); respObj != nil {
							fmt.Printf("    Response resolved: %s\n", respObj.Description)
						}
					}
				}
			}
		}
	}

	fmt.Println("References resolved as encountered!")
	// Output: Processing path: /users/{userId}
	//   Processing operation: get
	//     Resolving operation parameter reference [0]: #/components/parameters/testParamRef
	//     Parameter resolved: userId
	//     Resolving response reference [200]: #/components/responses/testResponseRef
	//     Response resolved: User response
	// Processing path: /users
	//   Processing operation: post
	// References resolved as encountered!
}

// Example_creating demonstrates how to create an OpenAPI document from scratch.
// Shows building a complete document with paths, operations, and responses programmatically.
func Example_creating() {
	ctx := context.Background()

	// Create a new OpenAPI document
	paths := openapi.NewPaths()

	// Create a path item with a GET operation
	pathItem := openapi.NewPathItem()
	pathItem.Set(openapi.HTTPMethodGet, &openapi.Operation{
		OperationID: pointer.From("getUsers"),
		Summary:     pointer.From("Get all users"),
		Responses:   openapi.NewResponses(),
	})

	// Add a 200 response
	response200 := &openapi.ReferencedResponse{
		Object: &openapi.Response{
			Description: "Successful response",
		},
	}
	pathItem.Get().Responses.Set("200", response200)

	// Add the path item to paths
	referencedPathItem := &openapi.ReferencedPathItem{
		Object: pathItem,
	}
	paths.Set("/users", referencedPathItem)

	doc := &openapi.OpenAPI{
		OpenAPI: openapi.Version,
		Info: openapi.Info{
			Title:       "My API",
			Description: pointer.From("A sample API created programmatically"),
			Version:     "1.0.0",
		},
		Servers: []*openapi.Server{
			{
				URL:         "https://api.example.com/v1",
				Description: pointer.From("Production server"),
				Name:        pointer.From("prod"),
			},
		},
		Paths: paths,
	}

	buf := bytes.NewBuffer([]byte{})

	err := openapi.Marshal(ctx, doc, buf)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", buf.String())
	// Output: openapi: 3.2.0
	// info:
	//   title: My API
	//   version: 1.0.0
	//   description: A sample API created programmatically
	// servers:
	//   - url: https://api.example.com/v1
	//     description: Production server
	//     name: prod
	// paths:
	//   /users:
	//     get:
	//       operationId: getUsers
	//       summary: Get all users
	//       responses:
	//         "200":
	//           description: Successful response
}

// Example_workingWithComponents demonstrates how to work with reusable components
// in an OpenAPI document, including schemas, parameters, responses, etc.
func Example_workingWithComponents() {
	ctx := context.Background()

	// Create schema components
	schemas := sequencedmap.New(
		sequencedmap.NewElem("User", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type: oas3.NewTypeFromString(oas3.SchemaTypeObject),
			Properties: sequencedmap.New(
				sequencedmap.NewElem("id", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type: oas3.NewTypeFromString(oas3.SchemaTypeInteger),
				})),
				sequencedmap.NewElem("name", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type: oas3.NewTypeFromString(oas3.SchemaTypeString),
				})),
			),
			Required: []string{"id", "name"},
		})),
	)

	// Create parameter components
	parameters := sequencedmap.New(
		sequencedmap.NewElem("UserIdParam", &openapi.ReferencedParameter{
			Object: &openapi.Parameter{
				Name:     "userId",
				In:       "path",
				Required: pointer.From(true),
				Schema: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type: oas3.NewTypeFromString(oas3.SchemaTypeInteger),
				}),
			},
		}),
	)

	// Create paths that use the components
	paths := openapi.NewPaths()
	pathItem := openapi.NewPathItem()

	// Add parameter reference
	ref := references.Reference("#/components/parameters/UserIdParam")
	pathItem.Parameters = []*openapi.ReferencedParameter{
		{
			Reference: &ref,
		},
	}

	// Add GET operation
	pathItem.Set(openapi.HTTPMethodGet, &openapi.Operation{
		OperationID: pointer.From("getUser"),
		Responses:   openapi.NewResponses(),
	})

	// Add response with schema reference
	response200 := &openapi.ReferencedResponse{
		Object: &openapi.Response{
			Description: "User details",
			Content: sequencedmap.New(
				sequencedmap.NewElem("application/json", &openapi.MediaType{
					Schema: oas3.NewJSONSchemaFromReference("#/components/schemas/User"),
				}),
			),
		},
	}
	pathItem.Get().Responses.Set("200", response200)

	paths.Set("/users/{userId}", &openapi.ReferencedPathItem{
		Object: pathItem,
	})

	// Create the OpenAPI document with components
	doc := &openapi.OpenAPI{
		OpenAPI: openapi.Version,
		Info: openapi.Info{
			Title:   "API with Components",
			Version: "1.0.0",
		},
		Components: &openapi.Components{
			Schemas:    schemas,
			Parameters: parameters,
		},
		Paths: paths,
	}

	// Access components
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas.All() {
			fmt.Printf("Found schema component: %s\n", name)
			if schema.IsSchema() && schema.GetSchema().Type != nil {
				types := schema.GetSchema().GetType()
				if len(types) > 0 {
					fmt.Printf("  Type: %s\n", types[0])
				}
			}
		}
	}

	buf := bytes.NewBuffer([]byte{})
	if err := openapi.Marshal(ctx, doc, buf); err != nil {
		panic(err)
	}

	fmt.Printf("Document with components:\n%s", buf.String())
	// Output: Found schema component: User
	//   Type: object
	// Document with components:
	// openapi: 3.2.0
	// info:
	//   title: API with Components
	//   version: 1.0.0
	// paths:
	//   /users/{userId}:
	//     get:
	//       operationId: getUser
	//       responses:
	//         "200":
	//           description: User details
	//           content:
	//             application/json:
	//               schema:
	//                 $ref: '#/components/schemas/User'
	//     parameters:
	//       - $ref: '#/components/parameters/UserIdParam'
	// components:
	//   schemas:
	//     User:
	//       type: object
	//       properties:
	//         id:
	//           type: integer
	//         name:
	//           type: string
	//       required:
	//         - id
	//         - name
	//   parameters:
	//     UserIdParam:
	//       name: userId
	//       in: path
	//       required: true
	//       schema:
	//         type: integer
}

// Example_inliningSchema demonstrates how to inline all references in a JSON Schema
// creating a self-contained schema that doesn't depend on external definitions.
func Example_inliningSchema() {
	ctx := context.Background()

	// JSON Schema with references that will be inlined
	schemaJSON := `{
  "type": "object",
  "properties": {
    "user": {"$ref": "#/$defs/User"},
    "users": {
      "type": "array",
      "items": {"$ref": "#/$defs/User"}
    }
  },
  "$defs": {
    "User": {
      "type": "object",
      "properties": {
        "id": {"type": "integer"},
        "name": {"type": "string"},
        "address": {"$ref": "#/$defs/Address"}
      },
      "required": ["id", "name"]
    },
    "Address": {
      "type": "object",
      "properties": {
        "street": {"type": "string"},
        "city": {"type": "string"}
      },
      "required": ["street", "city"]
    }
  }
}`

	// Unmarshal the JSON Schema
	var schema oas3.JSONSchema[oas3.Referenceable]
	validationErrs, err := marshaller.Unmarshal(ctx, bytes.NewReader([]byte(schemaJSON)), &schema)
	if err != nil {
		panic(err)
	}
	if len(validationErrs) > 0 {
		for _, err := range validationErrs {
			fmt.Printf("Validation error: %s\n", err.Error())
		}
	}

	// Configure inlining options
	opts := oas3.InlineOptions{
		ResolveOptions: oas3.ResolveOptions{
			TargetLocation: "schema.json",
			RootDocument:   &schema,
		},
		RemoveUnusedDefs: true, // Clean up unused definitions after inlining
	}

	// Inline all references
	inlinedSchema, err := oas3.Inline(ctx, &schema, opts)
	if err != nil {
		panic(err)
	}

	fmt.Println("After inlining:")
	buf := bytes.NewBuffer([]byte{})
	ctx = yml.ContextWithConfig(ctx, schema.GetCore().Config) // Use the same config as the original schema
	if err := marshaller.Marshal(ctx, inlinedSchema, buf); err != nil {
		panic(err)
	}
	fmt.Printf("%s", buf.String())
	// Output: After inlining:
	// {
	//   "type": "object",
	//   "properties": {
	//     "user": {
	//       "type": "object",
	//       "properties": {
	//         "id": {
	//           "type": "integer"
	//         },
	//         "name": {
	//           "type": "string"
	//         },
	//         "address": {
	//           "type": "object",
	//           "properties": {
	//             "street": {
	//               "type": "string"
	//             },
	//             "city": {
	//               "type": "string"
	//             }
	//           },
	//           "required": [
	//             "street",
	//             "city"
	//           ]
	//         }
	//       },
	//       "required": [
	//         "id",
	//         "name"
	//       ]
	//     },
	//     "users": {
	//       "type": "array",
	//       "items": {
	//         "type": "object",
	//         "properties": {
	//           "id": {
	//             "type": "integer"
	//           },
	//           "name": {
	//             "type": "string"
	//           },
	//           "address": {
	//             "type": "object",
	//             "properties": {
	//               "street": {
	//                 "type": "string"
	//               },
	//               "city": {
	//                 "type": "string"
	//               }
	//             },
	//             "required": [
	//               "street",
	//               "city"
	//             ]
	//           }
	//         },
	//         "required": [
	//           "id",
	//           "name"
	//         ]
	//       }
	//     }
	//   }
	// }
}

// Example_upgrading demonstrates how to upgrade an OpenAPI document from 3.0.x to 3.2.0.
// Shows the automatic conversion of nullable fields, examples, and other version differences.
func Example_upgrading() {
	ctx := context.Background()

	// OpenAPI 3.0.3 document with features that need upgrading
	openAPIYAML := `openapi: 3.0.3
info:
  title: Legacy API
  version: 1.0.0
  description: An API that needs upgrading from 3.0.3 to 3.2.0
paths:
  /users:
    get:
      summary: Get users
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
          nullable: true
          example: "John Doe"
        email:
          type: string
          format: email
          exclusiveMaximum: true
          maximum: 100
      required:
        - id`

	// Unmarshal the OpenAPI document
	doc, _, err := openapi.Unmarshal(ctx, bytes.NewReader([]byte(openAPIYAML)))
	if err != nil {
		panic(err)
	}

	// Upgrade the document to the latest version
	upgraded, err := openapi.Upgrade(ctx, doc)
	if err != nil {
		panic(err)
	}
	if !upgraded {
		panic("upgrade should have been performed")
	}

	fmt.Printf("Upgraded OpenAPI Version: %s\n", doc.OpenAPI)

	// Marshal the upgraded document
	fmt.Println("\nAfter upgrade:")
	buf := bytes.NewBuffer([]byte{})
	if err := openapi.Marshal(ctx, doc, buf); err != nil {
		panic(err)
	}
	fmt.Printf("%s", buf.String())
	// Output: Upgraded OpenAPI Version: 3.2.0
	//
	// After upgrade:
	// openapi: 3.2.0
	// info:
	//   title: Legacy API
	//   version: 1.0.0
	//   description: An API that needs upgrading from 3.0.3 to 3.2.0
	// paths:
	//   /users:
	//     get:
	//       summary: Get users
	//       responses:
	//         '200':
	//           description: Success
	//           content:
	//             application/json:
	//               schema:
	//                 $ref: '#/components/schemas/User'
	// components:
	//   schemas:
	//     User:
	//       type: object
	//       properties:
	//         id:
	//           type: integer
	//         name:
	//           type:
	//             - string
	//             - "null"
	//           examples:
	//             - "John Doe"
	//         email:
	//           type: string
	//           format: email
	//           exclusiveMaximum: 100
	//       required:
	//         - id
}
