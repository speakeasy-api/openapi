package arazzo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/system"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

// LoadSourcesOptions are the options for loading the documents referenced by an Arazzo document's sourceDescriptions.
type LoadSourcesOptions struct {
	// DocumentLocation is the location (file path or URL) of the Arazzo document. Relative source description URLs
	// are resolved against the document's `$self` if set, otherwise against this location. If empty, relative URLs
	// are resolved against the current working directory.
	DocumentLocation string
	// VirtualFS is an optional virtual file system used to load file based source descriptions. If not provided the
	// normal file system is used.
	VirtualFS system.VirtualFS
	// HTTPClient is an optional HTTP client used to load source descriptions served over HTTP(S). If not provided
	// a client with a 30 second timeout is used.
	HTTPClient system.Client
}

var defaultSourcesHTTPClient = &http.Client{Timeout: 30 * time.Second}

// SourceDocument is a loaded document referenced by a source description.
type SourceDocument struct {
	// Name is the name of the source description the document was loaded for.
	Name string
	// Location is the resolved location (file path or URL) the document was loaded from.
	Location string
	// Type is the type of the loaded document, detected from its content.
	Type SourceDescriptionType
	// OpenAPI is the loaded document if it is an OpenAPI document, with its references resolved.
	OpenAPI *openapi.OpenAPI
	// Arazzo is the loaded document if it is an Arazzo document.
	Arazzo *Arazzo
	// AsyncAPI is the loaded document if it is an AsyncAPI 3.x document, with its references resolved.
	AsyncAPI *asyncapi.AsyncAPI
}

// SourceDocuments are the documents loaded for an Arazzo document's sourceDescriptions, keyed by source description name.
type SourceDocuments struct {
	documents map[string]*SourceDocument
}

// Get returns the loaded document for the named source description, or nil if it wasn't loaded.
func (s *SourceDocuments) Get(name string) *SourceDocument {
	if s == nil {
		return nil
	}
	return s.documents[name]
}

// LoadSourceDescriptions loads the OpenAPI, AsyncAPI and Arazzo documents referenced by the Arazzo document's
// sourceDescriptions so the document can be validated against them with ValidateSources.
//
// Problems loading a source are returned as warnings rather than errors, as a source may be temporarily
// unavailable without the Arazzo document being invalid. This includes AsyncAPI documents of a version the asyncapi
// package doesn't support, such as 2.x. A source description declaring a type that doesn't match the loaded document
// is reported as an error.
func LoadSourceDescriptions(ctx context.Context, a *Arazzo, opts LoadSourcesOptions) (*SourceDocuments, []error) {
	sources := &SourceDocuments{documents: map[string]*SourceDocument{}}
	if a == nil {
		return sources, nil
	}

	if opts.VirtualFS == nil {
		opts.VirtualFS = &system.FileSystem{}
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = defaultSourcesHTTPClient
	}

	errs := []error{}

	base := opts.DocumentLocation
	if a.Self != nil && *a.Self != "" {
		self, err := resolveSourceLocation(opts.DocumentLocation, *a.Self)
		if err != nil {
			core := a.GetCore()
			errs = append(errs, validation.NewValueError(validation.SeverityWarning, validation.RuleValidationInvalidReference, fmt.Errorf("`$self` can't be used to resolve source descriptions: %w", err), core, core.Self))
		} else {
			base = self
		}
	}

	for _, sd := range a.SourceDescriptions {
		if sd == nil || sd.Name == "" || sd.URL == "" {
			continue
		}

		doc, sdErrs := loadSourceDescription(ctx, sd, base, opts)
		errs = append(errs, sdErrs...)
		if doc != nil {
			sources.documents[sd.Name] = doc
		}
	}

	return sources, errs
}

func loadSourceDescription(ctx context.Context, sd *SourceDescription, base string, opts LoadSourcesOptions) (*SourceDocument, []error) {
	core := sd.GetCore()

	warn := func(err error) []error {
		return []error{validation.NewValueError(validation.SeverityWarning, validation.RuleValidationInvalidReference, err, core, core.URL)}
	}

	location, err := resolveSourceLocation(base, sd.URL)
	if err != nil {
		return nil, warn(fmt.Errorf("sourceDescription `%s` url can't be resolved: %w", sd.Name, err))
	}

	data, err := fetchSource(ctx, location, opts)
	if err != nil {
		return nil, warn(fmt.Errorf("sourceDescription `%s` could not be loaded from `%s`: %w", sd.Name, location, err))
	}

	detected := detectSourceType(data)
	switch {
	case detected == "":
		return nil, warn(fmt.Errorf("sourceDescription `%s` at `%s` is not an OpenAPI 3.x, AsyncAPI or Arazzo document", sd.Name, location))
	case sd.Type != "" && sd.Type != detected:
		return nil, []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationTypeMismatch, fmt.Errorf("sourceDescription `%s` is declared as type `%s` but `%s` is an %s document", sd.Name, sd.Type, location, detected), core, core.Type)}
	}

	doc := &SourceDocument{Name: sd.Name, Location: location, Type: detected}

	switch detected {
	case SourceDescriptionTypeArazzo:
		doc.Arazzo, _, err = Unmarshal(ctx, bytes.NewReader(data), WithSkipValidation())
	case SourceDescriptionTypeAsyncAPI:
		doc.AsyncAPI, _, err = asyncapi.Unmarshal(ctx, bytes.NewReader(data), asyncapi.WithSkipValidation())
	default:
		doc.OpenAPI, _, err = openapi.Unmarshal(ctx, bytes.NewReader(data), openapi.WithSkipValidation())
	}
	if err != nil {
		return nil, warn(fmt.Errorf("sourceDescription `%s` at `%s` could not be parsed: %w", sd.Name, location, err))
	}

	switch {
	case doc.AsyncAPI != nil:
		return loadAsyncAPISource(ctx, doc, opts, warn)
	case doc.OpenAPI == nil:
		return doc, nil
	}

	// Unresolved references only reduce how much can be checked, so the source is still usable.
	if _, err := doc.OpenAPI.ResolveAllReferences(ctx, openapi.ResolveAllOptions{
		OpenAPILocation: location,
		VirtualFS:       opts.VirtualFS,
		HTTPClient:      opts.HTTPClient,
	}); err != nil {
		return doc, warn(fmt.Errorf("sourceDescription `%s` has references that could not be resolved, some checks may be skipped: %w", sd.Name, err))
	}

	return doc, nil
}

// resolveSourceLocation resolves a reference against a base location. Absolute URLs are used as is, as are
// absolute file paths unless the base is a URL (where they are URI paths on the base's host).
func resolveSourceLocation(base, ref string) (string, error) {
	if utils.IsURL(ref) || (filepath.IsAbs(ref) && !utils.IsURL(base)) {
		return ref, nil
	}
	return utils.JoinReference(base, ref)
}

// loadAsyncAPISource checks the AsyncAPI version is supported and resolves the document's references.
func loadAsyncAPISource(ctx context.Context, doc *SourceDocument, opts LoadSourcesOptions, warn func(error) []error) (*SourceDocument, []error) {
	if version := doc.AsyncAPI.GetAsyncAPI(); !asyncapi.IsSupportedVersion(version) {
		return nil, warn(fmt.Errorf("sourceDescription `%s` is an AsyncAPI %s document, only AsyncAPI %s are supported so steps targeting it are not checked", doc.Name, version, strings.Join(asyncapi.SupportedMinorVersions(), " and ")))
	}

	// Unresolved references only reduce how much can be checked, so the source is still usable.
	resolveErrs, err := doc.AsyncAPI.ResolveAllReferences(ctx, asyncapi.ResolveAllOptions{
		AsyncAPILocation: doc.Location,
		VirtualFS:        opts.VirtualFS,
		HTTPClient:       opts.HTTPClient,
	})
	if err == nil {
		err = firstResolutionError(resolveErrs)
	}
	if err != nil {
		return doc, warn(fmt.Errorf("sourceDescription `%s` has references that could not be resolved, some checks may be skipped: %w", doc.Name, err))
	}

	return doc, nil
}

// firstResolutionError returns the first error from resolving an AsyncAPI document's references. Warnings and hints,
// such as schema formats that aren't validated, don't stop anything being checked so they are ignored.
func firstResolutionError(errs []error) error {
	for _, err := range errs {
		var vErr *validation.Error
		if errors.As(err, &vErr) && vErr.Severity != validation.SeverityError {
			continue
		}
		return err
	}
	return nil
}

func fetchSource(ctx context.Context, location string, opts LoadSourcesOptions) ([]byte, error) {
	if !utils.IsURL(location) {
		f, err := opts.VirtualFS.Open(location)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}

	resp, err := opts.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("no response received")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("unexpected HTTP status " + resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// detectSourceType detects the type of a source document from its top level version field.
func detectSourceType(data []byte) SourceDescriptionType {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil || len(root.Content) == 0 {
		return ""
	}

	node := root.Content[0]
	if node.Kind != yaml.MappingNode {
		return ""
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case "openapi":
			return SourceDescriptionTypeOpenAPI
		case "arazzo":
			return SourceDescriptionTypeArazzo
		case "asyncapi":
			return SourceDescriptionTypeAsyncAPI
		}
	}

	return ""
}
