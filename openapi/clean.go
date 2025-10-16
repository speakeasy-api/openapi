package openapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// Clean removes unused, unreachable elements from the OpenAPI document using reachability from paths and security.
//
// How it works (high level):
//
//   - Seed reachability only from:
//   - Operations under /paths (responses, request bodies, parameters, schemas, etc.)
//   - Security requirements (top-level and operation-level), referenced by $ref and by name
//   - Expand reachability only through components already marked as used until a fixed point
//   - Remove anything not reachable from those seeds
//   - Also remove top-level tags that are not referenced by any operation
//
// This function modifies the document in place.
//
// Why use Clean?
//
//   - **Reduce document size**: Remove unused component definitions and tags that bloat the specification
//   - **Improve clarity**: Keep only the elements that are actually used by operations/security
//   - **Optimize tooling performance**: Smaller documents with fewer unused elements process faster
//   - **Maintain clean specifications**: Prevent accumulation of dead code in API definitions
//   - **Prepare for distribution**: Clean up specifications before sharing or publishing
//
// What gets cleaned:
//
//   - Unused schemas in components/schemas
//   - Unused responses in components/responses
//   - Unused parameters in components/parameters
//   - Unused examples in components/examples
//   - Unused request bodies in components/requestBodies
//   - Unused headers in components/headers
//   - Unused security schemes in components/securitySchemes (with special handling)
//   - Unused links in components/links
//   - Unused callbacks in components/callbacks
//   - Unused path items in components/pathItems
//   - Unused tags in the top-level tags array
//
// Special handling for security schemes:
//
// Security schemes can be referenced in two ways:
//  1. By $ref (like other components)
//  2. By name in security requirement objects (global or operation-level)
//
// The Clean function handles both cases correctly.
//
// Example usage:
//
//	// Load an OpenAPI document and prune unused elements (modifies doc in place)
//	err := Clean(ctx, doc)
//	if err != nil {
//		return fmt.Errorf("failed to clean document: %w", err)
//	}
//
//	// doc now contains only elements reachable from /paths and security, with unused tags removed
//
// Parameters:
//   - ctx: Context for the operation
//   - doc: The OpenAPI document to clean (modified in place)
//
// Returns:
//   - error: Any error that occurred during cleaning
func Clean(ctx context.Context, doc *OpenAPI) error {
	if doc == nil {
		return nil
	}

	// Track referenced components by type and name
	referenced := &referencedComponentTracker{
		schemas:         make(map[string]bool),
		responses:       make(map[string]bool),
		parameters:      make(map[string]bool),
		examples:        make(map[string]bool),
		requestBodies:   make(map[string]bool),
		headers:         make(map[string]bool),
		securitySchemes: make(map[string]bool),
		links:           make(map[string]bool),
		callbacks:       make(map[string]bool),
		pathItems:       make(map[string]bool),
	}

	// Phase 1: Seed references only from within /paths
	err := walkAndTrackWithFilter(ctx, doc, referenced, func(ptr string) bool {
		// Only allow references originating under paths
		return strings.HasPrefix(ptr, "/paths/") || strings.HasPrefix(ptr, "/security")
	})
	if err != nil {
		return fmt.Errorf("failed to track references from paths: %w", err)
	}

	// Phase 2: Expand closure of references reachable from used components.
	// We repeatedly walk the document but only allow visiting content under components
	// that are already marked as referenced, until no new references are discovered.
	for {
		before := countTracked(referenced)

		err := walkAndTrackWithFilter(ctx, doc, referenced, func(ptr string) bool {
			typ, name, ok := extractComponentTypeAndName(ptr)
			if !ok {
				return false
			}
			switch typ {
			case "schemas":
				return referenced.schemas[name]
			case "responses":
				return referenced.responses[name]
			case "parameters":
				return referenced.parameters[name]
			case "examples":
				return referenced.examples[name]
			case "requestBodies":
				return referenced.requestBodies[name]
			case "headers":
				return referenced.headers[name]
			case "securitySchemes":
				return referenced.securitySchemes[name]
			case "links":
				return referenced.links[name]
			case "callbacks":
				return referenced.callbacks[name]
			case "pathItems":
				return referenced.pathItems[name]
			default:
				return false
			}
		})
		if err != nil {
			return fmt.Errorf("failed to expand reachable references: %w", err)
		}

		after := countTracked(referenced)
		if after == before {
			break // fixed point reached
		}
	}

	// Remove unused components
	removeUnusedComponentsFromDocument(doc, referenced)

	// Remove unused top-level tags
	removeUnusedTagsFromDocument(doc, referenced)

	return nil
}

// referencedComponentTracker tracks which components are referenced
type referencedComponentTracker struct {
	schemas         map[string]bool
	responses       map[string]bool
	parameters      map[string]bool
	examples        map[string]bool
	requestBodies   map[string]bool
	headers         map[string]bool
	securitySchemes map[string]bool
	links           map[string]bool
	callbacks       map[string]bool
	pathItems       map[string]bool
	// tags used by operations (referenced by name)
	tags map[string]bool
}

// trackSchemaReferences tracks references within JSON schemas
func trackSchemaReferences(schema *oas3.JSONSchema[oas3.Referenceable], tracker *referencedComponentTracker) error {
	if schema == nil {
		return nil
	}

	// Walk through the schema to find all references
	for item := range oas3.Walk(context.Background(), schema) {
		err := item.Match(oas3.SchemaMatcher{
			Schema: func(s *oas3.JSONSchema[oas3.Referenceable]) error {
				schemaObj := s.GetLeft()
				if schemaObj != nil && schemaObj.Ref != nil {
					refStr := schemaObj.Ref.String()
					componentName := extractComponentName(refStr, "schemas")
					if componentName != "" {
						tracker.schemas[componentName] = true
					}
				}
				return nil
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// trackPathItemReference tracks a reference to a path item component
func trackPathItemReference(ref *ReferencedPathItem, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "pathItems")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// trackParameterReference tracks a reference to a parameter component
func trackParameterReference(ref *ReferencedParameter, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "parameters")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// trackExampleReference tracks a reference to an example component
func trackExampleReference(ref *ReferencedExample, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "examples")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// trackRequestBodyReference tracks a reference to a request body component
func trackRequestBodyReference(ref *ReferencedRequestBody, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "requestBodies")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// trackResponseReference tracks a reference to a response component
func trackResponseReference(ref *ReferencedResponse, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "responses")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// trackHeaderReference tracks a reference to a header component
func trackHeaderReference(ref *ReferencedHeader, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "headers")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// trackCallbackReference tracks a reference to a callback component
func trackCallbackReference(ref *ReferencedCallback, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "callbacks")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// trackLinkReference tracks a reference to a link component
func trackLinkReference(ref *ReferencedLink, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "links")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// trackSecuritySchemeReference tracks a reference to a security scheme component
func trackSecuritySchemeReference(ref *ReferencedSecurityScheme, tracker map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}
	refStr := ref.GetReference().String()
	componentName := extractComponentName(refStr, "securitySchemes")
	if componentName != "" {
		tracker[componentName] = true
	}
	return nil
}

// extractComponentName extracts the component name from a reference string
func extractComponentName(refStr, componentType string) string {
	prefix := "#/components/" + componentType + "/"
	if strings.HasPrefix(refStr, prefix) {
		return strings.TrimPrefix(refStr, prefix)
	}
	return ""
}

// removeUnusedComponentsFromDocument removes unused components from the document
func removeUnusedComponentsFromDocument(doc *OpenAPI, tracker *referencedComponentTracker) {
	if doc.Components == nil {
		return
	}

	// Remove unused schemas
	if doc.Components.Schemas != nil {
		newSchemas := sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
		for name, schema := range doc.Components.Schemas.All() {
			if tracker.schemas[name] {
				newSchemas.Set(name, schema)
			}
		}
		if newSchemas.Len() > 0 {
			doc.Components.Schemas = newSchemas
		} else {
			doc.Components.Schemas = nil
		}
	}

	// Remove unused responses
	if doc.Components.Responses != nil {
		newResponses := sequencedmap.New[string, *ReferencedResponse]()
		for name, response := range doc.Components.Responses.All() {
			if tracker.responses[name] {
				newResponses.Set(name, response)
			}
		}
		if newResponses.Len() > 0 {
			doc.Components.Responses = newResponses
		} else {
			doc.Components.Responses = nil
		}
	}

	// Remove unused parameters
	if doc.Components.Parameters != nil {
		newParameters := sequencedmap.New[string, *ReferencedParameter]()
		for name, parameter := range doc.Components.Parameters.All() {
			if tracker.parameters[name] {
				newParameters.Set(name, parameter)
			}
		}
		if newParameters.Len() > 0 {
			doc.Components.Parameters = newParameters
		} else {
			doc.Components.Parameters = nil
		}
	}

	// Remove unused examples
	if doc.Components.Examples != nil {
		newExamples := sequencedmap.New[string, *ReferencedExample]()
		for name, example := range doc.Components.Examples.All() {
			if tracker.examples[name] {
				newExamples.Set(name, example)
			}
		}
		if newExamples.Len() > 0 {
			doc.Components.Examples = newExamples
		} else {
			doc.Components.Examples = nil
		}
	}

	// Remove unused request bodies
	if doc.Components.RequestBodies != nil {
		newRequestBodies := sequencedmap.New[string, *ReferencedRequestBody]()
		for name, requestBody := range doc.Components.RequestBodies.All() {
			if tracker.requestBodies[name] {
				newRequestBodies.Set(name, requestBody)
			}
		}
		if newRequestBodies.Len() > 0 {
			doc.Components.RequestBodies = newRequestBodies
		} else {
			doc.Components.RequestBodies = nil
		}
	}

	// Remove unused headers
	if doc.Components.Headers != nil {
		newHeaders := sequencedmap.New[string, *ReferencedHeader]()
		for name, header := range doc.Components.Headers.All() {
			if tracker.headers[name] {
				newHeaders.Set(name, header)
			}
		}
		if newHeaders.Len() > 0 {
			doc.Components.Headers = newHeaders
		} else {
			doc.Components.Headers = nil
		}
	}

	// Remove unused security schemes
	if doc.Components.SecuritySchemes != nil {
		newSecuritySchemes := sequencedmap.New[string, *ReferencedSecurityScheme]()
		for name, securityScheme := range doc.Components.SecuritySchemes.All() {
			if tracker.securitySchemes[name] {
				newSecuritySchemes.Set(name, securityScheme)
			}
		}
		if newSecuritySchemes.Len() > 0 {
			doc.Components.SecuritySchemes = newSecuritySchemes
		} else {
			doc.Components.SecuritySchemes = nil
		}
	}

	// Remove unused links
	if doc.Components.Links != nil {
		newLinks := sequencedmap.New[string, *ReferencedLink]()
		for name, link := range doc.Components.Links.All() {
			if tracker.links[name] {
				newLinks.Set(name, link)
			}
		}
		if newLinks.Len() > 0 {
			doc.Components.Links = newLinks
		} else {
			doc.Components.Links = nil
		}
	}

	// Remove unused callbacks
	if doc.Components.Callbacks != nil {
		newCallbacks := sequencedmap.New[string, *ReferencedCallback]()
		for name, callback := range doc.Components.Callbacks.All() {
			if tracker.callbacks[name] {
				newCallbacks.Set(name, callback)
			}
		}
		if newCallbacks.Len() > 0 {
			doc.Components.Callbacks = newCallbacks
		} else {
			doc.Components.Callbacks = nil
		}
	}

	// Remove unused path items
	if doc.Components.PathItems != nil {
		newPathItems := sequencedmap.New[string, *ReferencedPathItem]()
		for name, pathItem := range doc.Components.PathItems.All() {
			if tracker.pathItems[name] {
				newPathItems.Set(name, pathItem)
			}
		}
		if newPathItems.Len() > 0 {
			doc.Components.PathItems = newPathItems
		} else {
			doc.Components.PathItems = nil
		}
	}

	// If all component sections are empty, remove the components object entirely
	if doc.Components.Schemas == nil &&
		doc.Components.Responses == nil &&
		doc.Components.Parameters == nil &&
		doc.Components.Examples == nil &&
		doc.Components.RequestBodies == nil &&
		doc.Components.Headers == nil &&
		doc.Components.SecuritySchemes == nil &&
		doc.Components.Links == nil &&
		doc.Components.Callbacks == nil &&
		doc.Components.PathItems == nil {
		doc.Components = nil
	}
}

// removeUnusedTagsFromDocument prunes tags declared in the top-level tags array
// when they are not referenced by any operation's tags.
func removeUnusedTagsFromDocument(doc *OpenAPI, tracker *referencedComponentTracker) {
	if doc == nil {
		return
	}

	// If no tags are declared, nothing to do
	if len(doc.Tags) == 0 {
		return
	}

	// If there were no tags referenced, drop tags entirely
	if tracker == nil || len(tracker.tags) == 0 {
		doc.Tags = nil
		return
	}

	// Keep only tags with names referenced by operations
	kept := make([]*Tag, 0, len(doc.Tags))
	for _, tg := range doc.Tags {
		if tg == nil {
			continue
		}
		if tracker.tags[tg.GetName()] {
			kept = append(kept, tg)
		}
	}

	if len(kept) > 0 {
		doc.Tags = kept
	} else {
		doc.Tags = nil
	}
}

// walkAndTrackWithFilter walks the OpenAPI document and tracks referenced components,
// but only for WalkItems whose JSON pointer location satisfies the allow predicate.
func walkAndTrackWithFilter(ctx context.Context, doc *OpenAPI, tracker *referencedComponentTracker, allow func(ptr string) bool) error {
	for item := range Walk(ctx, doc) {
		loc := string(item.Location.ToJSONPointer())

		if !allow(loc) {
			// Skip tracking for this location
			continue
		}

		err := item.Match(Matcher{
			// Track schema references only when allowed by location
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				return trackSchemaReferences(schema, tracker)
			},
			// Track component references only when allowed by location
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return trackPathItemReference(ref, tracker.pathItems)
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return trackParameterReference(ref, tracker.parameters)
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return trackExampleReference(ref, tracker.examples)
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return trackRequestBodyReference(ref, tracker.requestBodies)
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return trackResponseReference(ref, tracker.responses)
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return trackHeaderReference(ref, tracker.headers)
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return trackCallbackReference(ref, tracker.callbacks)
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return trackLinkReference(ref, tracker.links)
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return trackSecuritySchemeReference(ref, tracker.securitySchemes)
			},
			// Track operation tags (only under allowed locations)
			Operation: func(op *Operation) error {
				if op == nil {
					return nil
				}
				for _, tag := range op.GetTags() {
					if tracker.tags == nil {
						tracker.tags = make(map[string]bool)
					}
					tracker.tags[tag] = true
				}
				return nil
			},
			// Track security requirements (special case for security schemes)
			Security: func(req *SecurityRequirement) error {
				if req != nil {
					for schemeName := range req.All() {
						tracker.securitySchemes[schemeName] = true
					}
				}
				return nil
			},
		})
		if err != nil {
			return fmt.Errorf("failed to track references: %w", err)
		}
	}

	return nil
}

// extractComponentTypeAndName returns component type and name from a JSON pointer location like:
//
//	/components/schemas/User/... -> ("schemas", "User", true)
//
// Returns ok=false if the pointer does not point under /components/{type}/{name}
func extractComponentTypeAndName(ptr string) (typ, name string, ok bool) {
	const prefix = "/components/"
	if !strings.HasPrefix(ptr, prefix) {
		return "", "", false
	}

	parts := strings.Split(ptr, "/")
	// Expect at least: "", "components", "{type}", "{name}", ...
	if len(parts) < 4 {
		return "", "", false
	}

	typ = parts[2]
	name = unescapeJSONPointerToken(parts[3])
	if typ == "" || name == "" {
		return "", "", false
	}
	return typ, name, true
}

// unescapeJSONPointerToken reverses JSON Pointer escaping (~1 => /, ~0 => ~)
func unescapeJSONPointerToken(s string) string {
	// Per RFC 6901: ~1 is '/', ~0 is '~'. Order matters: replace ~1 first, then ~0.
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}

// countTracked returns the total number of referenced components recorded in the tracker.
// This is used to detect a fixed point during reachability expansion.
func countTracked(tr *referencedComponentTracker) int {
	if tr == nil {
		return 0
	}
	total := 0
	total += len(tr.schemas)
	total += len(tr.responses)
	total += len(tr.parameters)
	total += len(tr.examples)
	total += len(tr.requestBodies)
	total += len(tr.headers)
	total += len(tr.securitySchemes)
	total += len(tr.links)
	total += len(tr.callbacks)
	total += len(tr.pathItems)
	return total
}
