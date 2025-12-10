<p align="center">
  <p align="center">
    <img  width="200px" alt="OpenAPI" src="https://github.com/user-attachments/assets/b9fa9c14-1c6f-4d8b-910f-15e5f962bab6">
  </p>
  <h1 align="center"><b>OpenAPI Parser</b></h1>
  <p align="center">An API for working with <a href="https://spec.openapis.org/oas/v3.2.0">OpenAPI documents</a> including: read, walk, create, mutate, validate, and upgrade
</p>
  <p align="center">
       <!-- OpenAPI Hub Badge -->
    <a href="https://www.speakeasy.com/openapi"><img alt="OpenAPI Hub" src="https://www.speakeasy.com/assets/badges/openapi-hub.svg" /></a>
   <!-- Built By Speakeasy Badge -->
    <a href="https://speakeasy.com/"><img alt="Built by Speakeasy" src="https://www.speakeasy.com/assets/badges/built-by-speakeasy.svg" /></a>
    <a href="https://github.com/speakeasy-api/openapi/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/speakeasy-api/openapi.svg?style=for-the-badge"></a>
    <a href="https://pkg.go.dev/github.com/speakeasy-api/openapi/openapi?tab=doc"><img alt="Go Doc" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge"></a>
   <br />
    <a href="https://github.com/speakeasy-api/openapi/actions/workflows/test.yaml"><img alt="GitHub Action: Test" src="https://img.shields.io/github/actions/workflow/status/speakeasy-api/openapi/test.yaml?style=for-the-badge"></a>
    <a href="https://goreportcard.com/report/github.com/speakeasy-api/openapi"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/speakeasy-api/openapi?style=for-the-badge"></a>
    <a href="/LICENSE"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge"></a>
  </p>
</p>

## Features

- **Full OpenAPI 3.0.x, 3.1.x, and 3.2.x Support**: Parse and work with OpenAPI 3.0.x, 3.1.x, and 3.2.x documents
- **Validation**: Built-in validation against the OpenAPI Specification
- **Walking**: Traverse all elements in an OpenAPI document with a powerful iterator pattern
- **Upgrading**: Automatically upgrade OpenAPI 3.0.x and 3.1.x documents to 3.2.0
- **Mutation**: Modify OpenAPI documents programmatically
- **JSON Schema Support**: Direct access to JSON Schema functionality
- **Reference Resolution**: Resolve $ref references within documents
- **Circular Reference Handling**: Proper handling of circular references in schemas
- **Extension Support**: Full support for OpenAPI extensions (x-* fields)
- **Type Safety**: Strongly typed Go structs for all OpenAPI elements

## Supported OpenAPI Versions

- OpenAPI 3.0.0 through 3.0.4
- OpenAPI 3.1.0 through 3.1.2
- OpenAPI 3.2.0

The package can automatically upgrade documents from 3.0.x and 3.1.x to 3.2.0, handling the differences in specification between versions.

<!-- START USAGE EXAMPLES -->

## Read and parse an OpenAPI document from a file

This includes validation by default and shows how to access document properties.

```go
ctx := context.Background()

r, err := os.Open("testdata/test.openapi.yaml")
if err != nil {
	panic(err)
}
defer r.Close()

doc, validationErrs, err := openapi.Unmarshal(ctx, r)
if err != nil {
	panic(err)
}

for _, err := range validationErrs {
	fmt.Println(err.Error())
}

fmt.Printf("OpenAPI Version: %s\n", doc.OpenAPI)
fmt.Printf("API Title: %s\n", doc.Info.Title)
fmt.Printf("API Version: %s\n", doc.Info.Version)
```

## Work with JSON Schema directly

Shows how to unmarshal a JSONSchema from YAML or JSON and validate it manually.

```go
ctx := context.Background()

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

additionalErrs := schema.Validate(ctx)
validationErrs = append(validationErrs, additionalErrs...)

if len(validationErrs) > 0 {
	for _, err := range validationErrs {
		fmt.Println("Validation error:", err.Error())
	}
}

if schema.IsSchema() {
	schemaObj := schema.GetSchema()
	fmt.Println("Schema Types:")
	for _, t := range schemaObj.GetType() {
		fmt.Printf("  %s\n", t)
	}
	fmt.Printf("Required Fields: %v\n", schemaObj.GetRequired())
	fmt.Printf("Number of Properties: %d\n", schemaObj.GetProperties().Len())
}
```

## Marshal an OpenAPI document to a writer

Shows creating a simple document and outputting it as YAML.

```go
ctx := context.Background()

doc := &openapi.OpenAPI{
	OpenAPI: openapi.Version,
	Info: openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	},
	Paths: openapi.NewPaths(),
}

buf := bytes.NewBuffer([]byte{})

if err := openapi.Marshal(ctx, doc, buf); err != nil {
	panic(err)
}

fmt.Printf("%s", buf.String())
```

## Marshal a JSONSchema directly

Shows creating a schema programmatically and outputting it as YAML.

```go
ctx := context.Background()

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

if err := marshaller.Marshal(ctx, schema, buf); err != nil {
	panic(err)
}

fmt.Printf("%s", buf.String())
```

## Validate an OpenAPI document and fix validation errors

Shows automatic validation during unmarshaling, fixing errors programmatically, and re-validating.

```go
ctx := context.Background()

path := "testdata/invalid.openapi.yaml"

f, err := os.Open(path)
if err != nil {
	panic(err)
}
defer f.Close()

doc, validationErrs, err := openapi.Unmarshal(ctx, f)
if err != nil {
	panic(err)
}

fmt.Printf("Initial validation errors: %d\n", len(validationErrs))
for _, err := range validationErrs {
	fmt.Printf("  %s\n", err.Error())
}

fmt.Println("\nFixing validation errors...")

doc.Info.Version = "1.0.0"
fmt.Println("  ✓ Added missing info.version")

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

newValidationErrs := doc.Validate(ctx)
validation.SortValidationErrors(newValidationErrs)

fmt.Printf("\nValidation errors after fixes: %d\n", len(newValidationErrs))
for _, err := range newValidationErrs {
	fmt.Printf("  %s\n", err.Error())
}

fmt.Printf("\nReduced validation errors from %d to %d\n", len(validationErrs), len(newValidationErrs))
```

## Read and modify an OpenAPI document

Shows loading a document, making changes, and marshaling it back to YAML.

```go
ctx := context.Background()

r, err := os.Open("testdata/simple.openapi.yaml")
if err != nil {
	panic(err)
}
defer r.Close()

doc, validationErrs, err := openapi.Unmarshal(ctx, r)
if err != nil {
	panic(err)
}

for _, err := range validationErrs {
	fmt.Println(err.Error())
}

doc.Info.Title = "Updated Simple API"
doc.Info.Description = pointer.From("This API has been updated with new description")

doc.Servers = append(doc.Servers, &openapi.Server{
	URL:         "https://api.updated.com/v1",
	Description: pointer.From("Updated server"),
})

buf := bytes.NewBuffer([]byte{})

if err := openapi.Marshal(ctx, doc, buf); err != nil {
	panic(err)
}

fmt.Println("Updated document:")
fmt.Println(buf.String())
```

## Traverse an OpenAPI document using the iterator API

Shows how to match different types of objects and terminate the walk early.

```go
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

for item := range openapi.Walk(ctx, doc) {

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
```

## Resolve all references in an OpenAPI document

in a single operation, which is convenient as you can then use MustGetObject() and expect them to be resolved already.

```go
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

doc, validationErrs, err := openapi.Unmarshal(ctx, f)
if err != nil {
	panic(err)
}

if len(validationErrs) > 0 {
	for _, err := range validationErrs {
		fmt.Printf("Validation error: %s\n", err.Error())
	}
}

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

if doc.Paths != nil {
	for path, pathItem := range doc.Paths.All() {
		if pathItem.IsReference() && pathItem.IsResolved() {
			fmt.Printf("Path %s is a resolved reference\n", path)
		}
	}
}

fmt.Println("All references resolved successfully!")
```

## Resolve references individually

as you encounter them during document traversal using the model API instead of the walk API.

```go
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

doc, _, err := openapi.Unmarshal(ctx, f)
if err != nil {
	panic(err)
}

resolveOpts := openapi.ResolveOptions{
	TargetLocation: absPath,
	RootDocument:   doc,
}

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

		pathItemObj := pathItem.GetObject()
		if pathItemObj == nil {
			continue
		}

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

		for method, operation := range pathItemObj.All() {
			fmt.Printf("  Processing operation: %s\n", method)

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
```

## Create an OpenAPI document from scratch

Shows building a complete document with paths, operations, and responses programmatically.

```go
ctx := context.Background()

paths := openapi.NewPaths()

pathItem := openapi.NewPathItem()
pathItem.Set(openapi.HTTPMethodGet, &openapi.Operation{
	OperationID: pointer.From("getUsers"),
	Summary:     pointer.From("Get all users"),
	Responses:   openapi.NewResponses(),
})

response200 := &openapi.ReferencedResponse{
	Object: &openapi.Response{
		Description: "Successful response",
	},
}
pathItem.Get().Responses.Set("200", response200)

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
```

## Work with reusable components

in an OpenAPI document, including schemas, parameters, responses, etc.

```go
ctx := context.Background()

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

paths := openapi.NewPaths()
pathItem := openapi.NewPathItem()

ref := references.Reference("#/components/parameters/UserIdParam")
pathItem.Parameters = []*openapi.ReferencedParameter{
	{
		Reference: &ref,
	},
}

pathItem.Set(openapi.HTTPMethodGet, &openapi.Operation{
	OperationID: pointer.From("getUser"),
	Responses:   openapi.NewResponses(),
})

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
```

## Inline all references in a JSON Schema

creating a self-contained schema that doesn't depend on external definitions.

```go
ctx := context.Background()

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

opts := oas3.InlineOptions{
	ResolveOptions: oas3.ResolveOptions{
		TargetLocation: "schema.json",
		RootDocument:   &schema,
	},
	RemoveUnusedDefs: true,
}

inlinedSchema, err := oas3.Inline(ctx, &schema, opts)
if err != nil {
	panic(err)
}

fmt.Println("After inlining:")
buf := bytes.NewBuffer([]byte{})
ctx = yml.ContextWithConfig(ctx, schema.GetCore().Config)
if err := marshaller.Marshal(ctx, inlinedSchema, buf); err != nil {
	panic(err)
}
fmt.Printf("%s", buf.String())
```

## Upgrade an OpenAPI document from 3.0.x to 3.2.0

Shows the automatic conversion of nullable fields, examples, and other version differences.

```go
ctx := context.Background()

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

doc, _, err := openapi.Unmarshal(ctx, bytes.NewReader([]byte(openAPIYAML)))
if err != nil {
	panic(err)
}

upgraded, err := openapi.Upgrade(ctx, doc)
if err != nil {
	panic(err)
}
if !upgraded {
	panic("upgrade should have been performed")
}

fmt.Printf("Upgraded OpenAPI Version: %s\n", doc.OpenAPI)

fmt.Println("\nAfter upgrade:")
buf := bytes.NewBuffer([]byte{})
if err := openapi.Marshal(ctx, doc, buf); err != nil {
	panic(err)
}
fmt.Printf("%s", buf.String())
```

<!-- END USAGE EXAMPLES -->

## Contributing

This repository is maintained by Speakeasy, but we welcome and encourage contributions from the community to help improve its capabilities and stability.

### How to Contribute

1. **Open Issues**: Found a bug or have a feature suggestion? Open an issue to describe what you'd like to see changed.

2. **Pull Requests**: We welcome pull requests! If you'd like to contribute code:
   - Fork the repository
   - Create a new branch for your feature/fix
   - Submit a PR with a clear description of the changes and any related issues

3. **Feedback**: Share your experience using the packages or suggest improvements.

All contributions, whether they're bug reports, feature requests, or code changes, help make this project better for everyone.

Please ensure your contributions adhere to our coding standards and include appropriate tests where applicable.
