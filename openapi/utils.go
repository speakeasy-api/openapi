package openapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
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
