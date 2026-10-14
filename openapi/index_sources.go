package openapi

import (
	"errors"
	"iter"

	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

// Index-generated errors can initially carry the immediate reference's filename
// even when an alias chain resolves to another document. Correct those owned
// diagnostics from node identity once traversal and resolution are complete.
func (i *Index) attributeSourceDiagnostics() {
	i.validationErrs = i.attributeSourceErrors(i.validationErrs)
	i.resolutionErrs = i.attributeSourceErrors(i.resolutionErrs)
	i.circularErrs = i.attributeSourceErrors(i.circularErrs)
}

func (i *Index) attributeSourceErrors(errs []error) []error {
	if len(errs) == 0 {
		return errs
	}
	attributed := append([]error(nil), errs...)
	for index, err := range errs {
		var finding *validation.Error
		if !errors.As(err, &finding) {
			continue
		}
		location := i.GetDocumentLocation(finding.Node)
		if location == "" || location == finding.DocumentLocation {
			continue
		}
		updated := *finding
		updated.DocumentLocation = location
		attributed[index] = &updated
	}
	return attributed
}

// NewDocumentLocationLookup creates an independent lookup of the currently
// cached source documents. Use this after custom rules that may resolve more
// documents, without mutating a shared index's immutable source snapshot.
func (i *Index) NewDocumentLocationLookup() func(*yaml.Node) string {
	if i == nil {
		return func(*yaml.Node) string { return "" }
	}
	snapshot := &Index{Doc: i.Doc, resolveOpts: i.resolveOpts}
	return snapshot.GetDocumentLocation
}

// GetDocumentLocation returns the source file or URL of a node in the parsed
// external documents retained by resolution. Empty means a node in the original
// root model or an unknown node. A separately parsed copy of the main file keeps
// its filename, since it is not part of the root model the fix engine persists.
// It never loads or resolves a document.
//
// Like the other index lookups, the mapping assumes the index and its source
// documents are not modified after construction. Concurrent lookups are safe.
func (i *Index) GetDocumentLocation(node *yaml.Node) string {
	if i == nil || i.Doc == nil || node == nil {
		return ""
	}
	i.documentLocationsOnce.Do(func() {
		i.documentLocations = make(map[*yaml.Node]string)
		mainRoot := i.Doc.GetRootNode()
		var documents interface{ CachedExternalDocuments() iter.Seq2[string, any] } = i.Doc
		if i.resolveOpts.RootDocument != nil {
			var ok bool
			documents, ok = i.resolveOpts.RootDocument.(interface{ CachedExternalDocuments() iter.Seq2[string, any] })
			if !ok {
				return
			}
		}
		for location, cached := range documents.CachedExternalDocuments() {
			root, ok := cached.(*yaml.Node)
			if !ok {
				continue
			}
			pending := []*yaml.Node{root}
			for len(pending) > 0 {
				current := pending[len(pending)-1]
				pending = pending[:len(pending)-1]
				if current == nil || current == mainRoot {
					continue
				}
				if _, visited := i.documentLocations[current]; visited {
					continue
				}
				i.documentLocations[current] = location
				pending = append(pending, current.Content...)
				if current.Alias != nil {
					pending = append(pending, current.Alias)
				}
			}
		}
	})
	return i.documentLocations[node]
}
