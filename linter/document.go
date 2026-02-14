package linter

import (
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
)

// DocumentInfo contains a document and its metadata for linting
type DocumentInfo[T any] struct {
	// Document is the parsed document to lint
	Document T

	// Location is the absolute location (URL or file path) of the document
	// This is used for resolving relative references
	Location string

	// Index contains an index of various nodes from the provided document
	Index *openapi.Index
}

// NewDocumentInfo creates a new DocumentInfo with the given document and location
func NewDocumentInfo[T any](doc T, location string) *DocumentInfo[T] {
	return &DocumentInfo[T]{
		Document: doc,
		Location: location,
	}
}

// NewDocumentInfoWithIndex creates a new DocumentInfo with a pre-computed index
func NewDocumentInfoWithIndex[T any](doc T, location string, index *openapi.Index) *DocumentInfo[T] {
	return &DocumentInfo[T]{
		Document: doc,
		Location: location,
		Index:    index,
	}
}

// LintOptions contains runtime options for linting
type LintOptions struct {
	// ResolveOptions contains options for reference resolution
	// If nil, default options will be used
	ResolveOptions *references.ResolveOptions

	// VersionFilter is the document version (e.g., "3.0.0", "3.1", "3.2.0")
	// If set, only rules that apply to this version will be run.
	// Rules with nil/empty Versions() apply to all versions.
	VersionFilter *string
}
