package references

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/system"
	"gopkg.in/yaml.v3"
)

type ResolutionTarget interface {
	InitCache()

	GetCachedReferencedObject(key string) (any, bool)
	StoreReferencedObjectInCache(key string, obj any)

	GetCachedReferenceDocument(key string) ([]byte, bool)
	StoreReferenceDocumentInCache(key string, doc []byte)
}

type Resolvable[T any] interface {
	GetReference() Reference
	Resolve(ctx context.Context, opts ResolveOptions) ([]error, error)
	IsResolved() bool
	GetResolvedObject() *T
}

// AbsoluteReferenceResult contains the result of resolving an absolute reference
type AbsoluteReferenceResult struct {
	// AbsoluteReference is the resolved absolute reference string
	AbsoluteReference string
	// Classification contains the reference type classification
	Classification *utils.ReferenceClassification
}

// ResolveAbsoluteReference resolves a reference to an absolute reference string
// based on the target location. It handles empty URIs, absolute URLs, absolute file paths,
// and relative URIs that need to be joined with the target location.
// This function now uses caching to avoid repeated computation of the same (reference, target) pairs.
func ResolveAbsoluteReference(ref Reference, targetLocation string) (*AbsoluteReferenceResult, error) {
	return ResolveAbsoluteReferenceCached(ref, targetLocation)
}

type Unmarshal[T any] func(ctx context.Context, node *yaml.Node, skipValidation bool) (*T, []error, error)

// ResolveResult contains the result of a reference resolution operation
type ResolveResult[T any] struct {
	// Object is the resolved object
	Object *T
	// AbsoluteDocumentPath is the absolute reference that was resolved
	AbsoluteDocumentPath string
	// AbsoluteReference is the absolute reference that was resolved
	AbsoluteReference Reference
	// ResolvedDocument is the document that was resolved against (for chaining resolutions)
	ResolvedDocument any
}

// ResolveOptions represent the options available when resolving a reference.
type ResolveOptions struct {
	// RootDocument is the root document of the resolution chain, will be resolved against if TargetDocument is not set. Will hold the cached resolutions results.
	RootDocument ResolutionTarget
	// TargetLocation should represent the absolute location on disk or URL of the target document. All references will be resolved relative to this location.
	TargetLocation string
	// TargetDocument is the document that will be used to resolve references against.
	TargetDocument any
	// DisableExternalRefs will disable resolving external references.
	DisableExternalRefs bool
	// VirtualFS is an optional virtual file system that will be used for any file based references. If not provided normal file system operations will be used.
	VirtualFS system.VirtualFS
	// HTTPClient is an optional HTTP client that will be used for any HTTP based references. If not provided http.DefaultClient will be used.
	HTTPClient system.Client
	// SkipValidation will skip validation of the target document during resolution.
	SkipValidation bool
}

func Resolve[T any](ctx context.Context, ref Reference, unmarshaler Unmarshal[T], opts ResolveOptions) (*ResolveResult[T], []error, error) {
	if opts.RootDocument == nil {
		return nil, nil, errors.New("root document is required")
	}
	if opts.TargetLocation == "" {
		opts.TargetLocation = "."
	}
	if opts.TargetDocument == nil {
		return nil, nil, errors.New("target document is required")
	}
	if opts.VirtualFS == nil {
		opts.VirtualFS = &system.FileSystem{}
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}

	uri := ref.GetURI()
	jp := ref.GetJSONPointer()

	// Use the extracted logic to resolve the absolute reference
	result, err := ResolveAbsoluteReference(ref, opts.TargetLocation)
	if err != nil {
		return nil, nil, err
	}

	absDocPath := result.AbsoluteReference
	finalClassification := result.Classification

	absRefWithJP := utils.BuildAbsoluteReference(absDocPath, string(jp))

	// Try and get the object from the cache as we should avoid recreating it if possible
	var obj *T
	co, coOK := opts.RootDocument.GetCachedReferencedObject(absRefWithJP)
	if coOK {
		obj, coOK = co.(*T)
	}

	// If the reference URI is empty the JSON pointer is relative to the target document
	if uri == "" {
		if coOK {
			return &ResolveResult[T]{
				Object:               obj,
				AbsoluteDocumentPath: absDocPath,
				AbsoluteReference:    Reference(absRefWithJP),
				ResolvedDocument:     opts.TargetDocument,
			}, nil, nil
		}

		obj, validationErrs, err := resolveAgainstDocument(ctx, jp, opts.TargetDocument, unmarshaler, opts)
		if err != nil {
			return nil, validationErrs, err
		}

		opts.RootDocument.InitCache()
		opts.RootDocument.StoreReferencedObjectInCache(absRefWithJP, obj)

		return &ResolveResult[T]{
			Object:               obj,
			AbsoluteDocumentPath: opts.TargetLocation,
			AbsoluteReference:    Reference(utils.BuildAbsoluteReference(opts.TargetLocation, string(jp))),
			ResolvedDocument:     opts.TargetDocument,
		}, validationErrs, nil
	} else if opts.DisableExternalRefs {
		return nil, nil, errors.New("external reference not allowed")
	}

	cd, cdOK := opts.RootDocument.GetCachedReferenceDocument(absDocPath)

	if coOK && cdOK {
		return &ResolveResult[T]{
			Object:               obj,
			AbsoluteDocumentPath: absDocPath,
			AbsoluteReference:    Reference(absRefWithJP),
			ResolvedDocument:     cd,
		}, nil, nil
	}

	// If we have a cached document, try and resolve against it
	if cdOK {
		obj, resolvedDoc, validationErrs, err := resolveAgainstData(ctx, absDocPath, bytes.NewReader(cd), jp, unmarshaler, opts)
		if err != nil {
			return nil, validationErrs, err
		}
		return &ResolveResult[T]{
			Object:               obj,
			AbsoluteDocumentPath: absDocPath,
			AbsoluteReference:    Reference(absRefWithJP),
			ResolvedDocument:     resolvedDoc,
		}, validationErrs, nil
	}

	// Otherwise resolve the reference
	switch finalClassification.Type {
	case utils.ReferenceTypeURL:
		obj, resolvedDoc, validationErrs, err := resolveAgainstURL(ctx, absDocPath, jp, unmarshaler, opts)
		if err != nil {
			return nil, validationErrs, err
		}

		opts.RootDocument.InitCache()
		opts.RootDocument.StoreReferencedObjectInCache(absRefWithJP, obj)

		return &ResolveResult[T]{
			Object:               obj,
			AbsoluteDocumentPath: absDocPath,
			AbsoluteReference:    Reference(absRefWithJP),
			ResolvedDocument:     resolvedDoc,
		}, validationErrs, nil
	case utils.ReferenceTypeFilePath:
		obj, resolvedDoc, validationErrs, err := resolveAgainstFilePath(ctx, absDocPath, jp, unmarshaler, opts)
		if err != nil {
			return nil, validationErrs, err
		}

		opts.RootDocument.InitCache()
		opts.RootDocument.StoreReferencedObjectInCache(absRefWithJP, obj)

		return &ResolveResult[T]{
			Object:               obj,
			AbsoluteDocumentPath: absDocPath,
			AbsoluteReference:    Reference(absRefWithJP),
			ResolvedDocument:     resolvedDoc,
		}, validationErrs, nil
	default:
		return nil, nil, fmt.Errorf("unsupported reference type: %d", finalClassification.Type)
	}
}

func resolveAgainstURL[T any](ctx context.Context, absRef string, jp jsonpointer.JSONPointer, unmarshaler Unmarshal[T], opts ResolveOptions) (*T, any, []error, error) {
	// TODO handle authentication
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, absRef, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	resp, err := opts.HTTPClient.Do(req)
	if err != nil || resp == nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()

	// Check if the response was successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	return resolveAgainstData(ctx, absRef, resp.Body, jp, unmarshaler, opts)
}

func resolveAgainstFilePath[T any](ctx context.Context, absRef string, jp jsonpointer.JSONPointer, unmarshaler Unmarshal[T], opts ResolveOptions) (*T, any, []error, error) {
	f, err := opts.VirtualFS.Open(absRef)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()

	return resolveAgainstData(ctx, absRef, f, jp, unmarshaler, opts)
}

func resolveAgainstDocument[T any](ctx context.Context, jp jsonpointer.JSONPointer, rootDocument any, unmarshaler Unmarshal[T], opts ResolveOptions) (*T, []error, error) {
	// If the JSON pointer is empty, the target is the root document
	if jp == "" {
		t, err := cast[T](rootDocument)
		return t, nil, err
	}

	target, err := jsonpointer.GetTarget(rootDocument, jp, jsonpointer.WithStructTags("key"))
	if err != nil {
		return nil, nil, err
	}

	if node, ok := target.(*yaml.Node); ok {
		return unmarshaler(ctx, node, opts.SkipValidation)
	}

	t, err := cast[T](target)
	return t, nil, err
}

func resolveAgainstData[T any](ctx context.Context, absRef string, reader io.Reader, jp jsonpointer.JSONPointer, unmarshaler Unmarshal[T], opts ResolveOptions) (*T, any, []error, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, nil, err
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, nil, nil, err
	}

	var target any

	// Handle empty JSON pointer case - if jp is empty, target the root node directly
	if jp == "" {
		target = &node
	} else {
		var jpErr error
		target, jpErr = jsonpointer.GetTarget(node, jp)
		if jpErr != nil {
			return nil, nil, nil, jpErr
		}
	}

	if target == nil {
		return nil, nil, nil, errors.New("target not found")
	}

	targetNode, ok := target.(*yaml.Node)
	if !ok {
		return nil, nil, nil, fmt.Errorf("expected *yaml.Node, got %T", target)
	}

	resolved, validationErrs, err := unmarshaler(ctx, targetNode, opts.SkipValidation)
	if err != nil {
		return nil, nil, validationErrs, err
	}

	if resolved == nil {
		return nil, nil, validationErrs, fmt.Errorf("nil %T returned from unmarshaler", target)
	}

	opts.RootDocument.InitCache()
	opts.RootDocument.StoreReferenceDocumentInCache(absRef, data)

	return resolved, &node, validationErrs, nil
}

func cast[T any](target any) (*T, error) {
	// First try direct pointer cast - if target is already *T
	if targetT, ok := target.(*T); ok {
		return targetT, nil
	}

	// Then try value cast - if target is T
	if targetT, ok := target.(T); ok {
		return &targetT, nil
	}

	value, _ := json.Marshal(target)

	var expectedType T
	return nil, fmt.Errorf("expected %T, got %T (value: %s)", expectedType, target, value)
}
