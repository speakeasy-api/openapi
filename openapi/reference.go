package openapi

import (
	"context"
	"fmt"
	"sync"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type (
	// ReferencedPathItem represents a path item that can either be referenced from elsewhere or declared inline.
	ReferencedPathItem = Reference[PathItem, *PathItem, *core.PathItem]
	// ReferencedExample represents an example that can either be referenced from elsewhere or declared inline.
	ReferencedExample = Reference[Example, *Example, *core.Example]
	// ReferencedParameter represents a parameter that can either be referenced from elsewhere or declared inline.
	ReferencedParameter = Reference[Parameter, *Parameter, *core.Parameter]
	// ReferencedHeader represents a header that can either be referenced from elsewhere or declared inline.
	ReferencedHeader = Reference[Header, *Header, *core.Header]
	// ReferencedRequestBody represents a request body that can either be referenced from elsewhere or declared inline.
	ReferencedRequestBody = Reference[RequestBody, *RequestBody, *core.RequestBody]
	// ReferencedCallback represents a callback that can either be referenced from elsewhere or declared inline.
	ReferencedCallback = Reference[Callback, *Callback, *core.Callback]
	// ReferencedResponse represents a response that can either be referenced from elsewhere or declared inline.
	ReferencedResponse = Reference[Response, *Response, *core.Response]
	// ReferencedLink represents a link that can either be referenced from elsewhere or declared inline.
	ReferencedLink = Reference[Link, *Link, *core.Link]
	// ReferencedSecurityScheme represents a security scheme that can either be referenced from elsewhere or declared inline.
	ReferencedSecurityScheme = Reference[SecurityScheme, *SecurityScheme, *core.SecurityScheme]
)

type ReferencedObject[T any] interface {
	IsReference() bool
	GetObject() *T
}

type Reference[T any, V interfaces.Validator[T], C marshaller.CoreModeler] struct {
	marshaller.Model[core.Reference[C]]

	// Reference is the URI to the
	Reference *references.Reference

	// A short summary of the referenced object. Should override any summary provided in the referenced object.
	Summary *string
	// A longer description of the referenced object. Should override any description provided in the referenced object.
	Description *string

	// If this was an inline object instead of a reference this will contain that object.
	Object *T

	// Mutex to protect concurrent access to cache fields (pointer to allow struct copying)
	cacheMutex               *sync.RWMutex
	referenceResolutionCache *references.ResolveResult[Reference[T, V, C]]
	validationErrsCache      []error
	circularErrorFound       bool

	// Parent reference links - private fields to avoid serialization
	// These are set when the reference was resolved via a reference chain.
	//
	// Parent links are only set if this reference was accessed through reference resolution.
	// If you access a reference directly (e.g., by iterating through a document's components),
	// these will be nil even if the reference could be referenced elsewhere.
	//
	// Example scenarios when parent links are set:
	// - Single reference: main.yaml#/components/parameters/Param -> Parameter object
	//   parent = nil, topLevelParent = nil (this is the original reference)
	// - Chained reference: main.yaml -> external.yaml#/Param -> final Parameter object
	//   For the intermediate reference: parent = original reference, topLevelParent = original reference
	//   For the final resolved object: parent links are set during resolution
	parent         *Reference[T, V, C] // Immediate parent reference in the chain
	topLevelParent *Reference[T, V, C] // Top-level parent (root of the reference chain)
}

var _ interfaces.Model[core.Reference[*core.Info]] = (*Reference[Info, *Info, *core.Info])(nil)

// ResolveOptions represent the options available when resolving a reference.
type ResolveOptions = references.ResolveOptions

// Resolve will fully resolve the reference and return the object referenced. This will recursively resolve any intermediate references as well. Will return errors if there is a circular reference issue.
// Validation errors can be skipped by setting the skipValidation flag to true. This will skip the missing field errors that occur during unmarshaling.
// Resolution doesn't run the Validate function on the resolved object. So if you want to fully validate the object after resolution, you need to call the Validate function manually.
func (r *Reference[T, V, C]) Resolve(ctx context.Context, opts ResolveOptions) ([]error, error) {
	if r == nil {
		return nil, nil
	}

	return resolveObjectWithTracking(ctx, r, references.ResolveOptions{
		RootDocument:        opts.RootDocument,
		TargetLocation:      opts.TargetLocation,
		TargetDocument:      opts.RootDocument,
		DisableExternalRefs: opts.DisableExternalRefs,
		VirtualFS:           opts.VirtualFS,
		HTTPClient:          opts.HTTPClient,
	}, []string{})
}

// IsReference returns true if the reference is a reference (via $ref) to an object as opposed to an inline object.
func (r *Reference[T, V, C]) IsReference() bool {
	if r == nil {
		return false
	}
	return r.Reference != nil
}

// IsResolved returns true if the reference is resolved (not a reference or the reference has been resolved)
func (r *Reference[T, V, C]) IsResolved() bool {
	if r == nil {
		return false
	}

	if !r.IsReference() {
		return true
	}

	r.ensureMutex()
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()
	return (r.referenceResolutionCache != nil && r.referenceResolutionCache.Object != nil) || r.circularErrorFound
}

// GetReference returns the value of the Reference field. Returns empty string if not set.
func (r *Reference[T, V, C]) GetReference() references.Reference {
	if r == nil || r.Reference == nil {
		return ""
	}
	return *r.Reference
}

// GetObject returns the referenced object. If this is a reference and its unresolved, this will return nil.
func (r *Reference[T, V, C]) GetObject() *T {
	if r == nil {
		return nil
	}

	if !r.IsReference() {
		return r.Object
	}

	r.ensureMutex()
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	if (r.referenceResolutionCache != nil && r.referenceResolutionCache.Object != nil) || r.circularErrorFound {
		if r.referenceResolutionCache != nil && r.referenceResolutionCache.Object != nil {
			return r.referenceResolutionCache.Object.GetObject()
		}
	}
	return nil
}

// MustGetObject will return the referenced object. If this is a reference and its unresolved, this will panic.
// Useful if references have been resolved before hand.
func (r *Reference[T, V, C]) MustGetObject() *T {
	if r == nil {
		return nil
	}

	obj := r.GetObject()
	if r.IsReference() && obj == nil {
		panic("unresolved reference, resolve first")
	}
	return obj
}

// GetSummary returns the value of the Summary field. Returns empty string if not set.
func (r *Reference[T, V, C]) GetSummary() string {
	if r == nil || r.Summary == nil {
		return ""
	}
	return *r.Summary
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (r *Reference[T, V, C]) GetDescription() string {
	if r == nil || r.Description == nil {
		return ""
	}
	return *r.Description
}

// GetParent returns the immediate parent reference if this reference was resolved via a reference chain.
//
// Returns nil if:
// - This reference was not resolved via a reference (accessed directly)
// - This reference is the top-level reference in a chain
// - The reference was accessed by iterating through document components rather than reference resolution
//
// Example: main.yaml -> external.yaml#/Parameter -> Parameter object
// The intermediate external.yaml reference's GetParent() returns the original main.yaml reference.
func (r *Reference[T, V, C]) GetParent() *Reference[T, V, C] {
	if r == nil {
		return nil
	}
	return r.parent
}

// GetTopLevelParent returns the top-level parent reference if this reference was resolved via a reference chain.
//
// Returns nil if:
// - This reference was not resolved via a reference (accessed directly)
// - This reference is already the top-level reference
// - The reference was accessed by iterating through document components rather than reference resolution
//
// Example: main.yaml -> external.yaml#/Param -> chained.yaml#/Param -> final Parameter object
// The intermediate references' GetTopLevelParent() returns the original main.yaml reference.
func (r *Reference[T, V, C]) GetTopLevelParent() *Reference[T, V, C] {
	if r == nil {
		return nil
	}
	return r.topLevelParent
}

// SetParent sets the immediate parent reference for this reference.
// This is a public API for manually constructing reference chains.
//
// Use this when you need to manually establish parent-child relationships
// between references, typically when creating reference chains programmatically
// rather than through the normal resolution process.
func (r *Reference[T, V, C]) SetParent(parent *Reference[T, V, C]) {
	if r == nil {
		return
	}
	r.parent = parent
}

// SetTopLevelParent sets the top-level parent reference for this reference.
// This is a public API for manually constructing reference chains.
//
// Use this when you need to manually establish the root of a reference chain,
// typically when creating reference chains programmatically rather than
// through the normal resolution process.
func (r *Reference[T, V, C]) SetTopLevelParent(topLevelParent *Reference[T, V, C]) {
	if r == nil {
		return
	}
	r.topLevelParent = topLevelParent
}

// Validate will validate the reusable object against the Arazzo specification.
func (r *Reference[T, V, C]) Validate(ctx context.Context, opts ...validation.Option) []error {
	if r == nil {
		return []error{fmt.Errorf("reference is nil")}
	}

	core := r.GetCore()
	if core == nil {
		return []error{fmt.Errorf("reference core is nil")}
	}

	errs := []error{}

	if core.Reference.Present {
		if err := r.Reference.Validate(); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError(err.Error()), core, core.Reference))
		}
	} else if r.Object != nil {
		// Use the validator interface V to validate the object
		var validator V
		if v, ok := any(r.Object).(V); ok {
			validator = v
			errs = append(errs, validator.Validate(ctx, opts...)...)
		}
	}

	r.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

func (r *Reference[T, V, C]) Populate(source any) error {
	var s *core.Reference[C]
	switch src := source.(type) {
	case *core.Reference[C]:
		s = src
	case core.Reference[C]:
		s = &src
	default:
		return fmt.Errorf("expected *core.Reference[C] or core.Reference[C], got %T", source)
	}

	if s.Reference.Present {
		r.Reference = pointer.From(references.Reference(*s.Reference.Value))
		r.Summary = s.Summary.Value
		r.Description = s.Description.Value
	} else {
		if err := marshaller.Populate(s.Object, &r.Object); err != nil {
			return err
		}
	}

	r.SetCore(s)

	return nil
}

func (r *Reference[T, V, C]) GetNavigableNode() (any, error) {
	if !r.IsReference() {
		return r.Object, nil
	}

	obj := r.GetObject()
	if obj == nil {
		return nil, fmt.Errorf("unresolved reference")
	}
	return obj, nil
}

func (r *Reference[T, V, C]) resolve(ctx context.Context, opts references.ResolveOptions) (*T, *Reference[T, V, C], []error, error) {
	if !r.IsReference() {
		return r.Object, nil, nil, nil
	}

	r.ensureMutex()

	// Check if already resolved (with read lock)
	r.cacheMutex.RLock()
	if r.referenceResolutionCache != nil {
		cache := r.referenceResolutionCache
		validationErrs := r.validationErrsCache
		r.cacheMutex.RUnlock()

		if cache.Object.IsReference() {
			return nil, cache.Object, validationErrs, nil
		} else {
			return cache.Object.Object, nil, validationErrs, nil
		}
	}
	r.cacheMutex.RUnlock()

	// Need to resolve (with write lock)
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	// Double-check after acquiring write lock
	if r.referenceResolutionCache != nil {
		if r.referenceResolutionCache.Object.IsReference() {
			return nil, r.referenceResolutionCache.Object, r.validationErrsCache, nil
		} else {
			return r.referenceResolutionCache.Object.Object, nil, r.validationErrsCache, nil
		}
	}

	rootDoc, ok := opts.RootDocument.(*OpenAPI)
	if !ok {
		return nil, nil, nil, fmt.Errorf("root document must be *OpenAPI, got %T", opts.RootDocument)
	}
	result, validationErrs, err := references.Resolve(ctx, *r.Reference, unmarshaller[T, V, C](rootDoc), opts)
	if err != nil {
		return nil, nil, validationErrs, err
	}

	r.referenceResolutionCache = result
	r.validationErrsCache = validationErrs

	if r.referenceResolutionCache.Object.IsReference() {
		return nil, r.referenceResolutionCache.Object, r.validationErrsCache, nil
	} else {
		return r.referenceResolutionCache.Object.Object, nil, r.validationErrsCache, nil
	}
}

// resolveObjectWithTracking recursively resolves references while tracking visited references to detect cycles
func resolveObjectWithTracking[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, ref *Reference[T, V, C], opts references.ResolveOptions, referenceChain []string) ([]error, error) {
	// If this is not a reference, return the inline object
	if !ref.IsReference() {
		return nil, nil
	}

	// Get the absolute reference string for tracking using the extracted logic
	reference := ref.GetReference()

	absRefResult, err := references.ResolveAbsoluteReference(reference, opts.TargetLocation)
	if err != nil {
		return nil, err
	}

	jsonPtr := string(reference.GetJSONPointer())
	absRef := utils.BuildAbsoluteReference(absRefResult.AbsoluteReference, jsonPtr)

	// Check for circular reference by looking for the current reference in the chain
	for _, chainRef := range referenceChain {
		if chainRef == absRef {
			// Build circular reference error message showing the full chain
			chainWithCurrent := append(referenceChain, absRef)
			ref.ensureMutex()
			ref.cacheMutex.Lock()
			ref.circularErrorFound = true
			ref.cacheMutex.Unlock()
			return nil, fmt.Errorf("circular reference detected: %s", joinReferenceChain(chainWithCurrent))
		}
	}

	// Add this reference to the chain
	newChain := append(referenceChain, absRef)

	// Resolve the current reference
	obj, nextRef, validationErrs, err := ref.resolve(ctx, opts)
	if err != nil {
		return validationErrs, err
	}

	// If we have an object already resolved then finish here
	if obj != nil {
		return validationErrs, nil
	}

	// If we got another reference, recursively resolve it with the resolved document as the new target
	if nextRef != nil {
		// Set parent links for the resolved reference
		// The resolved reference's parent is the current reference
		// The top-level parent is either the current reference's top-level parent, or the current reference if it's the top-level
		var topLevel *Reference[T, V, C]
		if ref.topLevelParent != nil {
			topLevel = ref.topLevelParent
		} else {
			topLevel = ref
		}
		nextRef.SetParent(ref)
		nextRef.SetTopLevelParent(topLevel)

		// For chained resolutions, we need to use the resolved document from the previous step
		// The ResolveResult.ResolvedDocument should be used as the new TargetDocument
		ref.ensureMutex()
		ref.cacheMutex.RLock()
		targetDoc := ref.referenceResolutionCache.ResolvedDocument
		targetLoc := ref.referenceResolutionCache.AbsoluteReference
		ref.cacheMutex.RUnlock()

		opts.TargetDocument = targetDoc
		opts.TargetLocation = targetLoc
		return resolveObjectWithTracking(ctx, nextRef, opts, newChain)
	}

	return validationErrs, fmt.Errorf("unable to resolve reference: %s", ref.GetReference())
}

// joinReferenceChain joins the reference chain with arrows to show the circular path
func joinReferenceChain(chain []string) string {
	if len(chain) == 0 {
		return ""
	}
	if len(chain) == 1 {
		return chain[0]
	}

	result := chain[0]
	for i := 1; i < len(chain); i++ {
		result += " -> " + chain[i]
	}
	return result
}

func unmarshaller[T any, V interfaces.Validator[T], C marshaller.CoreModeler](o *OpenAPI) func(context.Context, *yaml.Node, bool) (*Reference[T, V, C], []error, error) {
	return func(ctx context.Context, node *yaml.Node, skipValidation bool) (*Reference[T, V, C], []error, error) {
		var ref Reference[T, V, C]
		validationErrs, err := marshaller.UnmarshalNode(ctx, node, &ref)
		if skipValidation {
			validationErrs = nil
		}
		if err != nil {
			return nil, validationErrs, err
		}

		return &ref, validationErrs, nil
	}
}

// ensureMutex initializes the mutex if it's nil (lazy initialization)
func (r *Reference[T, V, C]) ensureMutex() {
	if r.cacheMutex == nil {
		r.cacheMutex = &sync.RWMutex{}
	}
}
