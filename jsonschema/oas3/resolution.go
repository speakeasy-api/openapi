package oas3

import (
	"context"
	"fmt"
	"strings"
	"unsafe"

	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"go.yaml.in/yaml/v4"
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
	return references.Reference(j.referenceResolutionCache.AbsoluteDocumentPath + "#" + ref.GetJSONPointer().String())
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
			opts.TargetLocation = s.referenceResolutionCache.AbsoluteDocumentPath
		}
	}

	// Get the absolute reference string for tracking using the extracted logic
	ref := s.GetRef()

	// Determine the effective base URI for this schema
	// This accounts for nested $id values that change the base URI for relative references
	effectiveBase := s.getEffectiveBaseURI(opts)

	// Try to resolve via registry first ($id and $anchor lookups)
	if result := s.tryResolveViaRegistry(ctx, ref, opts); result != nil {
		// Compute absolute reference for circular detection
		// Use the result's AbsoluteReference combined with any anchor/fragment
		absRef := result.AbsoluteDocumentPath
		if anchor := ExtractAnchor(string(ref)); anchor != "" {
			absRef = absRef + "#" + anchor
		} else if jp := ref.GetJSONPointer(); jp != "" {
			absRef = absRef + "#" + string(jp)
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

		s.referenceResolutionCache = result
		return newChain, nil, nil
	}

	absRefResult, err := references.ResolveAbsoluteReference(ref, effectiveBase)
	if err != nil {
		return nil, nil, err
	}

	jsonPtr := string(ref.GetJSONPointer())
	absRef := utils.BuildAbsoluteReference(absRefResult.AbsoluteReference, jsonPtr)

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

	// Create resolve opts with the effective base URI
	// This ensures relative references are resolved against the correct base (from $id chain)
	resolveOpts := opts
	if effectiveBase != "" && effectiveBase != opts.TargetLocation {
		resolveOpts.TargetLocation = effectiveBase
	}

	// Determine resolution strategy based on reference type
	switch {
	case strings.HasPrefix(string(ref.GetJSONPointer()), "/$defs/"):
		// Check if this is a $defs reference and handle it specially
		result, validationErrs, err = s.resolveDefsReference(ctx, ref, resolveOpts)
	case ref.IsAnchorReference() && ref.GetURI() != "":
		// Handle external anchor references specially - fetch the document first, then resolve the anchor
		result, validationErrs, err = s.resolveExternalAnchorReference(ctx, ref, resolveOpts)
	case ref.GetURI() != "" && ref.GetJSONPointer() != "":
		// Handle external references with JSON pointer fragments
		// We need to fetch the whole document, set up its registry, then navigate to the fragment
		result, validationErrs, err = s.resolveExternalRefWithFragment(ctx, ref, resolveOpts)
	default:
		// Resolve as JSONSchema to handle both Schema and boolean cases
		result, validationErrs, err = references.Resolve(ctx, ref, unmarshaler, resolveOpts)
	}
	if err != nil {
		return nil, validationErrs, err
	}

	schema := result.Object

	// Use $id as base URI if present in the resolved schema (JSON Schema spec)
	// The $id keyword identifies a schema resource with its canonical URI
	// and serves as the base URI for relative references within that schema
	baseURI := result.AbsoluteDocumentPath
	if !schema.IsBool() && schema.GetSchema() != nil {
		if schemaID := schema.GetSchema().GetID(); schemaID != "" {
			baseURI = schemaID
		}
	}

	// Set up the schema registry for remote schemas
	// This enables $id and $anchor resolution within the fetched document
	setupRemoteSchemaRegistry(ctx, schema, baseURI)

	// Collect nested reference schemas that need parent links set
	var nestedRefs []*JSONSchemaReferenceable

	for item := range Walk(ctx, schema) {
		_ = item.Match(SchemaMatcher{
			Schema: func(js *JSONSchemaReferenceable) error {
				if js.IsReference() {
					// Check if this schema has its own $id, which would override the parent's base URI
					localBaseURI := baseURI
					if !js.IsBool() && js.GetSchema() != nil {
						if jsID := js.GetSchema().GetID(); jsID != "" {
							localBaseURI = jsID
						}
					}
					// Get the ref to build absolute reference with fragment
					jsRef := js.GetRef()
					absRef := utils.BuildAbsoluteReference(localBaseURI, string(jsRef.GetJSONPointer()))
					js.referenceResolutionCache = &references.ResolveResult[JSONSchemaReferenceable]{
						AbsoluteDocumentPath: localBaseURI,
						AbsoluteReference:    references.Reference(absRef),
						ResolvedDocument:     result.ResolvedDocument,
					}

					// Collect this reference for setting parent links after the walk
					nestedRefs = append(nestedRefs, js)
				}
				return nil
			},
		})
	}

	// Set parent links for all nested references found during the walk
	// This maintains reference chain tracking when accessing properties of resolved schemas
	var topLevel *JSONSchemaReferenceable
	if s.topLevelParent != nil {
		topLevel = s.topLevelParent
	} else {
		topLevel = (*JSONSchemaReferenceable)(s)
	}
	for _, js := range nestedRefs {
		js.SetParent((*JSONSchemaReferenceable)(s))
		js.SetTopLevelParent(topLevel)
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
