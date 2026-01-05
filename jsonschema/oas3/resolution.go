package oas3

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"gopkg.in/yaml.v3"
)

// ResolveOptions represent the options available when resolving a JSON Schema reference.
type ResolveOptions = references.ResolveOptions

type JSONSchemaReferenceable = JSONSchema[Referenceable]

func (s *JSONSchema[Referenceable]) IsResolved() bool {
	if s == nil {
		return false
	}

	return !s.IsReference() || s.resolvedSchemaCache != nil || (s.referenceResolutionCache != nil && s.referenceResolutionCache.Object != nil) || s.circularErrorFound
}

// IsReference returns true if the JSONSchema is a reference, false otherwise
func (j *JSONSchema[Referenceable]) IsReference() bool {
	if j == nil || j.IsBool() {
		return false
	}

	return j.GetSchema().IsReference()
}

// GetReference returns the reference of the JSONSchema if present, otherwise an empty string
// This method is identical to GetRef() but was added to support the Resolvable interface
func (j *JSONSchema[Referenceable]) GetReference() references.Reference {
	if j == nil {
		return ""
	}

	return j.GetRef()
}

// GetRef returns the reference of the JSONSchema if present, otherwise an empty string
// This method is identical to GetReference() but was kept for backwards compatibility
func (j *JSONSchema[Referenceable]) GetRef() references.Reference {
	if j == nil || j.IsBool() {
		return ""
	}

	return j.GetSchema().GetRef()
}

// GetAbsRef returns the absolute reference of the JSONSchema if present, otherwise an empty string
func (j *JSONSchema[Referenceable]) GetAbsRef() references.Reference {
	if !j.IsReference() {
		return ""
	}

	ref := j.GetRef()
	if j.referenceResolutionCache == nil {
		return ref
	}
	return references.Reference(j.referenceResolutionCache.AbsoluteReference + "#" + ref.GetJSONPointer().String())
}

// Resolve will fully resolve the reference and return the JSONSchema referenced. This will recursively resolve any intermediate references as well.
// Validation errors can be skipped by setting the skipValidation flag to true. This will skip the missing field errors that occur during unmarshaling.
// Resolution doesn't run the Validate function on the resolved object. So if you want to fully validate the object after resolution, you need to call the Validate function manually.
func (s *JSONSchema[Referenceable]) Resolve(ctx context.Context, opts ResolveOptions) ([]error, error) {
	targetDocument := opts.TargetDocument
	if targetDocument == nil {
		targetDocument = opts.RootDocument
	}

	return resolveJSONSchemaWithTracking(ctx, (*JSONSchemaReferenceable)(unsafe.Pointer(s)), references.ResolveOptions{ //nolint:gosec
		TargetLocation:      opts.TargetLocation,
		RootDocument:        opts.RootDocument,
		TargetDocument:      targetDocument,
		DisableExternalRefs: opts.DisableExternalRefs,
		VirtualFS:           opts.VirtualFS,
		HTTPClient:          opts.HTTPClient,
	}, []string{})
}

// GetResolvedObject will return either this schema or the referenced schema if previously resolved.
// This methods is identical to GetResolvedSchema but was added to support the Resolvable interface
func (s *JSONSchema[Referenceable]) GetResolvedObject() *JSONSchema[Concrete] {
	if s == nil {
		return nil
	}

	return s.GetResolvedSchema()
}

// GetResolvedSchema will return either this schema or the referenced schema if previously resolved.
// This methods is identical to GetResolvedObject but was kept for backwards compatibility
func (s *JSONSchema[Referenceable]) GetResolvedSchema() *JSONSchema[Concrete] {
	if s == nil || !s.IsResolved() {
		return nil
	}

	if s.resolvedSchemaCache != nil {
		return s.resolvedSchemaCache
	}

	var result *JSONSchema[Concrete]

	if !s.IsReference() {
		result = (*JSONSchema[Concrete])(unsafe.Pointer(s)) //nolint:gosec
	} else {
		if s.referenceResolutionCache == nil || s.referenceResolutionCache.Object == nil {
			return nil
		}

		// Get the resolved schema from the cache
		resolvedSchema := s.referenceResolutionCache.Object

		// If the resolved schema is itself a reference, we need to get its resolved form
		if resolvedSchema.IsReference() {
			// Get the final resolved schema from the referenced schema
			result = resolvedSchema.GetResolvedSchema()
			if result == nil {
				return nil
			}
		} else {
			result = (*JSONSchema[Concrete])(unsafe.Pointer(resolvedSchema)) //nolint:gosec
		}
	}

	s.resolvedSchemaCache = result
	return result
}

// MustGetResolvedSchema will return the resolved schema. If this is a reference and its unresolved, this will panic.
// Useful if references have been resolved before hand.
func (s *JSONSchema[Referenceable]) MustGetResolvedSchema() *JSONSchema[Concrete] {
	if s == nil {
		return nil
	}

	obj := s.GetResolvedSchema()
	if s.IsReference() && obj == nil {
		panic("unresolved reference, resolve first")
	}
	return obj
}

func (r *JSONSchema[Referenceable]) GetReferenceResolutionInfo() *references.ResolveResult[JSONSchemaReferenceable] {
	if r == nil {
		return nil
	}

	if !r.IsReference() {
		return nil
	}

	if r.referenceResolutionCache == nil {
		return nil
	}

	return r.referenceResolutionCache
}

// ReferenceChainEntry represents a step in the reference resolution chain.
// Each entry contains the schema that holds the reference and the reference itself.
type ReferenceChainEntry struct {
	// Schema is the JSONSchema node that contains the $ref.
	// This is the schema that was resolved to get to the next step in the chain.
	Schema *JSONSchema[Referenceable]

	// Reference is the $ref value from the schema (e.g., "#/components/schemas/User").
	Reference references.Reference
}

// GetReferenceChain returns the chain of references that were followed to resolve this schema.
// The chain is ordered from the outermost reference (top-level parent) to the innermost (immediate parent).
// Returns nil if this schema was not resolved via references.
//
// Example: If a response schema references Schema1, which references SchemaShared,
// calling GetReferenceChain() on the resolved SchemaShared would return:
//   - [0]: response schema with reference "#/components/schemas/Schema1"
//   - [1]: Schema1 with reference "#/components/schemas/SchemaShared"
//
// This allows tracking which schemas first referenced nested schemas during iteration.
func (j *JSONSchema[T]) GetReferenceChain() []*ReferenceChainEntry {
	if j == nil || j.parent == nil {
		return nil
	}

	var chain []*ReferenceChainEntry

	// Walk from the immediate parent up to the top-level
	current := j.parent
	for current != nil {
		if current.IsReference() {
			entry := &ReferenceChainEntry{
				Schema:    current,
				Reference: current.GetRef(),
			}
			// Prepend to get topLevel first (outer -> inner order)
			chain = append([]*ReferenceChainEntry{entry}, chain...)
		}

		// Move to the parent of current
		current = current.GetParent()
	}

	return chain
}

// GetImmediateReference returns the immediate parent reference that resolved to this schema.
// Returns nil if this schema was not resolved via a reference.
//
// This is a convenience method equivalent to getting the last element of GetReferenceChain().
func (j *JSONSchema[T]) GetImmediateReference() *ReferenceChainEntry {
	if j == nil || j.parent == nil || !j.parent.IsReference() {
		return nil
	}

	return &ReferenceChainEntry{
		Schema:    j.parent,
		Reference: j.parent.GetRef(),
	}
}

// GetTopLevelReference returns the outermost (first) reference in the chain that led to this schema.
// Returns nil if this schema was not resolved via a reference.
//
// This is a convenience method equivalent to getting the first element of GetReferenceChain().
func (j *JSONSchema[T]) GetTopLevelReference() *ReferenceChainEntry {
	if j == nil || j.topLevelParent == nil || !j.topLevelParent.IsReference() {
		return nil
	}

	return &ReferenceChainEntry{
		Schema:    j.topLevelParent,
		Reference: j.topLevelParent.GetRef(),
	}
}

func (s *JSONSchema[Referenceable]) resolve(ctx context.Context, opts references.ResolveOptions, referenceChain []string) ([]string, []error, error) {
	if !s.IsReference() {
		return referenceChain, nil, nil
	}

	// Check if we have a cached resolved schema don't bother resolving it again
	if s.referenceResolutionCache != nil {
		if s.referenceResolutionCache.Object != nil {
			return nil, nil, nil
		}

		// For chained resolutions or refs found in external docs, we need to use the resolved document from the previous step
		// The ResolveResult.ResolvedDocument should be used as the new TargetDocument
		if s.referenceResolutionCache.ResolvedDocument != nil {
			opts.TargetDocument = s.referenceResolutionCache.ResolvedDocument
			opts.TargetLocation = s.referenceResolutionCache.AbsoluteReference
		}
	}

	// Get the absolute reference string for tracking using the extracted logic
	ref := s.GetRef()

	absRefResult, err := references.ResolveAbsoluteReference(ref, opts.TargetLocation)
	if err != nil {
		return nil, nil, err
	}

	jsonPtr := string(ref.GetJSONPointer())
	absRef := utils.BuildAbsoluteReference(absRefResult.AbsoluteReference, jsonPtr)

	// Special case: detect self-referencing schemas (references to root document)
	// This catches cases like "#" which reference the root document itself
	// Only consider it circular if this schema has no parent (i.e., it's at the root level)
	if ref.GetURI() == "" && ref.GetJSONPointer() == "" {
		// Check if this schema has a parent - if it does, then referencing "#" is legitimate
		// If it has no parent, then it's the root schema referencing itself, which is circular
		if s.GetParent() == nil && s.GetTopLevelParent() == nil {
			s.circularErrorFound = true
			return nil, nil, errors.New("circular reference detected: self-referencing schema")
		}
	}

	// Check for circular reference by looking for the current reference in the chain
	for _, chainRef := range referenceChain {
		if chainRef == absRef {
			// Build circular reference error message showing the full chain
			chainWithCurrent := referenceChain
			chainWithCurrent = append(chainWithCurrent, absRef)
			s.circularErrorFound = true
			return nil, nil, fmt.Errorf("circular reference detected: %s", joinReferenceChain(chainWithCurrent))
		}
	}

	// Add this reference to the chain
	newChain := referenceChain
	newChain = append(newChain, absRef)

	var result *references.ResolveResult[JSONSchemaReferenceable]
	var validationErrs []error

	// Check if this is a $defs reference and handle it specially
	if strings.HasPrefix(string(ref.GetJSONPointer()), "/$defs/") {
		result, validationErrs, err = s.resolveDefsReference(ctx, ref, opts)
	} else {
		// Resolve as JSONSchema to handle both Schema and boolean cases
		result, validationErrs, err = references.Resolve(ctx, ref, unmarshaler, opts)
	}
	if err != nil {
		return nil, validationErrs, err
	}

	schema := result.Object
	for item := range Walk(ctx, schema) {
		_ = item.Match(SchemaMatcher{
			Schema: func(js *JSONSchemaReferenceable) error {
				if js.IsReference() {
					js.referenceResolutionCache = &references.ResolveResult[JSONSchemaReferenceable]{
						AbsoluteReference: result.AbsoluteReference,
						ResolvedDocument:  result.ResolvedDocument,
					}
				}
				return nil
			},
		})
	}

	s.referenceResolutionCache = result
	s.validationErrsCache = validationErrs

	return newChain, validationErrs, nil
}

// joinReferenceChain joins a chain of references with arrows for error messages
func joinReferenceChain(chain []string) string {
	if len(chain) == 0 {
		return ""
	}
	if len(chain) == 1 {
		return chain[0]
	}
	return strings.Join(chain, " -> ")
}

// resolveJSONSchemaWithTracking recursively resolves references while tracking visited references to detect cycles
func resolveJSONSchemaWithTracking(ctx context.Context, schema *JSONSchema[Referenceable], opts references.ResolveOptions, referenceChain []string) ([]error, error) {
	// If this is not a reference, return the inline object
	if !schema.IsReference() {
		return nil, nil
	}

	// Resolve the current reference
	newChain, validationErrs, err := schema.resolve(ctx, opts, referenceChain)
	if err != nil {
		return validationErrs, err
	}

	var obj *JSONSchema[Referenceable]
	if schema.referenceResolutionCache != nil {
		obj = schema.referenceResolutionCache.Object
	}

	if obj == nil {
		return validationErrs, fmt.Errorf("unable to resolve reference: %s", schema.GetRef())
	}

	if obj.IsBool() {
		return validationErrs, nil
	}

	// Set parent links for the resolved object
	// The resolved object's parent is the current schema (which is a reference)
	// The top-level parent is either the current schema's top-level parent, or the current schema if it's the top-level
	var topLevel *JSONSchema[Referenceable]
	if schema.topLevelParent != nil {
		topLevel = schema.topLevelParent
	} else {
		topLevel = schema
	}
	obj.SetParent(schema)
	obj.SetTopLevelParent(topLevel)

	// If we got another reference, recursively resolve it with the resolved document as the new target
	if obj.IsReference() {
		return resolveJSONSchemaWithTracking(ctx, obj, opts, newChain)
	}

	return validationErrs, nil
}

// resolveDefsReference handles special resolution for $defs references
// It uses the standard references.Resolve infrastructure but adjusts the target document for $defs resolution
func (s *JSONSchema[Referenceable]) resolveDefsReference(ctx context.Context, ref references.Reference, opts references.ResolveOptions) (*references.ResolveResult[JSONSchemaReferenceable], []error, error) {
	jp := ref.GetJSONPointer()

	// Validate this is a $defs reference
	if !strings.HasPrefix(jp.String(), "/$defs/") {
		return nil, nil, fmt.Errorf("not a $defs reference: %s", ref)
	}

	// First, try to resolve using the standard references.Resolve with the target document
	// This handles external $defs, caching, and all standard resolution features
	result, validationErrs, err := references.Resolve(ctx, ref, unmarshaler, opts)
	if err == nil {
		return result, validationErrs, nil
	}

	// If standard resolution failed and we have a parent, try resolving with the parent as target
	if parent := s.GetParent(); parent != nil {
		parentOpts := opts
		parentOpts.TargetDocument = parent
		parentOpts.TargetLocation = opts.TargetLocation // Keep the same location for caching

		result, validationErrs, err := references.Resolve(ctx, ref, unmarshaler, parentOpts)
		if err == nil {
			return result, validationErrs, nil
		}
	}

	// Fallback: try JSON pointer navigation when no parent chain exists
	if s.GetParent() == nil && s.GetTopLevelParent() == nil {
		result, validationErrs, err := s.tryResolveDefsUsingJSONPointerNavigation(ctx, ref, opts)
		if err == nil && result != nil {
			return result, validationErrs, nil
		}
	}

	return nil, nil, fmt.Errorf("definition not found: %s", ref)
}

type GetRootNoder interface {
	GetRootNode() *yaml.Node
}

// tryResolveDefsUsingJSONPointerNavigation attempts to resolve $defs by walking up the JSON pointer structure
// This is used when there's no parent chain available
func (s *JSONSchema[Referenceable]) tryResolveDefsUsingJSONPointerNavigation(ctx context.Context, ref references.Reference, opts references.ResolveOptions) (*references.ResolveResult[JSONSchemaReferenceable], []error, error) {
	// When we don't have a parent chain, we need to find our location in the document
	// and walk up the JSON pointer chain to find parent schemas

	// Get the top-level root node from the target document
	var topLevelRootNode *yaml.Node
	if targetDoc, ok := opts.TargetDocument.(GetRootNoder); ok {
		topLevelRootNode = targetDoc.GetRootNode()
	}

	if topLevelRootNode == nil {
		return nil, nil, nil
	}

	// Get our JSON pointer location within the document using the CoreModel
	ourJSONPtr := s.GetCore().GetJSONPointer(topLevelRootNode)
	if ourJSONPtr == "" {
		return nil, nil, nil
	}

	// Walk up the parent JSON pointers
	parentJSONPtr := getParentJSONPointer(ourJSONPtr)
	for parentJSONPtr != "" {
		// Get the parent target using JSON pointer
		parentTarget, err := jsonpointer.GetTarget(opts.TargetDocument, jsonpointer.JSONPointer(parentJSONPtr), jsonpointer.WithStructTags("key"))
		if err == nil {
			parentOpts := opts
			parentOpts.TargetDocument = parentTarget
			parentOpts.TargetLocation = opts.TargetLocation // Keep the same location for caching

			result, validationErrs, err := references.Resolve(ctx, ref, unmarshaler, parentOpts)
			if err == nil {
				return result, validationErrs, nil
			}
		}

		// Move up to the next parent
		parentJSONPtr = getParentJSONPointer(parentJSONPtr)
	}

	return nil, nil, fmt.Errorf("definition not found: %s", ref)
}

// getParentJSONPointer returns the parent JSON pointer by removing the last segment
// e.g., "/properties/nested/properties/inner" -> "/properties/nested/properties"
// Returns empty string when reaching the root
func getParentJSONPointer(jsonPtr string) string {
	if jsonPtr == "" || jsonPtr == "/" {
		return ""
	}

	// Find the last slash
	lastSlash := strings.LastIndex(jsonPtr, "/")
	if lastSlash <= 0 {
		return ""
	}

	return jsonPtr[:lastSlash]
}

func unmarshaler(ctx context.Context, node *yaml.Node, skipValidation bool) (*JSONSchema[Referenceable], []error, error) {
	jsonSchema := &JSONSchema[Referenceable]{}
	validationErrs, err := marshaller.UnmarshalNode(ctx, "", node, jsonSchema)
	if skipValidation {
		validationErrs = nil
	}
	if err != nil {
		return nil, validationErrs, err
	}

	return jsonSchema, validationErrs, nil
}
