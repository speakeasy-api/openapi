package openapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/system"
)

// ResolveAllOptions represents the options available when resolving all references in an OpenAPI document.
type ResolveAllOptions struct {
	// OpenAPILocation is the location of the OpenAPI document to resolve.
	OpenAPILocation string
	// DisableExternalRefs when set to true will disable resolving of external references and return an error instead.
	DisableExternalRefs bool
	// VirtualFS is an optional virtual file system that will be used for any file based references. If not provided normal file system operations will be used.
	VirtualFS system.VirtualFS
	// HTTPClient is an optional HTTP client that will be used for any HTTP based references. If not provided http.DefaultClient will be used.
	HTTPClient system.Client
}

// ResolveAllReferences will resolve all references in the OpenAPI document, allowing them to be resolved and cached in a single operation.
func (o *OpenAPI) ResolveAllReferences(ctx context.Context, opts ResolveAllOptions) ([]error, error) {
	validationErrs := []error{}
	errs := []error{}

	rOpts := ResolveOptions{
		TargetLocation:      opts.OpenAPILocation,
		RootDocument:        o,
		DisableExternalRefs: opts.DisableExternalRefs,
		VirtualFS:           opts.VirtualFS,
		HTTPClient:          opts.HTTPClient,
	}

	resolve := func(r resolvable) error { //nolint:unparam
		vErrs, err := resolveAny(ctx, r, rOpts)
		validationErrs = append(validationErrs, vErrs...)
		if err != nil {
			errs = append(errs, err)
		}

		return nil
	}

	for item := range Walk(ctx, o) {
		_ = item.Match(Matcher{
			ReferencedPathItem: func(rpi *ReferencedPathItem) error {
				return resolve(rpi)
			},
			ReferencedParameter: func(rp *ReferencedParameter) error {
				return resolve(rp)
			},
			ReferencedHeader: func(rh *ReferencedHeader) error {
				return resolve(rh)
			},
			ReferencedRequestBody: func(rrb *ReferencedRequestBody) error {
				return resolve(rrb)
			},
			ReferencedExample: func(re *ReferencedExample) error {
				return resolve(re)
			},
			ReferencedResponse: func(rr *ReferencedResponse) error {
				return resolve(rr)
			},
			ReferencedLink: func(rl *ReferencedLink) error {
				return resolve(rl)
			},
			ReferencedCallback: func(rc *ReferencedCallback) error {
				return resolve(rc)
			},
			ReferencedSecurityScheme: func(rss *ReferencedSecurityScheme) error {
				return resolve(rss)
			},
			Schema: func(j *oas3.JSONSchema[oas3.Referenceable]) error {
				return resolve(j)
			},
		})
	}

	return validationErrs, errors.Join(errs...)
}

type resolvable interface {
	IsReference() bool
	IsResolved() bool
}

func resolveAny(ctx context.Context, resolvable resolvable, opts ResolveOptions) ([]error, error) {
	if !resolvable.IsReference() || resolvable.IsResolved() {
		return nil, nil
	}

	var vErrs []error
	var err error

	switch r := resolvable.(type) {
	case *ReferencedPathItem:
		vErrs, err = r.Resolve(ctx, opts)
	case *ReferencedParameter:
		vErrs, err = r.Resolve(ctx, opts)
	case *ReferencedHeader:
		vErrs, err = r.Resolve(ctx, opts)
	case *ReferencedRequestBody:
		vErrs, err = r.Resolve(ctx, opts)
	case *ReferencedResponse:
		vErrs, err = r.Resolve(ctx, opts)
	case *ReferencedLink:
		vErrs, err = r.Resolve(ctx, opts)
	case *ReferencedSecurityScheme:
		vErrs, err = r.Resolve(ctx, opts)
	case *ReferencedExample:
		vErrs, err = r.Resolve(ctx, opts)
	case *ReferencedCallback:
		vErrs, err = r.Resolve(ctx, opts)
	case *oas3.JSONSchema[oas3.Referenceable]:
		vErrs, err = r.Resolve(ctx, oas3.ResolveOptions{
			TargetLocation:      opts.TargetLocation,
			RootDocument:        opts.RootDocument,
			DisableExternalRefs: opts.DisableExternalRefs,
			VirtualFS:           opts.VirtualFS,
			HTTPClient:          opts.HTTPClient,
		})
	default:
		panic(fmt.Sprintf("unsupported resolvable type: %T", resolvable))
	}

	return vErrs, err
}

// ExtractMethodAndPath extracts the HTTP method and path from location context when walking operations.
// This utility function is designed to be used alongside Walk operations to determine the HTTP method
// and path that an operation relates to.
//
// The function traverses the location context in reverse order to find:
// - The path from a "Paths" parent (e.g., "/users/{id}")
// - The HTTP method from a "PathItem" parent (e.g., "get", "post", "put", etc.)
//
// Returns empty strings if the locations don't represent a valid operation context or if
// the location hierarchy doesn't match the expected OpenAPI structure.
//
// Example usage:
//
//	for item := range openapi.Walk(ctx, doc) {
//		err := item.Match(openapi.Matcher{
//			Operation: func(op *openapi.Operation) error {
//				method, path := openapi.ExtractMethodAndPath(item.Location)
//				fmt.Printf("Found operation: %s %s\n", method, path)
//				return nil
//			},
//		})
//	}
func ExtractMethodAndPath(locations Locations) (string, string) {
	if len(locations) == 0 {
		return "", ""
	}

	var method, path string

	for i := len(locations) - 1; i >= 0; i-- {
		switch GetParentType(locations[i]) {
		case "Paths":
			path = pointer.Value(locations[i].ParentKey)
		case "PathItem":
			method = pointer.Value(locations[i].ParentKey)
		case "OpenAPI":
		default:
			// Matched something unexpected so not likely an operation in paths
			return "", ""
		}
	}

	return method, path
}

// GetParentType determines the type of the parent object in a location context.
// This utility function is used to identify the parent type when traversing OpenAPI
// document structures during walking operations.
//
// Returns one of the following strings based on the parent type:
// - "Paths" for *openapi.Paths
// - "PathItem" for *openapi.ReferencedPathItem
// - "OpenAPI" for *openapi.OpenAPI
// - "Unknown" for any other type
//
// This function is primarily used internally by ExtractMethodAndPath but is exposed
// as a public utility for advanced use cases where users need to inspect the
// parent type hierarchy during document traversal.
func GetParentType(location LocationContext) string {
	parentType := ""
	_ = location.ParentMatchFunc(Matcher{
		Any: func(a any) error {
			switch a.(type) {
			case *Paths:
				parentType = "Paths"
			case *ReferencedPathItem:
				parentType = "PathItem"
			case *OpenAPI:
				parentType = "OpenAPI"
			default:
				parentType = "Unknown"
			}
			return nil
		},
	})
	return parentType
}
