package linter

import (
	"errors"

	baseLinter "github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

// Include index diagnostics for caller-supplied indexes too. A caller may
// already have forwarded the same error objects, so do not append them twice.
func appendIndexDiagnostics(existing []error, idx *openapi.Index) []error {
	if idx == nil || !idx.HasErrors() {
		return existing
	}
	seen := make(map[*validation.Error]struct{})
	for _, err := range existing {
		var finding *validation.Error
		if errors.As(err, &finding) {
			seen[finding] = struct{}{}
		}
	}
	combined := append([]error(nil), existing...)
	for _, errs := range [][]error{idx.GetValidationErrors(), idx.GetResolutionErrors(), idx.GetCircularReferenceErrors()} {
		for _, err := range errs {
			var finding *validation.Error
			if errors.As(err, &finding) {
				if _, duplicate := seen[finding]; duplicate {
					continue
				}
				seen[finding] = struct{}{}
			}
			combined = append(combined, err)
		}
	}
	return combined
}

type externalDiagnosticKey struct {
	node     *yaml.Node
	rule     string
	severity validation.Severity
	location string
	message  string
}

// Attribute after rule execution, when cached source documents are complete.
// Copy changed errors: callers and rule caches may retain the original errors.
func attributeExternalDiagnostics(output *baseLinter.Output, docInfo *baseLinter.DocumentInfo[*openapi.OpenAPI]) {
	if output == nil || docInfo == nil || docInfo.Index == nil {
		return
	}
	lookupSource := docInfo.Index.NewDocumentLocationLookup()
	seen := make(map[externalDiagnosticKey]struct{})
	results := make([]error, 0, len(output.Results))
	for _, err := range output.Results {
		var finding *validation.Error
		if !errors.As(err, &finding) {
			results = append(results, err)
			continue
		}
		sourceLocation := lookupSource(finding.Node)
		location := finding.DocumentLocation
		if location == "" {
			location = sourceLocation
		}
		external := sourceLocation != "" || (location != "" && location != docInfo.Location)
		if location != finding.DocumentLocation || (external && finding.Fix != nil) {
			updated := *finding
			updated.DocumentLocation = location
			if external {
				// The current fix engine persists only the main document. An
				// external node edit would otherwise be reported as saved but lost.
				updated.Fix = nil
			}
			finding = &updated
			err = finding
		}
		if sourceLocation != "" {
			message := ""
			if finding.UnderlyingError != nil {
				message = finding.UnderlyingError.Error()
			}
			key := externalDiagnosticKey{finding.Node, finding.Rule, finding.Severity, finding.DocumentLocation, message}
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
		}
		results = append(results, err)
	}
	output.Results = results
	validation.SortValidationErrors(output.Results)
}
