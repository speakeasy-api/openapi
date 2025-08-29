package openapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/hashing"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// InlineOptions represents the options available when inlining an OpenAPI document.
type InlineOptions struct {
	// ResolveOptions are the options to use when resolving references during inlining.
	ResolveOptions ResolveOptions
	// RemoveUnusedComponents determines whether to remove components that are no longer referenced after inlining.
	RemoveUnusedComponents bool
}

// Inline transforms an OpenAPI document by replacing all $ref references with their actual content,
// creating a self-contained document that doesn't depend on external definitions or component references.
// This operation modifies the document in place.
//
// Why use Inline?
//
//   - **Simplify document distribution**: Create standalone OpenAPI documents that can be shared without worrying
//     about missing referenced files or component definitions
//   - **AI and tooling integration**: Provide complete, self-contained OpenAPI documents to AI systems and
//     tools that work better with fully expanded specifications
//   - **Improve compatibility**: Some tools work better with fully expanded documents rather
//     than ones with references
//   - **Generate documentation**: Create complete API representations for documentation
//     where all schemas and components are visible inline
//   - **Optimize for specific use cases**: Eliminate the need for reference resolution in
//     performance-critical applications
//   - **Debug API issues**: See the full expanded document to understand how references resolve
//
// What you'll get:
//
// Before inlining:
//
//	{
//	  "openapi": "3.1.0",
//	  "paths": {
//	    "/users": {
//	      "get": {
//	        "responses": {
//	          "200": {"$ref": "#/components/responses/UserResponse"}
//	        }
//	      }
//	    }
//	  },
//	  "components": {
//	    "responses": {
//	      "UserResponse": {
//	        "description": "User response",
//	        "content": {
//	          "application/json": {
//	            "schema": {"$ref": "#/components/schemas/User"}
//	          }
//	        }
//	      }
//	    },
//	    "schemas": {
//	      "User": {"type": "object", "properties": {"name": {"type": "string"}}}
//	    }
//	  }
//	}
//
// After inlining:
//
//	{
//	  "openapi": "3.1.0",
//	  "paths": {
//	    "/users": {
//	      "get": {
//	        "responses": {
//	          "200": {
//	            "description": "User response",
//	            "content": {
//	              "application/json": {
//	                "schema": {"type": "object", "properties": {"name": {"type": "string"}}}
//	              }
//	            }
//	          }
//	        }
//	      }
//	    }
//	  }
//	}
//
// Handling References:
//
// Unlike JSON Schema references, OpenAPI component references are simpler to handle since they don't
// typically have circular reference issues. The function will:
//
// 1. Walk through the entire OpenAPI document
// 2. For each reference encountered, resolve and inline it in place
// 3. For JSON schemas, use the existing oas3.Inline functionality
// 4. Optionally remove unused components after inlining
//
// Example usage:
//
//	// Load an OpenAPI document with references
//	doc := &OpenAPI{...}
//
//	// Configure inlining
//	opts := InlineOptions{
//		ResolveOptions: ResolveOptions{
//			RootDocument: doc,
//			TargetLocation: "openapi.yaml",
//		},
//		RemoveUnusedComponents: true, // Clean up unused components
//	}
//
//	// Inline all references (modifies doc in place)
//	err := Inline(ctx, doc, opts)
//	if err != nil {
//		return fmt.Errorf("failed to inline document: %w", err)
//	}
//
//	// doc is now a self-contained OpenAPI document with all references expanded
//
// Parameters:
//   - ctx: Context for the operation
//   - doc: The OpenAPI document to inline (modified in place)
//   - opts: Configuration options for inlining
//
// Returns:
//   - error: Any error that occurred during inlining
func Inline(ctx context.Context, doc *OpenAPI, opts InlineOptions) error {
	if doc == nil {
		return nil
	}

	// Track collected $defs to avoid duplication
	collectedDefs := sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
	defHashes := make(map[string]string) // name -> hash

	if err := inlineObject(ctx, doc, doc, opts.ResolveOptions, collectedDefs, defHashes); err != nil {
		return err
	}

	// Remove unused components if requested
	if opts.RemoveUnusedComponents {
		removeUnusedComponents(doc, collectedDefs)
	}

	return nil
}

func inlineObject[T any](ctx context.Context, obj *T, doc *OpenAPI, opts ResolveOptions, collectedDefs *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]], defHashes map[string]string) error {
	inlinedSchemas := map[*oas3.JSONSchema[oas3.Referenceable]]*oas3.JSONSchema[oas3.Referenceable]{}

	for item := range Walk(ctx, obj) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				location := item.Location.ToJSONPointer().String()

				// Only skip top-level component schema definitions (e.g., /components/schemas/User)
				// but allow inlining of schemas that reference them (e.g., /components/responses/UserResponse/content/application~1json/schema)
				if strings.HasPrefix(location, "/components") {
					// Split the path to check if this is a top-level schema definition
					parts := strings.Split(location, "/")
					// parts[0] is empty, parts[1] is "components", parts[2] is "schemas" or some other section, parts[3] is the schema name
					// If we have exactly 4 parts, this is a top-level schema definition like /components/schemas/User
					if len(parts) == 4 {
						return nil
					}
				}

				parent := item.Location[len(item.Location)-1].ParentMatchFunc

				parentIsSchema := false
				_ = parent(Matcher{
					Schema: func(parentSchema *oas3.JSONSchema[oas3.Referenceable]) error {
						parentIsSchema = true
						return nil
					},
				})
				// If the parent is a schema, we don't need to inline it
				if parentIsSchema {
					return nil
				}

				inlineOpts := oas3.InlineOptions{
					ResolveOptions:   opts,
					RemoveUnusedDefs: true,
				}

				inlined, err := oas3.Inline(ctx, schema, inlineOpts)
				if err != nil {
					return fmt.Errorf("failed to inline schema: %w", err)
				}

				inlinedSchemas[schema] = inlined

				return nil
			},
		})
		if err != nil {
			return fmt.Errorf("failed to inline schemas: %w", err)
		}
	}

	// Walk through the document and inline all references
	for item := range Walk(ctx, obj) {
		err := item.Match(Matcher{
			// Handle JSON Schema references using the existing oas3.Inline functionality
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				inlined, exists := inlinedSchemas[schema]
				if !exists {
					return nil
				}

				// Process $defs from the inlined schema before replacement
				inlinedSchema := inlined.GetLeft()
				if inlinedSchema != nil && inlinedSchema.Defs != nil && inlinedSchema.Defs.Len() > 0 {
					// Ensure components/schemas exists
					if doc.Components == nil {
						doc.Components = &Components{}
					}
					if doc.Components.Schemas == nil {
						doc.Components.Schemas = sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
					}

					// Process each $def and build a mapping for this specific schema
					nameMapping := make(map[string]string)
					for defName, defSchema := range inlinedSchema.Defs.All() {
						targetName := defName
						defHash := hashing.Hash(defSchema)

						// Check for conflicts
						if existingHash, exists := defHashes[defName]; exists {
							if existingHash != defHash {
								// Different schema with same name - add suffix
								counter := 1
								for {
									candidateName := fmt.Sprintf("%s_%d", defName, counter)
									if existingHash, exists := defHashes[candidateName]; !exists || existingHash == defHash {
										targetName = candidateName
										break
									}
									counter++
								}
							}
						}

						// Store the mapping for this schema
						nameMapping[defName] = targetName

						// Store the schema if it's new
						if _, exists := defHashes[targetName]; !exists {
							defHashes[targetName] = defHash
							collectedDefs.Set(targetName, defSchema)
							doc.Components.Schemas.Set(targetName, defSchema)
						}
					}

					// Rewrite $refs in the inlined schema to point to components/schemas
					rewriteRefsWithMapping(inlined, nameMapping)

					// Remove $defs from the inlined schema
					inlinedSchema.Defs = nil
				}

				// Replace the schema in place
				*schema = *inlined
				return nil
			},

			// Handle OpenAPI component references
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return inlineReference(ctx, ref, doc, opts, collectedDefs, defHashes)
			},
		})

		if err != nil {
			return fmt.Errorf("failed to inline references: %w", err)
		}
	}

	return nil
}

// inlineReference inlines a generic OpenAPI reference by resolving it and replacing the reference with the actual object
func inlineReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, ref *Reference[T, V, C], doc *OpenAPI, opts references.ResolveOptions, collectedDefs *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]], defHashes map[string]string) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	// Resolve the reference
	validationErrs, err := ref.Resolve(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to resolve reference %s: %w", ref.GetReference(), err)
	}

	// Log validation errors but don't fail on them
	if len(validationErrs) > 0 {
		// In a production system, you might want to log these or handle them differently
		// For now, we'll continue with the inlining process
		_ = validationErrs // Acknowledge we're intentionally ignoring these
	}

	// Get the resolved object
	obj := ref.GetObject()
	if obj == nil {
		return fmt.Errorf("reference %s resolved to nil object", ref.GetReference())
	}

	targetDocInfo := ref.GetReferenceResolutionInfo()

	// Replace the reference with the actual object in place
	ref.Reference = nil
	ref.Object = obj
	ref.Summary = nil
	ref.Description = nil

	// Recursively inline any references within the newly inlined object
	if targetDocInfo != nil && targetDocInfo.ResolvedDocument != nil {
		recursiveOpts := ResolveOptions{
			RootDocument:   opts.RootDocument,
			TargetDocument: targetDocInfo.ResolvedDocument,
			TargetLocation: targetDocInfo.AbsoluteReference,
		}
		if err := inlineObject(ctx, ref, doc, recursiveOpts, collectedDefs, defHashes); err != nil {
			return fmt.Errorf("failed to inline nested references in %s: %w", ref.GetReference(), err)
		}
	}

	return nil
}

// rewriteRefsWithMapping uses the Walk API to rewrite $ref paths from $defs to components/schemas
// using a specific name mapping for this schema
func rewriteRefsWithMapping(schema *oas3.JSONSchema[oas3.Referenceable], nameMapping map[string]string) {
	if schema == nil {
		return
	}

	// Walk through all schemas and rewrite references
	for item := range oas3.Walk(context.Background(), schema) {
		err := item.Match(oas3.SchemaMatcher{
			Schema: func(s *oas3.JSONSchema[oas3.Referenceable]) error {
				schemaObj := s.GetLeft()
				if schemaObj != nil && schemaObj.Ref != nil {
					refStr := schemaObj.Ref.String()
					if strings.HasPrefix(refStr, "#/$defs/") {
						defName := strings.TrimPrefix(refStr, "#/$defs/")
						// Use the specific mapping for this schema
						if targetName, exists := nameMapping[defName]; exists {
							newRef := "#/components/schemas/" + targetName
							*schemaObj.Ref = references.Reference(newRef)
						}
					} else if strings.HasPrefix(refStr, "#/") {
						// Handle external file references like "#/User"
						defName := strings.TrimPrefix(refStr, "#/")
						if targetName, exists := nameMapping[defName]; exists {
							newRef := "#/components/schemas/" + targetName
							*schemaObj.Ref = references.Reference(newRef)
						}
					}
				}
				return nil
			},
		})
		if err != nil {
			// Log error but continue processing
			_ = err
		}
	}
}

// removeUnusedComponents removes components that are no longer referenced after inlining
func removeUnusedComponents(doc *OpenAPI, preserveSchemas *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]]) {
	if doc == nil || doc.Components == nil {
		return
	}

	// Find security schemes that are referenced by global security requirements
	preserveSecuritySchemes := sequencedmap.New[string, *ReferencedSecurityScheme]()
	if doc.Security != nil && doc.Components.SecuritySchemes != nil {
		for _, securityRequirement := range doc.Security {
			if securityRequirement != nil {
				for schemeName := range securityRequirement.All() {
					if scheme, exists := doc.Components.SecuritySchemes.Get(schemeName); exists {
						preserveSecuritySchemes.Set(schemeName, scheme)
					}
				}
			}
		}
	}

	// Create new components with preserved schemas and security schemes
	newComponents := &Components{}

	// Preserve schemas moved from $defs
	if preserveSchemas != nil && preserveSchemas.Len() > 0 {
		newComponents.Schemas = preserveSchemas
	}

	// Preserve security schemes referenced by global security requirements
	if preserveSecuritySchemes.Len() > 0 {
		newComponents.SecuritySchemes = preserveSecuritySchemes
	}

	// Only set components if we have something to preserve
	if newComponents.Schemas != nil || newComponents.SecuritySchemes != nil {
		doc.Components = newComponents
	} else {
		// No components to preserve, clear all components
		doc.Components = nil
	}
}
